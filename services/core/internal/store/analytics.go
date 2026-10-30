package store

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"math"
	"time"

	"github.com/jackc/pgx/v5"
	"tutor-platform/services/core/internal/apperr"
	"tutor-platform/services/core/internal/domain"
	"tutor-platform/services/core/internal/platform"
)

// Tutors see only their teaching context, not another tutor's work with the same child.
const analyticsRunFrom = ` from learning_attempts la
 join lesson_versions v on v.id=la.version_id join courses c on c.id=v.course_id
 left join learning_review_sessions rs on rs.learning_attempt_id=la.id
 left join learning_attempts origin on origin.id=rs.source_run_id
 left join assignments a on a.id=coalesce(la.assignment_id,origin.assignment_id)
 left join study_sessions ss on ss.learning_attempt_id=coalesce(rs.source_run_id,la.id)
 left join exam_specifications es on es.id=ss.specification_id
 left join bank_collections bc on bc.version_id=v.id `
const analyticsRunScope = `la.student_id=$1 and c.organization_id=$2 and
 ($3='student' or ($3='tutor' and exists(select 1 from relations rel where rel.student_id=la.student_id and rel.tutor_id=$4 and rel.organization_id=$2 and rel.status='active')
 and (a.tutor_id=$4 or es.author_id=$4 or (a.id is null and ss.id is null and c.author_id=$4))))`
const analyticsTopicFrom = ` join exercises e on e.version_id=v.id
 join topics t on t.id=v.topic_id
 left join lateral (select p.item from jsonb_array_elements(coalesce(bc.provenance,'[]'::jsonb)) with ordinality p(item,n) where p.n=e.position+1) provenance on true `
const analyticsTopicName = `coalesce(provenance.item->>'topic',t.title)`
const analyticsSubtopicName = `coalesce(provenance.item->>'subtopic','')`
const analyticsTopicKey = `md5(jsonb_build_array(` + analyticsTopicName + `,` + analyticsSubtopicName + `)::text)`
const analyticsFactsHead = `with facts as (
 select ea.id,ea.exercise_id,ea.learning_attempt_id,ea.status,ea.created_at,ea.attempt_number,
 ` + analyticsTopicName + ` topic,` + analyticsSubtopicName + ` subtopic,` + analyticsTopicKey + ` topic_key,
 row_number() over(partition by ea.learning_attempt_id,ea.exercise_id order by ea.attempt_number desc) rank,
 (select count(*) from learning_hints h where h.learning_attempt_id=la.id and h.exercise_id=e.id and h.revealed_at >= $5 and h.revealed_at < $6) hints
 ` + analyticsRunFrom + analyticsTopicFrom + ` join exercise_attempts ea on ea.learning_attempt_id=la.id and ea.exercise_id=e.id
 where `
const analyticsFactsTail = ` and (la.mode<>'exam' or la.completed_at is not null) and ea.created_at >= $5 and ea.created_at < $6
), latest as(select * from facts where rank=1) `
const analyticsFacts = analyticsFactsHead + analyticsRunScope + analyticsFactsTail
const analyticsMetricJSON = `jsonb_build_object(
 'attempts',count(*),'graded',count(*) filter(where rank=1 and status in ('correct','incorrect')),
 'correct',count(*) filter(where rank=1 and status='correct'),'incorrect',count(*) filter(where rank=1 and status='incorrect'),
 'pending',count(*) filter(where rank=1 and status='needs_review'),'mistakes',count(*) filter(where status='incorrect'),
 'hints',coalesce(sum(hints) filter(where rank=1),0),'hinted',count(*) filter(where rank=1 and hints>0))`

func analyticsArgs(actor domain.User, studentID string, f domain.AnalyticsFilter) []any {
	return []any{studentID, actor.OrganizationID, string(actor.Role), actor.ID, f.From, f.To.AddDate(0, 0, 1)}
}
func ValidateAnalyticsFilter(f domain.AnalyticsFilter) error {
	if f.From.IsZero() || f.To.Before(f.From) || f.To.Sub(f.From) > 365*24*time.Hour || f.TopicOffset < 0 || f.ExamOffset < 0 || f.TopicOffset > 100000 || f.ExamOffset > 100000 {
		return apperr.ErrBadRequest
	}
	return nil
}
func (s *Store) analyticsAccess(ctx context.Context, actor domain.User, studentID string) (domain.AnalyticsStudent, error) {
	return analyticsAccess(ctx, s.db, actor, studentID)
}
func analyticsAccess(ctx context.Context, q catalogQuerier, actor domain.User, studentID string) (domain.AnalyticsStudent, error) {
	if actor.Role != domain.RoleStudent && actor.Role != domain.RoleTutor {
		return domain.AnalyticsStudent{}, apperr.ErrForbidden
	}
	if !platform.ValidUUID(studentID) {
		return domain.AnalyticsStudent{}, apperr.ErrBadRequest
	}
	if actor.Role == domain.RoleStudent && actor.ID != studentID {
		return domain.AnalyticsStudent{}, apperr.ErrNotFound
	}
	return catalogJSON[domain.AnalyticsStudent](ctx, q, `select jsonb_build_object('id',u.id,'display_name',u.display_name) from users u where u.id=$1 and u.organization_id=$2 and u.role='student' and ($3='student' or exists(select 1 from relations r where r.student_id=u.id and r.tutor_id=$4 and r.organization_id=$2 and r.status='active'))`, studentID, actor.OrganizationID, string(actor.Role), actor.ID)
}
func (s *Store) AnalyticsStudents(ctx context.Context, actor domain.User, f domain.CatalogFilter) (domain.CatalogPage[domain.AnalyticsStudent], error) {
	if actor.Role != domain.RoleTutor {
		return domain.CatalogPage[domain.AnalyticsStudent]{}, apperr.ErrForbidden
	}
	if f.Limit < 1 || f.Limit > 100 || f.Offset < 0 || f.Offset > 100000 {
		return domain.CatalogPage[domain.AnalyticsStudent]{}, apperr.ErrBadRequest
	}
	return catalogJSON[domain.CatalogPage[domain.AnalyticsStudent]](ctx, s.db, `with students as(select u.id,u.display_name from users u join relations r on r.student_id=u.id where r.organization_id=$1 and u.organization_id=$1 and r.tutor_id=$2 and r.status='active' and u.role='student') select jsonb_build_object('items',coalesce((select jsonb_agg(jsonb_build_object('id',id,'display_name',display_name) order by display_name,id) from(select * from students order by display_name,id limit $3 offset $4)p),'[]'::jsonb),'total',(select count(*) from students),'limit',$3::int,'offset',$4::int)`, actor.OrganizationID, actor.ID, f.Limit, f.Offset)
}
func analyticsPercent(n, d int) *float64 {
	if d == 0 {
		return nil
	}
	v := math.Round(float64(n)*1000/float64(d)) / 10
	return &v
}
func analyticsMetric(m *domain.AnalyticsMetric) {
	m.Accuracy = analyticsPercent(m.Correct, m.Graded)
	m.HintRate = analyticsPercent(m.Hinted, m.Graded+m.Pending)
}
func analyticsLevel(graded int, accuracy *float64) string {
	if graded < 5 || accuracy == nil {
		return "insufficient"
	}
	if *accuracy < 70 {
		return "practice"
	}
	if *accuracy < 85 {
		return "developing"
	}
	return "secure"
}
func (s *Store) LearningAnalytics(ctx context.Context, actor domain.User, studentID string, f domain.AnalyticsFilter) (domain.AnalyticsReport, error) {
	out := domain.AnalyticsReport{From: f.From.Format("2006-01-02"), To: f.To.Format("2006-01-02")}
	if err := ValidateAnalyticsFilter(f); err != nil {
		return out, err
	}
	if _, err := s.analyticsAccess(ctx, actor, studentID); err != nil {
		return out, err
	}
	if err := s.expireStudySessions(ctx, domain.User{ID: studentID}); err != nil {
		return out, err
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx)
	student, err := analyticsAccess(ctx, tx, actor, studentID)
	if err != nil {
		return out, err
	}
	out.Student = student
	args := analyticsArgs(actor, studentID, f)
	if err = tx.QueryRow(ctx, `select clock_timestamp()`).Scan(&out.GeneratedAt); err != nil {
		return out, err
	}
	out.Summary, err = catalogJSON[domain.AnalyticsMetric](ctx, tx, analyticsFacts+`select `+analyticsMetricJSON+` from facts`, args...)
	if err != nil {
		return out, err
	}
	analyticsMetric(&out.Summary)
	out.Topics, err = catalogJSON[domain.CatalogPage[domain.AnalyticsTopic]](ctx, tx, analyticsFacts+`, grouped as(select topic_key,topic,subtopic,`+analyticsMetricJSON+` metric,
 count(*) filter(where status in ('correct','incorrect') and created_at < $5::timestamptz+($6::timestamptz-$5::timestamptz)/2) previous_graded,
 count(*) filter(where status='correct' and created_at < $5::timestamptz+($6::timestamptz-$5::timestamptz)/2) previous_correct,
 count(*) filter(where status in ('correct','incorrect') and created_at >= $5::timestamptz+($6::timestamptz-$5::timestamptz)/2) recent_graded,
 count(*) filter(where status='correct' and created_at >= $5::timestamptz+($6::timestamptz-$5::timestamptz)/2) recent_correct from facts group by topic_key,topic,subtopic)
 select jsonb_build_object('items',coalesce((select jsonb_agg(item order by topic,subtopic,topic_key) from(select topic,subtopic,topic_key,metric||jsonb_build_object('key',topic_key,'topic',topic,'subtopic',subtopic,'previous_graded',previous_graded,'recent_graded',recent_graded,'change',case when previous_graded>=5 and recent_graded>=5 then round(100.0*recent_correct/recent_graded-100.0*previous_correct/previous_graded,1) else null end) item from grouped order by topic,subtopic,topic_key limit 20 offset $7)p),'[]'::jsonb),'total',(select count(*) from grouped),'limit',20,'offset',$7::int)`, append(args, f.TopicOffset)...)
	if err != nil {
		return out, err
	}
	for i := range out.Topics.Items {
		t := &out.Topics.Items[i]
		analyticsMetric(&t.AnalyticsMetric)
		t.Level = analyticsLevel(t.Graded, t.Accuracy)
	}
	out.Timeline, err = catalogJSON[[]domain.AnalyticsDay](ctx, tx, analyticsFacts+`select coalesce(jsonb_agg(item order by bucket),'[]'::jsonb) from(select (created_at at time zone 'UTC')::date bucket,jsonb_build_object('day',(created_at at time zone 'UTC')::date,'attempts',count(*),'graded',count(*) filter(where status in ('correct','incorrect')),'correct',count(*) filter(where status='correct')) item from facts group by bucket)d`, args...)
	if err != nil {
		return out, err
	}
	for i := range out.Timeline {
		d := &out.Timeline[i]
		d.Accuracy = analyticsPercent(d.Correct, d.Graded)
	}
	out.Homework, err = catalogJSON[domain.AnalyticsHomework](ctx, tx, `select jsonb_build_object('total',count(*),'done',count(*) filter(where status='done'),'submitted',count(*) filter(where status='submitted'),'overdue',count(*) filter(where status not in ('done','submitted') and due_at<clock_timestamp())) from assignments a where a.student_id=$1 and a.organization_id=$2 and ($3='student' or a.tutor_id=$4) and a.status<>'archived' and a.created_at >= $5 and a.created_at < $6`, args...)
	if err != nil {
		return out, err
	}
	out.Exams, err = catalogJSON[domain.CatalogPage[domain.AnalyticsExam]](ctx, tx, `with eligible as(select ss.id,la.started_at,jsonb_build_object('id',ss.id,'title',es.title,'started_at',la.started_at,'completed_at',la.completed_at,'points',case when la.completed_at is null then null else (select coalesce(sum(case when last.status='correct' then (ss.exercise_points->>last.exercise_key::text)::int else 0 end),0) from(select distinct on(e.exercise_key) e.exercise_key,ea.status from exercise_attempts ea join exercises e on e.id=ea.exercise_id where ea.learning_attempt_id=la.id order by e.exercise_key,ea.attempt_number desc)last) end,'max_points',(select sum(value::int) from jsonb_each_text(ss.exercise_points)),'pending',case when la.completed_at is null then 0 else(select count(*) from(select distinct on(exercise_id) status from exercise_attempts where learning_attempt_id=la.id order by exercise_id,attempt_number desc)last where status='needs_review')end) item`+analyticsRunFrom+`where `+analyticsRunScope+` and la.mode='exam' and ss.id is not null and la.started_at >= $5 and la.started_at < $6) select jsonb_build_object('items',coalesce((select jsonb_agg(item order by started_at desc,id) from(select * from eligible order by started_at desc,id limit 20 offset $7)p),'[]'::jsonb),'total',(select count(*) from eligible),'limit',20,'offset',$7::int)`, append(args, f.ExamOffset)...)
	if err != nil {
		return out, err
	}
	out.Recommendations, err = catalogJSON[[]domain.AnalyticsRecommendation](ctx, tx, analyticsFacts+`, weak as(select topic_key,topic,subtopic,count(*) filter(where status in ('correct','incorrect')) sample,count(*) filter(where status='incorrect') errors from latest group by topic_key,topic,subtopic having count(*) filter(where status='incorrect')>0), candidates as(select w.*,la.id run_id,la.lesson_id,case when rs.learning_attempt_id is not null then '/tasks/review/'||la.id::text when a.id is not null then '/tasks/'||a.id::text when ss.id is not null then '/tasks/study/'||ss.id::text else '/tasks/library/'||la.lesson_id::text end href,row_number()over(partition by w.topic_key order by la.started_at desc,la.id) rank from weak w join latest f on f.topic_key=w.topic_key and f.status='incorrect' join learning_attempts la on la.id=f.learning_attempt_id left join assignments a on a.id=la.assignment_id left join study_sessions ss on ss.learning_attempt_id=la.id left join learning_review_sessions rs on rs.learning_attempt_id=la.id)
 select coalesce(jsonb_agg(item order by errors desc,topic_key),'[]'::jsonb) from(select topic_key,errors,jsonb_build_object('topic_key',topic_key,'topic',topic,'subtopic',subtopic,'run_id',run_id,'lesson_id',lesson_id,'href',href,'sample',sample,'errors',errors,'exercises',
 (select coalesce(jsonb_agg(detail order by position,id),'[]'::jsonb) from(select e.position,e.id,jsonb_build_object('exercise_id',e.exercise_key,'prompt',e.prompt,'attempts',(select count(*) from facts all_f where all_f.exercise_id=e.id and all_f.learning_attempt_id=candidates.run_id),'mistakes',(select count(*) from facts all_f where all_f.exercise_id=e.id and all_f.learning_attempt_id=candidates.run_id and all_f.status='incorrect')) detail from latest f join exercises e on e.id=f.exercise_id where f.learning_attempt_id=candidates.run_id and f.topic_key=candidates.topic_key and f.status='incorrect' order by e.position,e.id limit 20)details),
 'reason',case when sample<5 then 'few_observations' when 100.0*(sample-errors)/sample<70 then 'weak_topic' else 'recent_errors' end) item from candidates where rank=1 order by errors desc,topic_key limit 5)p`, args...)
	if err != nil {
		return out, err
	}
	return out, tx.Commit(ctx)
}

func (s *Store) RepeatAnalyticsTopic(ctx context.Context, actor domain.User, studentID string, in domain.AnalyticsRepeatInput) (domain.Assignment, error) {
	var out domain.Assignment
	if actor.Role != domain.RoleTutor {
		return out, apperr.ErrForbidden
	}
	if !platform.ValidUUID(in.RequestID) || !platform.ValidUUID(in.RunID) || len(in.TopicKey) != 32 {
		return out, apperr.ErrBadRequest
	}
	if _, err := hex.DecodeString(in.TopicKey); err != nil {
		return out, apperr.ErrBadRequest
	}
	if _, err := s.analyticsAccess(ctx, actor, studentID); err != nil {
		return out, err
	}
	payload, _ := json.Marshal(map[string]any{"analytics_student": studentID, "input": in})
	if old, err := s.materialReplay(ctx, actor, in.RequestID, payload); err != nil || old.ID != "" {
		return old, err
	}
	var versionID string
	var keys []string
	err := s.db.QueryRow(ctx, `select la.version_id::text,array_agg(e.exercise_key::text order by e.position,e.id)`+analyticsRunFrom+analyticsTopicFrom+` where `+analyticsRunScope+` and la.id=$5 and `+analyticsTopicKey+`=$6 and (la.mode<>'exam' or la.completed_at is not null) and (select ea.status from exercise_attempts ea where ea.learning_attempt_id=la.id and ea.exercise_id=e.id order by ea.attempt_number desc limit 1)='incorrect' group by la.version_id`, studentID, actor.OrganizationID, string(actor.Role), actor.ID, in.RunID, in.TopicKey).Scan(&versionID, &keys)
	if err != nil {
		return out, catalogError(err)
	}
	if len(keys) > 200 {
		keys = keys[:200]
	}
	v, err := materialVersion(ctx, s.db, versionID, keys)
	if err != nil {
		return out, err
	}
	v.Title = "Повторение: " + v.Title
	return s.insertMaterialAssignment(ctx, actor, domain.MaterialAssignmentInput{RequestID: in.RequestID, StudentID: studentID, DueAt: in.DueAt, Mode: "practice"}, v, keys, "", payload)
}

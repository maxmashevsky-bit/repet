package store

import (
	"context"

	"github.com/jackc/pgx/v5"
	"tutor-platform/services/core/internal/apperr"
	"tutor-platform/services/core/internal/domain"
)

const familyGrant = `$3='guardian' and exists(select 1 from guardian_links g where g.organization_id=$2 and g.student_id=$1 and g.guardian_id=$4 and g.status='active')`
const familyRunScope = `la.student_id=$1 and c.organization_id=$2 and ` + familyGrant
const familyFacts = analyticsFactsHead + familyRunScope + analyticsFactsTail

func ValidateFamilyFilter(f domain.FamilyFilter) error {
	if err := ValidateAnalyticsFilter(f.AnalyticsFilter); err != nil {
		return err
	}
	for _, n := range []int{f.LessonOffset, f.AssignmentOffset, f.NoteOffset, f.EventOffset, f.ResultOffset} {
		if n < 0 || n > 100000 {
			return apperr.ErrBadRequest
		}
	}
	return nil
}

// Projections are server-owned SQL; only the final offset is appended to bound parameters.
func familyPage[T any](ctx context.Context, q catalogQuerier, selectSQL string, args []any, offset int) (domain.CatalogPage[T], error) {
	return catalogJSON[domain.CatalogPage[T]](ctx, q, `with context as(select $1::uuid student_id,$2::uuid organization_id,$3::text authority,$4::uuid actor_id,$5::timestamptz from_at,$6::timestamptz to_at), eligible as (`+selectSQL+`) select jsonb_build_object('items',coalesce((select jsonb_agg(item order by at desc,id) from(select * from eligible order by at desc,id limit 20 offset $7)p),'[]'::jsonb),'total',(select count(*) from eligible),'limit',20,'offset',$7::int)`, append(args[:len(args):len(args)], offset)...)
}

func (s *Store) FamilyReport(ctx context.Context, actor domain.User, studentID string, f domain.FamilyFilter) (domain.FamilyReport, error) {
	var out domain.FamilyReport
	if err := ValidateFamilyFilter(f); err != nil {
		return out, err
	}
	if _, err := guardianAccess(ctx, s.db, actor, studentID); err != nil {
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
	out.Student, err = guardianAccess(ctx, tx, actor, studentID)
	if err != nil {
		return out, err
	}
	out.From = f.From.Format("2006-01-02")
	out.To = f.To.Format("2006-01-02")
	if err = tx.QueryRow(ctx, `select clock_timestamp()`).Scan(&out.GeneratedAt); err != nil {
		return out, err
	}
	args := []any{studentID, actor.OrganizationID, "guardian", actor.ID, f.From, f.To.AddDate(0, 0, 1)}
	out.Summary, err = catalogJSON[domain.AnalyticsMetric](ctx, tx, familyFacts+`select `+analyticsMetricJSON+` from facts`, args...)
	if err != nil {
		return out, err
	}
	analyticsMetric(&out.Summary)
	out.Topics, err = catalogJSON[domain.CatalogPage[domain.AnalyticsTopic]](ctx, tx, familyFacts+`, grouped as(select topic_key,topic,subtopic,`+analyticsMetricJSON+` metric from facts group by topic_key,topic,subtopic)
 select jsonb_build_object('items',coalesce((select jsonb_agg(item order by topic,subtopic,topic_key) from(select topic,subtopic,topic_key,metric||jsonb_build_object('key',topic_key,'topic',topic,'subtopic',subtopic,'previous_graded',0,'recent_graded',0,'change',null) item from grouped order by topic,subtopic,topic_key limit 20 offset $7)p),'[]'::jsonb),'total',(select count(*) from grouped),'limit',20,'offset',$7::int)`, append(args[:6:6], f.TopicOffset)...)
	if err != nil {
		return out, err
	}
	for i := range out.Topics.Items {
		t := &out.Topics.Items[i]
		analyticsMetric(&t.AnalyticsMetric)
		t.Level = analyticsLevel(t.Graded, t.Accuracy)
	}
	out.Timeline, err = catalogJSON[[]domain.AnalyticsDay](ctx, tx, familyFacts+`select coalesce(jsonb_agg(item order by bucket),'[]'::jsonb) from(select (created_at at time zone 'UTC')::date bucket,jsonb_build_object('day',(created_at at time zone 'UTC')::date,'attempts',count(*),'graded',count(*) filter(where status in ('correct','incorrect')),'correct',count(*) filter(where status='correct')) item from facts group by bucket)d`, args...)
	if err != nil {
		return out, err
	}
	for i := range out.Timeline {
		d := &out.Timeline[i]
		d.Accuracy = analyticsPercent(d.Correct, d.Graded)
	}
	out.Results, err = catalogJSON[domain.CatalogPage[domain.FamilyResult]](ctx, tx, familyFacts+`, eligible as(select f.id,f.created_at at,jsonb_build_object('id',f.id,'lesson_title',v.title,'prompt',e.prompt,'status',f.status,'attempts',(select count(*) from facts all_f where all_f.learning_attempt_id=f.learning_attempt_id and all_f.exercise_id=f.exercise_id),'created_at',f.created_at) item from latest f join learning_attempts la on la.id=f.learning_attempt_id join lesson_versions v on v.id=la.version_id join exercises e on e.id=f.exercise_id) select jsonb_build_object('items',coalesce((select jsonb_agg(item order by at desc,id) from(select * from eligible order by at desc,id limit 20 offset $7)p),'[]'::jsonb),'total',(select count(*) from eligible),'limit',20,'offset',$7::int)`, append(args[:6:6], f.ResultOffset)...)
	if err != nil {
		return out, err
	}
	out.Exams, err = familyPage[domain.AnalyticsExam](ctx, tx, `select ss.id,la.started_at at,jsonb_build_object('id',ss.id,'title',es.title,'started_at',la.started_at,'completed_at',la.completed_at,'points',(select coalesce(sum(case when last.status='correct' then (ss.exercise_points->>last.exercise_key::text)::int else 0 end),0) from(select distinct on(e.exercise_key) e.exercise_key,ea.status from exercise_attempts ea join exercises e on e.id=ea.exercise_id where ea.learning_attempt_id=la.id order by e.exercise_key,ea.attempt_number desc)last),'max_points',(select sum(value::int) from jsonb_each_text(ss.exercise_points)),'pending',(select count(*) from(select distinct on(exercise_id) status from exercise_attempts where learning_attempt_id=la.id order by exercise_id,attempt_number desc)last where status='needs_review')) item`+analyticsRunFrom+`where `+familyRunScope+` and la.mode='exam' and ss.id is not null and la.completed_at is not null and la.started_at >= $5 and la.started_at < $6`, args, f.ExamOffset)
	if err != nil {
		return out, err
	}
	out.Lessons, err = familyPage[domain.FamilyLesson](ctx, tx, `select l.id,l.starts_at at,jsonb_build_object('id',l.id,'title',l.title,'tutor_name',t.display_name,'starts_at',l.starts_at,'ends_at',l.ends_at,'status',l.status) item from lessons l join users t on t.id=l.tutor_id where l.student_id=$1 and l.organization_id=$2 and `+familyGrant+` and l.starts_at >= $5 and l.starts_at < $6`, args, f.LessonOffset)
	if err != nil {
		return out, err
	}
	out.Assignments, err = familyPage[domain.FamilyAssignment](ctx, tx, `select a.id,a.due_at at,jsonb_build_object('id',a.id,'title',a.title,'tutor_name',t.display_name,'subject',a.subject,'topic',a.topic,'status',a.status,'due_at',a.due_at,'score',s.score,'max_score',a.max_score) item from assignments a join users t on t.id=a.tutor_id left join assignment_submissions s on s.assignment_id=a.id and s.student_id=a.student_id where a.student_id=$1 and a.organization_id=$2 and a.status not in ('draft','archived') and `+familyGrant+` and a.due_at >= $5 and a.due_at < $6`, args, f.AssignmentOffset)
	if err != nil {
		return out, err
	}
	out.Notes, err = familyPage[domain.GuardianNote](ctx, tx, `select n.id,n.created_at at,jsonb_build_object('id',n.id,'tutor_name',t.display_name,'body',n.body,'created_at',n.created_at) item from guardian_notes n join users t on t.id=n.tutor_id where n.student_id=$1 and n.organization_id=$2 and n.withdrawn_at is null and exists(select 1 from relations r where r.organization_id=$2 and r.student_id=$1 and r.tutor_id=n.tutor_id and r.status='active') and `+familyGrant+` and n.created_at >= $5 and n.created_at < $6`, args, f.NoteOffset)
	if err != nil {
		return out, err
	}
	const events = `from guardian_events e join guardian_links g on g.student_id=e.student_id and g.organization_id=e.organization_id and g.guardian_id=$4 and g.status='active' where e.student_id=$1 and e.organization_id=$2 and $3='guardian' and e.created_at>=g.accepted_at and e.created_at >= $5 and e.created_at < $6`
	out.Events, err = familyPage[domain.FamilyEvent](ctx, tx, `select e.id,e.created_at at,jsonb_build_object('id',e.id,'kind',e.kind,'created_at',e.created_at,'unread',e.created_at>coalesce(g.seen_at,g.accepted_at)) item `+events, args, f.EventOffset)
	if err != nil {
		return out, err
	}
	if err = tx.QueryRow(ctx, `select count(*) `+events+` and e.created_at>coalesce(g.seen_at,g.accepted_at)`, args...).Scan(&out.Unread); err != nil {
		return out, err
	}
	return out, tx.Commit(ctx)
}

package store

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"tutor-platform/services/core/internal/apperr"
	"tutor-platform/services/core/internal/domain"
	"tutor-platform/services/core/internal/platform"
)

// Progress may be historical; a new quiz must still have a live source grant.
const reviewSourceAccess = `($3='tutor' or
 (a.id is not null and a.status<>'archived' and exists(select 1 from relations r where r.organization_id=$2 and r.tutor_id=a.tutor_id and r.student_id=$1 and r.status='active'))
 or (a.id is null and ss.id is not null and (c.visibility='organization' or exists(select 1 from relations r where r.organization_id=$2 and r.tutor_id=es.author_id and r.student_id=$1 and r.status='active')))
 or (a.id is null and ss.id is null and not l.assignment_only and l.status='published' and c.status='published' and
 (c.visibility='organization' or exists(select 1 from relations r where r.organization_id=$2 and r.tutor_id=c.author_id and r.student_id=$1 and r.status='active')) and ` + catalogTopicPublished + `))`
const reviewData = `with raw as(select coalesce(ea.id,e.id) id,coalesce(ea.status,'unanswered') status,ea.created_at,e.exercise_key,e.position,e.prompt,la.id run_id,la.lesson_id,la.version_id,la.completed_at,
 coalesce(rs.source_run_id,la.id) root_id,coalesce(a.tutor_id,es.author_id,c.author_id) tutor_id,
 coalesce((provenance.item->>'lesson_id')::uuid,la.lesson_id) source_lesson,
 coalesce((provenance.item->'exercise'->>'id')::uuid,e.exercise_key) source_key,
 ` + analyticsTopicKey + ` topic_key,` + analyticsTopicName + ` topic,` + analyticsSubtopicName + ` subtopic,
 coalesce(ea.created_at,la.completed_at) observed_at,greatest(ea.created_at,coalesce(ea.reviewed_at,ea.created_at),la.completed_at) rated_at,
 row_number()over(partition by la.id,e.id order by ea.attempt_number desc) latest,
 first_value(coalesce(ea.status,'unanswered'))over(partition by la.id,e.id order by ea.attempt_number) first_status,
 exists(select 1 from learning_hints h where h.learning_attempt_id=la.id and h.exercise_id=e.id) hinted
 ` + analyticsRunFrom + analyticsTopicFrom + ` join learning_lessons l on l.id=la.lesson_id left join exercise_attempts ea on ea.learning_attempt_id=la.id and ea.exercise_id=e.id
 where ` + analyticsRunScope + ` and ` + reviewSourceAccess + ` and ` + learningExerciseScope + ` and la.completed_at is not null),
 runs as(select *,status='correct' and first_status='correct' and not hinted clean from raw where latest=1),
 ranked as(select *,row_number()over(partition by tutor_id,source_lesson,source_key order by observed_at desc,completed_at desc,id) rank from runs),
 recalled as(select r.*,(select count(distinct (q.completed_at at time zone 'UTC')::date) from runs q
 where q.tutor_id=r.tutor_id and q.source_lesson=r.source_lesson and q.source_key=r.source_key and q.clean and q.observed_at>
 coalesce((select max(b.observed_at) from runs b where b.tutor_id=r.tutor_id and b.source_lesson=r.source_lesson and b.source_key=r.source_key and not b.clean),'-infinity'::timestamptz)) clean_days from ranked r where rank=1),
 scheduled as(select *,case when status in ('incorrect','unanswered') then rated_at else rated_at+make_interval(days=>(array[1,3,7,14,30])[least(5,greatest(1,clean_days))::int]) end due_at from recalled) `

func ValidateReviewFilter(f *domain.ReviewFilter) error {
	if f.Limit == 0 {
		f.Limit = 20
	}
	if f.Mode == "" {
		f.Mode = "due"
	}
	if f.Limit < 1 || f.Limit > 100 || f.Offset < 0 || f.Offset > 100000 || f.ActiveOffset < 0 || f.ActiveOffset > 100000 || (f.Mode != "due" && f.Mode != "errors" && f.Mode != "all") {
		return apperr.ErrBadRequest
	}
	return nil
}
func (s *Store) LearningReviewPlan(ctx context.Context, actor domain.User, studentID string, f domain.ReviewFilter) (domain.ReviewPlan, error) {
	var out domain.ReviewPlan
	if err := ValidateReviewFilter(&f); err != nil {
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
	if _, err = analyticsAccess(ctx, tx, actor, studentID); err != nil {
		return out, err
	}
	if err = tx.QueryRow(ctx, `select clock_timestamp()`).Scan(&out.ServerTime); err != nil {
		return out, err
	}
	args := []any{studentID, actor.OrganizationID, string(actor.Role), actor.ID, out.ServerTime}
	out, err = catalogJSON[domain.ReviewPlan](ctx, tx, reviewData+`select jsonb_build_object('server_time',$5::timestamptz,
 'due_total',(select count(*) from scheduled where status<>'needs_review' and due_at<=$5),
 'mistake_total',(select count(*) from scheduled where status='incorrect'),'unanswered',(select count(*) from scheduled where status='unanswered'),'pending',(select count(*) from scheduled where status='needs_review'),
 'items',jsonb_build_object('items',coalesce((select jsonb_agg(item order by due_at,id) from(select due_at,id,jsonb_build_object('source_run_id',root_id,'lesson_id',lesson_id,'exercise_id',exercise_key,'topic_key',topic_key,'topic',topic,'subtopic',subtopic,'prompt',prompt,'status',status,'clean_days',clean_days,'due_at',due_at) item from scheduled where status<>'needs_review' and ($6='all' or $6='errors' and status in ('incorrect','unanswered') or $6='due' and due_at<=$5) order by due_at,id limit $7 offset $8)p),'[]'::jsonb),
 'total',(select count(*) from scheduled where status<>'needs_review' and ($6='all' or $6='errors' and status in ('incorrect','unanswered') or $6='due' and due_at<=$5)),'limit',$7::int,'offset',$8::int))`, append(args, f.Mode, f.Limit, f.Offset)...)
	if err != nil {
		return out, err
	}
	const reviewScope = analyticsRunFrom + ` join learning_lessons l on l.id=la.lesson_id where ` + analyticsRunScope + ` and ` + reviewSourceAccess + ` and rs.learning_attempt_id is not null`
	out.Active, err = catalogJSON[domain.CatalogPage[domain.ReviewEntry]](ctx, tx, `with active as(select la.id,la.started_at,jsonb_build_object('id',la.id,'title',v.title,'attempt',`+learningJSON+`) item`+reviewScope+` and la.completed_at is null) select jsonb_build_object('items',coalesce((select jsonb_agg(item order by started_at desc,id) from(select * from active order by started_at desc,id limit 20 offset $5)p),'[]'::jsonb),'total',(select count(*) from active),'limit',20,'offset',$5::int)`, append(args[:4:4], f.ActiveOffset)...)
	if err != nil {
		return out, err
	}
	err = tx.QueryRow(ctx, `select count(*)`+reviewScope+` and la.completed_at>=date_trunc('week',$5::timestamptz at time zone 'UTC') at time zone 'UTC'`, args...).Scan(&out.WeeklyCompleted)
	if err != nil {
		return out, err
	}
	out.WeeklyTarget = 3
	return out, tx.Commit(ctx)
}

func (s *Store) StartLearningReview(ctx context.Context, actor domain.User, in domain.ReviewStartInput) (domain.ReviewSession, error) {
	var out domain.ReviewSession
	if actor.Role != domain.RoleStudent {
		return out, apperr.ErrForbidden
	}
	if !platform.ValidUUID(in.RequestID) || !platform.ValidUUID(in.SourceRunID) || in.Count < 1 || in.Count > 10 || (in.Mode != "quiz" && in.Mode != "errors" && in.Mode != "due") {
		return out, apperr.ErrBadRequest
	}
	if in.TopicKey != "" {
		if b, e := hex.DecodeString(in.TopicKey); e != nil || len(b) != 16 {
			return out, apperr.ErrBadRequest
		}
	}
	var sourceID, lessonID string
	err := s.db.QueryRow(ctx, `select coalesce(rs.source_run_id,la.id)::text,la.lesson_id::text from learning_attempts la left join learning_review_sessions rs on rs.learning_attempt_id=la.id where la.id=$1 and la.student_id=$2`, in.SourceRunID, actor.ID).Scan(&sourceID, &lessonID)
	if err != nil {
		return out, catalogError(err)
	}
	if err = s.learningRunAccess(ctx, actor, lessonID, sourceID); err != nil {
		return out, err
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx)
	if err = lockLearning(ctx, tx, actor.ID); err != nil {
		return out, err
	}
	payload, _ := json.Marshal(in)
	var id string
	var matches bool
	err = tx.QueryRow(ctx, `select learning_attempt_id::text,payload=$3::jsonb from learning_review_sessions where student_id=$1 and request_id=$2`, actor.ID, in.RequestID, payload).Scan(&id, &matches)
	if err == nil {
		if !matches {
			return out, apperr.ErrConflict
		}
		if err = tx.Commit(ctx); err != nil {
			return out, err
		}
		return s.GetLearningReview(ctx, actor, id)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return out, err
	}
	var versionID, tutorID string
	var keys []string
	err = tx.QueryRow(ctx, reviewData+`select version_id::text,tutor_id::text,array_agg(exercise_key::text order by due_at,id) from
 (select * from scheduled where root_id=$5 and ($6='' or topic_key=$6) and status<>'needs_review'
 and ($7='quiz' or $7='errors' and status in ('incorrect','unanswered') or $7='due' and due_at<=clock_timestamp()) order by due_at,id limit $8) selected group by version_id,tutor_id`, actor.ID, actor.OrganizationID, string(actor.Role), actor.ID, sourceID, in.TopicKey, in.Mode, in.Count).Scan(&versionID, &tutorID, &keys)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, apperr.ErrInvalidState
	}
	if err != nil {
		return out, err
	}
	var conflict, manual bool
	if err = tx.QueryRow(ctx, `select exists(select 1 from learning_attempts where student_id=$1 and lesson_id=$2 and completed_at is null and (assignment_id is null or mode='exam'))`, actor.ID, lessonID).Scan(&conflict); err != nil {
		return out, err
	}
	if conflict {
		return out, apperr.ErrConflict
	}
	if err = tx.QueryRow(ctx, `select exists(select 1 from exercises where version_id=$1 and exercise_key=any($2::uuid[]) and kind='long_text') and not exists(select 1 from relations where student_id=$3 and tutor_id=$4 and organization_id=$5 and status='active')`, versionID, keys, actor.ID, tutorID, actor.OrganizationID).Scan(&manual); err != nil {
		return out, err
	}
	if manual {
		return out, apperr.ErrInvalidState
	}
	err = tx.QueryRow(ctx, `insert into learning_attempts(lesson_id,version_id,student_id,number,mode,player_step,theory_seen,example_seen)values($1,$2,$3,(select coalesce(max(number),0)+1 from learning_attempts where student_id=$3 and lesson_id=$1),'practice',$4,true,true) returning id::text`, lessonID, versionID, actor.ID, "exercise:"+keys[0]).Scan(&id)
	if err != nil {
		return out, catalogError(err)
	}
	_, err = tx.Exec(ctx, `insert into learning_review_sessions(learning_attempt_id,source_run_id,student_id,tutor_id,exercise_keys,request_id,payload)values($1,$2,$3,$4,$5,$6,$7)`, id, sourceID, actor.ID, tutorID, keys, in.RequestID, payload)
	if err != nil {
		return out, catalogError(err)
	}
	if err = tx.Commit(ctx); err != nil {
		return out, err
	}
	return s.GetLearningReview(ctx, actor, id)
}
func (s *Store) GetLearningReview(ctx context.Context, actor domain.User, id string) (domain.ReviewSession, error) {
	var out domain.ReviewSession
	if actor.Role != domain.RoleStudent {
		return out, apperr.ErrForbidden
	}
	if !platform.ValidUUID(id) {
		return out, apperr.ErrBadRequest
	}
	var lessonID string
	err := s.db.QueryRow(ctx, `select la.lesson_id::text from learning_review_sessions rs join learning_attempts la on la.id=rs.learning_attempt_id where rs.learning_attempt_id=$1 and rs.student_id=$2`, id, actor.ID).Scan(&lessonID)
	if err != nil {
		return out, catalogError(err)
	}
	out.ID = id
	out.State, err = s.GetLearningState(ctx, actor, lessonID, id)
	return out, err
}
func (s *Store) LearningReviewChecks(ctx context.Context, actor domain.User, f domain.CatalogFilter) (domain.CatalogPage[domain.StudyReview], error) {
	if actor.Role != domain.RoleTutor {
		return domain.CatalogPage[domain.StudyReview]{}, apperr.ErrForbidden
	}
	if err := ValidateCatalogFilter(&f); err != nil {
		return domain.CatalogPage[domain.StudyReview]{}, err
	}
	return catalogJSON[domain.CatalogPage[domain.StudyReview]](ctx, s.db, `with eligible as(select a.id,a.created_at,jsonb_build_object('session_id',rs.learning_attempt_id,'lesson_id',l.id,'attempt',`+attemptJSON+`) item`+attemptFrom+` join learning_review_sessions rs on rs.learning_attempt_id=a.learning_attempt_id join learning_attempts source on source.id=rs.source_run_id left join assignments assigned on assigned.id=source.assignment_id
 where l.organization_id=$1 and rs.tutor_id=$2 and a.status='needs_review' and (assigned.id is null or assigned.status<>'archived')
 and exists(select 1 from relations rel where rel.organization_id=$1 and rel.tutor_id=$2 and rel.student_id=a.student_id and rel.status='active')
 and not exists(select 1 from exercise_attempts newer where newer.learning_attempt_id=a.learning_attempt_id and newer.exercise_id=a.exercise_id and newer.attempt_number>a.attempt_number))
 select jsonb_build_object('items',coalesce((select jsonb_agg(item order by created_at,id) from(select * from eligible order by created_at,id limit $3 offset $4)p),'[]'::jsonb),'total',(select count(*) from eligible),'limit',$3::int,'offset',$4::int)`, actor.OrganizationID, actor.ID, f.Limit, f.Offset)
}

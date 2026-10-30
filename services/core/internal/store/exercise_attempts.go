package store

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"

	"tutor-platform/services/core/internal/apperr"
	"tutor-platform/services/core/internal/domain"
	"tutor-platform/services/core/internal/exercise"
	"tutor-platform/services/core/internal/platform"
)

const attemptFrom = ` from exercise_attempts a join exercises e on e.id=a.exercise_id
 join lesson_versions v on v.id=e.version_id join learning_lessons l on l.id=v.lesson_id
 join users student on student.id=a.student_id `
const attemptHidden = `exists(select 1 from learning_attempts hidden where hidden.id=a.learning_attempt_id and hidden.mode='exam' and hidden.completed_at is null)`
const attemptJSON = `jsonb_build_object('id',a.id,'exercise_id',e.exercise_key,'version_id',v.id,'version',v.number,
 'student_id',a.student_id,'student_name',student.display_name,'kind',e.kind,'prompt',e.prompt,
 'answer',a.answer,'status',case when ` + attemptHidden + ` then 'submitted' else a.status end,'feedback',case when ` + attemptHidden + ` then '' else a.feedback end,'created_at',a.created_at,'reviewed_at',case when ` + attemptHidden + ` then null else a.reviewed_at end,
 'learning_attempt_id',a.learning_attempt_id,'attempt_number',a.attempt_number,'hints_used',a.hints_used,'elapsed_seconds',a.elapsed_seconds)`

func (s *Store) SubmitExerciseAnswer(ctx context.Context, actor domain.User, lessonID, exerciseID string, in domain.ExerciseAttemptInput) (domain.ExerciseAttempt, error) {
	var out domain.ExerciseAttempt
	if actor.Role != domain.RoleStudent {
		return out, apperr.ErrForbidden
	}
	for _, id := range []string{lessonID, exerciseID, in.VersionID, in.RequestID} {
		if !platform.ValidUUID(id) {
			return out, apperr.ErrBadRequest
		}
	}
	if in.LearningAttemptID != "" && !platform.ValidUUID(in.LearningAttemptID) {
		return out, apperr.ErrBadRequest
	}
	if in.LearningAttemptID != "" {
		if err := s.learningRunAccess(ctx, actor, lessonID, in.LearningAttemptID); err != nil {
			return out, err
		}
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx)
	var current, authorID string
	if in.LearningAttemptID != "" {
		err = tx.QueryRow(ctx, `select la.version_id::text,coalesce(rs.tutor_id,a.tutor_id,l.author_id)::text from learning_attempts la join learning_lessons l on l.id=la.lesson_id left join assignments a on a.id=la.assignment_id left join learning_review_sessions rs on rs.learning_attempt_id=la.id where la.id=$1 and la.student_id=$2 and la.lesson_id=$3`, in.LearningAttemptID, actor.ID, lessonID).Scan(&current, &authorID)
	} else {
		err = tx.QueryRow(ctx, `select v.id::text,l.author_id::text`+catalogLessonFrom+` where `+catalogLessonAccess+` and l.id=$4 and v.id=l.published_version_id`, append(catalogArgs(actor), lessonID)...).Scan(&current, &authorID)
	}
	if err != nil {
		return out, catalogError(err)
	}
	if err = lockLearning(ctx, tx, actor.ID); err != nil {
		return out, err
	}
	var otherExam bool
	if err = tx.QueryRow(ctx, `select exists(select 1 from learning_attempts where student_id=$1 and lesson_id=$2 and mode='exam' and completed_at is null and id is distinct from $3::uuid)`, actor.ID, lessonID, nullableUUID(in.LearningAttemptID)).Scan(&otherExam); err != nil {
		return out, err
	}
	if otherExam {
		return out, apperr.ErrInvalidState
	}
	if in.LearningAttemptID != "" {
		var allowed bool
		err = tx.QueryRow(ctx, `select exists(select 1 from learning_attempts la join exercises e on e.version_id=la.version_id where la.id=$1 and e.exercise_key=$2 and `+learningExerciseScope+`)`, in.LearningAttemptID, exerciseID).Scan(&allowed)
		if err != nil {
			return out, err
		}
		if !allowed {
			return out, apperr.ErrNotFound
		}
	}
	if in.LearningAttemptID == "" {
		var exam bool
		if err = tx.QueryRow(ctx, `select exists(select 1 from learning_attempts where student_id=$1 and lesson_id=$2 and mode='exam' and completed_at is null)`, actor.ID, lessonID).Scan(&exam); err != nil {
			return out, err
		}
		if exam {
			return out, apperr.ErrInvalidState
		}
	}
	previous, err := catalogJSON[domain.ExerciseAttempt](ctx, tx, `select `+attemptJSON+attemptFrom+`where a.student_id=$1 and a.request_id=$2`, actor.ID, in.RequestID)
	if err == nil {
		if previous.ExerciseID != exerciseID || previous.VersionID != in.VersionID || !reflect.DeepEqual(previous.Answer, in.Answer) {
			return out, apperr.ErrConflict
		}
		if in.LearningAttemptID != "" && (previous.LearningAttemptID == nil || *previous.LearningAttemptID != in.LearningAttemptID) {
			return out, apperr.ErrConflict
		}
		var actualLesson string
		if err = tx.QueryRow(ctx, `select v.lesson_id::text from exercise_attempts a join exercises e on e.id=a.exercise_id join lesson_versions v on v.id=e.version_id where a.id=$1`, previous.ID).Scan(&actualLesson); err != nil {
			return out, err
		}
		if actualLesson != lessonID {
			return out, apperr.ErrConflict
		}
		return previous, tx.Commit(ctx)
	}
	if !errors.Is(err, apperr.ErrNotFound) {
		return out, err
	}
	if in.LearningAttemptID != "" {
		var completed bool
		err = tx.QueryRow(ctx, `select version_id::text,completed_at is not null from learning_attempts where id=$1 and student_id=$2 and lesson_id=$3`, in.LearningAttemptID, actor.ID, lessonID).Scan(&current, &completed)
		if err != nil {
			return out, catalogError(err)
		}
		if completed {
			return out, apperr.ErrInvalidState
		}
		var mode, step string
		if err = tx.QueryRow(ctx, `select mode,player_step from learning_attempts where id=$1`, in.LearningAttemptID).Scan(&mode, &step); err != nil {
			return out, err
		}
		if mode == "exam" {
			var answered bool
			if err = tx.QueryRow(ctx, `select exists(select 1 from exercise_attempts a join exercises e on e.id=a.exercise_id where a.learning_attempt_id=$1 and e.exercise_key=$2) and not exists(select 1 from study_sessions where learning_attempt_id=$1)`, in.LearningAttemptID, exerciseID).Scan(&answered); err != nil {
				return out, err
			}
			if answered || step != "exercise:"+exerciseID {
				return out, apperr.ErrInvalidState
			}
		}
	}
	if current != in.VersionID {
		return out, apperr.ErrConflict
	}
	var rowID, kind, key string
	var options []string
	var config json.RawMessage
	err = tx.QueryRow(ctx, `select e.id::text,e.kind,e.options,k.answer,e.config from exercises e join exercise_answer_keys k on k.exercise_id=e.id where e.version_id=$1 and e.exercise_key=$2`, current, exerciseID).Scan(&rowID, &kind, &options, &key, &config)
	if err != nil {
		return out, catalogError(err)
	}
	status, err := exercise.CheckExercise(domain.Exercise{Kind: kind, Options: options, Config: config}, key, in.Answer)
	if err != nil {
		return out, err
	}
	if status == "needs_review" {
		// An open template does not make its author the student's teacher.
		var relation string
		if err = tx.QueryRow(ctx, `select id::text from relations where organization_id=$1 and tutor_id=$2 and student_id=$3 and status='active' for share`, actor.OrganizationID, authorID, actor.ID).Scan(&relation); err != nil {
			return out, catalogError(err)
		}
	}
	data, err := json.Marshal(in.Answer)
	if err != nil {
		return out, err
	}
	var attemptID string
	err = tx.QueryRow(ctx, `insert into exercise_attempts(exercise_id,student_id,request_id,answer,status,learning_attempt_id,attempt_number,hints_used,elapsed_seconds)
 values($1,$2,$3,$4,$5,$6,
 (select coalesce(max(attempt_number),0)+1 from exercise_attempts where student_id=$2 and exercise_id=$1),
 (select count(*) from learning_hints where learning_attempt_id=$6 and exercise_id=$1),
 coalesce((select greatest(0,floor(extract(epoch from(clock_timestamp()-coalesce((select max(created_at) from exercise_attempts where learning_attempt_id=la.id and exercise_id=$1),la.started_at)))))::bigint from learning_attempts la where la.id=$6),0)) returning id::text`, rowID, actor.ID, in.RequestID, data, status, nullableUUID(in.LearningAttemptID)).Scan(&attemptID)
	if err != nil {
		return out, catalogError(err)
	}
	if status == "needs_review" {
		href, err := exerciseResultHref(ctx, tx, in.LearningAttemptID, lessonID)
		if err != nil {
			return out, err
		}
		var study bool
		if err = tx.QueryRow(ctx, `select exists(select 1 from study_sessions where learning_attempt_id=$1)`, nullableUUID(in.LearningAttemptID)).Scan(&study); err != nil {
			return out, err
		}
		if study {
			href = "/tasks/study/review"
		}
		var review bool
		if err = tx.QueryRow(ctx, `select exists(select 1 from learning_review_sessions where learning_attempt_id=$1)`, nullableUUID(in.LearningAttemptID)).Scan(&review); err != nil {
			return out, err
		}
		if review {
			href = "/tasks/review/check"
		}
		if err = insertNotification(ctx, tx, actor.OrganizationID, authorID, "Решение на проверку", actor.DisplayName+" отправил развёрнутое решение.", href, "exercise-attempt:"+attemptID); err != nil {
			return out, err
		}
	}
	out, err = catalogJSON[domain.ExerciseAttempt](ctx, tx, `select `+attemptJSON+attemptFrom+`where a.id=$1`, attemptID)
	if err != nil {
		return out, err
	}
	return out, tx.Commit(ctx)
}

func (s *Store) ListExerciseAttempts(ctx context.Context, actor domain.User, lessonID string, f domain.CatalogFilter) (domain.CatalogPage[domain.ExerciseAttempt], error) {
	var out domain.CatalogPage[domain.ExerciseAttempt]
	if err := ValidateCatalogFilter(&f); err != nil {
		return out, err
	}
	lesson, err := s.GetLearningLesson(ctx, actor, lessonID)
	if err != nil {
		return out, err
	}
	if actor.Role != domain.RoleStudent && !(actor.Role == domain.RoleTutor && lesson.AuthorID == actor.ID) {
		return out, apperr.ErrForbidden
	}
	return catalogJSON[domain.CatalogPage[domain.ExerciseAttempt]](ctx, s.db, `with available as (
 select `+attemptJSON+` as item,a.created_at,a.id`+attemptFrom+`where l.id=$1 and l.organization_id=$2 and ($3='tutor' and l.author_id=$4 and exists(select 1 from relations rel where rel.organization_id=$2 and rel.tutor_id=$4 and rel.student_id=a.student_id and rel.status='active') or $3='student' and a.student_id=$4)
 and not exists(select 1 from learning_attempts la where la.id=a.learning_attempt_id and la.assignment_id is not null)
 and not exists(select 1 from study_sessions ss where ss.learning_attempt_id=a.learning_attempt_id)
 and not exists(select 1 from learning_review_sessions rs where rs.learning_attempt_id=a.learning_attempt_id)
 ), page as (select item from available order by created_at desc,id desc limit $5 offset $6)
 select jsonb_build_object('items',coalesce((select jsonb_agg(item) from page),'[]'::jsonb),'total',(select count(*) from available),'limit',$5::int,'offset',$6::int)`, lessonID, actor.OrganizationID, string(actor.Role), actor.ID, f.Limit, f.Offset)
}

func (s *Store) ReviewExerciseAttempt(ctx context.Context, actor domain.User, lessonID, attemptID string, in domain.ExerciseReviewInput) (domain.ExerciseAttempt, error) {
	var out domain.ExerciseAttempt
	if actor.Role != domain.RoleTutor {
		return out, apperr.ErrForbidden
	}
	if !platform.ValidUUID(lessonID) || !platform.ValidUUID(attemptID) || (in.Status != "correct" && in.Status != "incorrect") || len(in.Feedback) > 10000 || strings.ContainsRune(in.Feedback, 0) {
		return out, apperr.ErrBadRequest
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx)
	var kind, status, feedback string
	err = tx.QueryRow(ctx, `select e.kind,a.status,a.feedback`+attemptFrom+`where a.id=$1 and l.id=$2 and l.organization_id=$3 and not (`+attemptHidden+`) and ((l.author_id=$4 and not exists(select 1 from learning_review_sessions rs where rs.learning_attempt_id=a.learning_attempt_id) and not exists(select 1 from learning_attempts la where la.id=a.learning_attempt_id and la.assignment_id is not null)
	and exists(select 1 from relations r where r.organization_id=$3 and r.tutor_id=$4 and r.student_id=a.student_id and r.status='active'))
 or exists(select 1 from learning_attempts la join assignments assigned on assigned.id=la.assignment_id join relations rel on rel.tutor_id=assigned.tutor_id and rel.student_id=assigned.student_id and rel.organization_id=assigned.organization_id and rel.status='active' where la.id=a.learning_attempt_id and assigned.tutor_id=$4 and assigned.organization_id=$3 and assigned.status<>'archived')
 or exists(select 1 from learning_review_sessions rs join learning_attempts source on source.id=rs.source_run_id left join assignments assigned on assigned.id=source.assignment_id join relations rel on rel.tutor_id=rs.tutor_id and rel.student_id=rs.student_id and rel.organization_id=$3 and rel.status='active' where rs.learning_attempt_id=a.learning_attempt_id and rs.tutor_id=$4 and (assigned.id is null or assigned.status<>'archived'))) for update of a`, attemptID, lessonID, actor.OrganizationID, actor.ID).Scan(&kind, &status, &feedback)
	if err != nil {
		return out, catalogError(err)
	}
	if kind != "long_text" {
		return out, apperr.ErrInvalidState
	}
	var studentID string
	if err = tx.QueryRow(ctx, `select student_id::text from exercise_attempts where id=$1`, attemptID).Scan(&studentID); err != nil {
		return out, err
	}
	if err = lockLearning(ctx, tx, studentID); err != nil {
		return out, err
	}
	if status != "needs_review" && (status != in.Status || feedback != in.Feedback) {
		return out, apperr.ErrConflict
	}
	if status == "needs_review" {
		if _, err = tx.Exec(ctx, `update exercise_attempts set status=$2,feedback=$3,reviewer_id=$4,reviewed_at=now() where id=$1`, attemptID, in.Status, in.Feedback, actor.ID); err != nil {
			return out, err
		}
		var runID *string
		if err = tx.QueryRow(ctx, `select learning_attempt_id::text from exercise_attempts where id=$1`, attemptID).Scan(&runID); err != nil {
			return out, err
		}
		if runID != nil {
			if err = syncMaterialAssignment(ctx, tx, *runID); err != nil {
				return out, err
			}
		}
	}
	out, err = catalogJSON[domain.ExerciseAttempt](ctx, tx, `select `+attemptJSON+attemptFrom+`where a.id=$1`, attemptID)
	if err != nil {
		return out, err
	}
	if status == "needs_review" {
		runID := ""
		if out.LearningAttemptID != nil {
			runID = *out.LearningAttemptID
		}
		href, err := exerciseResultHref(ctx, tx, runID, lessonID)
		if err != nil {
			return out, err
		}
		if err = insertNotification(ctx, tx, actor.OrganizationID, out.StudentID, "Решение проверено", "Преподаватель проверил ваше решение.", href, "exercise-review:"+attemptID); err != nil {
			return out, err
		}
	}
	return out, tx.Commit(ctx)
}

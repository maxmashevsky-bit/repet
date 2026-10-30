package store

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5"
	"tutor-platform/services/core/internal/apperr"
	"tutor-platform/services/core/internal/domain"
	"tutor-platform/services/core/internal/platform"
)

// All progress writes for one student share this lock, including legacy answer submission.
func lockLearning(ctx context.Context, tx pgx.Tx, studentID string) error {
	_, err := tx.Exec(ctx, `select pg_advisory_xact_lock(hashtextextended($1,0))`, studentID+":learning")
	return err
}
func learningReplay(ctx context.Context, tx pgx.Tx, studentID, requestID, kind string, payload []byte) (string, error) {
	var id string
	var matches bool
	err := tx.QueryRow(ctx, `select learning_attempt_id::text,kind=$3 and payload=$4::jsonb from learning_events where student_id=$1 and request_id=$2`, studentID, requestID, kind, payload).Scan(&id, &matches)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if !matches {
		return "", apperr.ErrConflict
	}
	return id, nil
}
func learningEvent(ctx context.Context, tx pgx.Tx, studentID, requestID, runID, kind string, payload []byte) error {
	_, err := tx.Exec(ctx, `insert into learning_events(student_id,request_id,learning_attempt_id,kind,payload) values($1,$2,$3,$4,$5)`, studentID, requestID, runID, kind, payload)
	return err
}
func (s *Store) learningAccess(ctx context.Context, actor domain.User, lessonID string) error {
	if actor.Role != domain.RoleStudent {
		return apperr.ErrForbidden
	}
	_, err := s.GetLearningLesson(ctx, actor, lessonID)
	return err
}
func ensureLearning(ctx context.Context, tx pgx.Tx, actor domain.User, lessonID, versionID string, restart bool, mode string) (string, error) {
	var id string
	var completed bool
	var previousMode string
	err := tx.QueryRow(ctx, `select id::text,completed_at is not null,mode from learning_attempts where student_id=$1 and lesson_id=$2 and assignment_id is null order by number desc limit 1`, actor.ID, lessonID).Scan(&id, &completed, &previousMode)
	if err == nil && (!restart || !completed) {
		if mode != "" && mode != previousMode {
			return "", apperr.ErrConflict
		}
		return id, nil
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return "", err
	}
	if mode == "" {
		mode = previousMode
	}
	if mode == "" {
		mode = "practice"
	}
	err = tx.QueryRow(ctx, `insert into learning_attempts(lesson_id,version_id,student_id,number,mode) values($1,$2,$3,(select coalesce(max(number),0)+1 from learning_attempts where student_id=$3 and lesson_id=$1),$4) returning id::text`, lessonID, versionID, actor.ID, mode).Scan(&id)
	return id, err
}
func (s *Store) StartLearning(ctx context.Context, actor domain.User, lessonID string, in domain.LearningStartInput) (domain.LearningState, error) {
	var out domain.LearningState
	if in.Mode != "" && in.Mode != "practice" && in.Mode != "exam" {
		return out, apperr.ErrBadRequest
	}
	if !platform.ValidUUID(in.RequestID) {
		return out, apperr.ErrBadRequest
	}
	if err := s.learningAccess(ctx, actor, lessonID); err != nil {
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
	payloadMap := map[string]any{"lesson_id": lessonID, "restart": in.Restart}
	if in.Mode != "" {
		payloadMap["mode"] = in.Mode
	}
	payload, _ := json.Marshal(payloadMap)
	id, err := learningReplay(ctx, tx, actor.ID, in.RequestID, "start", payload)
	if err != nil {
		return out, err
	}
	if id == "" {
		var versionID string
		err = tx.QueryRow(ctx, `select published_version_id::text from learning_lessons where id=$1`, lessonID).Scan(&versionID)
		if err != nil {
			return out, catalogError(err)
		}
		id, err = ensureLearning(ctx, tx, actor, lessonID, versionID, in.Restart, in.Mode)
		if err != nil {
			return out, err
		}
		var otherExam bool
		if err = tx.QueryRow(ctx, `select exists(select 1 from learning_attempts where student_id=$1 and lesson_id=$2 and mode='exam' and completed_at is null and id<>$3)`, actor.ID, lessonID, id).Scan(&otherExam); err != nil {
			return out, err
		}
		if otherExam {
			return out, apperr.ErrInvalidState
		}
		if err = learningEvent(ctx, tx, actor.ID, in.RequestID, id, "start", payload); err != nil {
			return out, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return out, err
	}
	return s.GetLearningState(ctx, actor, lessonID, id)
}

const learningJSON = `jsonb_build_object('id',la.id,'lesson_id',la.lesson_id,'version_id',la.version_id,'number',la.number,
 'assignment_id',la.assignment_id,
 'mode',la.mode,'player_step',la.player_step,'theory_seen',la.theory_seen,'example_seen',la.example_seen,
 'status',case when la.completed_at is null then 'in_progress' else 'completed' end,
 'started_at',la.started_at,'completed_at',la.completed_at,
 'elapsed_seconds',greatest(0,floor(extract(epoch from(coalesce(la.completed_at,clock_timestamp())-la.started_at))))::bigint,
 'total_exercises',(select count(*) from exercises e where e.version_id=la.version_id and ` + learningExerciseScope + `),
 'answered_exercises',(select count(distinct exercise_id) from exercise_attempts where learning_attempt_id=la.id),
 'correct_exercises',case when la.mode='exam' and la.completed_at is null then 0 else (select count(*) from(select distinct on(exercise_id) status from exercise_attempts where learning_attempt_id=la.id order by exercise_id,attempt_number desc) latest where status='correct') end,
 'pending_review',case when la.mode='exam' and la.completed_at is null then 0 else (select count(*) from(select distinct on(exercise_id) status from exercise_attempts where learning_attempt_id=la.id order by exercise_id,attempt_number desc) latest where status='needs_review') end,
 'incorrect_attempts',case when la.mode='exam' and la.completed_at is null then 0 else (select count(*) from exercise_attempts where learning_attempt_id=la.id and status='incorrect') end)`

func (s *Store) GetLearningState(ctx context.Context, actor domain.User, lessonID, id string) (domain.LearningState, error) {
	var out domain.LearningState
	if err := s.learningRunAccess(ctx, actor, lessonID, id); err != nil {
		return out, err
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx)
	out, err = readLearningState(ctx, tx, actor, lessonID, id)
	if err != nil {
		return out, err
	}
	return out, tx.Commit(ctx)
}
func readLearningState(ctx context.Context, tx catalogQuerier, actor domain.User, lessonID, id string) (domain.LearningState, error) {
	var out domain.LearningState
	var err error
	if id == "current" {
		if err := tx.QueryRow(ctx, `select id::text from learning_attempts where student_id=$1 and lesson_id=$2 and assignment_id is null order by number desc limit 1`, actor.ID, lessonID).Scan(&id); err != nil {
			return out, catalogError(err)
		}
	}
	if !platform.ValidUUID(id) {
		return out, apperr.ErrBadRequest
	}
	out.Attempt, err = catalogJSON[domain.LearningAttempt](ctx, tx, `select `+learningJSON+` from learning_attempts la where la.id=$1 and la.lesson_id=$2 and la.student_id=$3`, id, lessonID, actor.ID)
	if err != nil {
		return out, err
	}
	out.Version, err = catalogJSON[domain.LessonVersion](ctx, tx, `select `+catalogVersionJSON+` from lesson_versions v where v.id=$1 and v.lesson_id=$2 and v.sealed`, out.Attempt.VersionID, lessonID)
	if err == nil && out.Attempt.AssignmentID != nil {
		var ids []string
		if err = tx.QueryRow(ctx, `select exercise_keys from material_assignments where assignment_id=$1`, *out.Attempt.AssignmentID).Scan(&ids); err != nil {
			return out, err
		}
		out.Version, err = materialVersion(ctx, tx, out.Attempt.VersionID, ids)
	}
	if err == nil && out.Attempt.AssignmentID == nil {
		var ids []string
		e := tx.QueryRow(ctx, `select exercise_keys from study_sessions where learning_attempt_id=$1 union all select exercise_keys from learning_review_sessions where learning_attempt_id=$1`, id).Scan(&ids)
		if e == nil {
			out.Version, err = materialVersion(ctx, tx, out.Attempt.VersionID, ids)
		} else if !errors.Is(e, pgx.ErrNoRows) {
			return out, e
		}
	}
	if err != nil {
		return out, err
	}
	out.Answers, err = catalogJSON[[]domain.ExerciseAttempt](ctx, tx, `select coalesce(jsonb_agg(item),'[]'::jsonb) from(select distinct on(e.exercise_key) `+attemptJSON+` item`+attemptFrom+`where a.learning_attempt_id=$1 and a.student_id=$2 order by e.exercise_key,a.attempt_number desc) latest`, id, actor.ID)
	if err != nil {
		return out, err
	}
	out.Hints, err = catalogJSON[[]domain.LearningHint](ctx, tx, `select coalesce(jsonb_agg(jsonb_build_object('exercise_id',e.exercise_key,'number',h.number,'text',k.hints->>(h.number-1),'revealed_at',h.revealed_at) order by h.number),'[]'::jsonb) from learning_hints h join exercises e on e.id=h.exercise_id join exercise_answer_keys k on k.exercise_id=e.id where h.learning_attempt_id=$1`, id)
	if err != nil {
		return out, err
	}
	return out, nil
}
func (s *Store) ListLearningAttempts(ctx context.Context, actor domain.User, lessonID string, f domain.CatalogFilter) (domain.CatalogPage[domain.LearningAttempt], error) {
	var out domain.CatalogPage[domain.LearningAttempt]
	if err := s.learningAccess(ctx, actor, lessonID); err != nil {
		return out, err
	}
	if err := ValidateCatalogFilter(&f); err != nil {
		return out, err
	}
	return catalogJSON[domain.CatalogPage[domain.LearningAttempt]](ctx, s.db, `with available as(select `+learningJSON+` item,la.number from learning_attempts la where student_id=$1 and lesson_id=$2 and assignment_id is null),page as(select item from available order by number desc limit $3 offset $4) select jsonb_build_object('items',coalesce((select jsonb_agg(item) from page),'[]'::jsonb),'total',(select count(*) from available),'limit',$3::int,'offset',$4::int)`, actor.ID, lessonID, f.Limit, f.Offset)
}

func (s *Store) ListLearningAnswers(ctx context.Context, actor domain.User, lessonID, id string, f domain.CatalogFilter) (domain.CatalogPage[domain.ExerciseAttempt], error) {
	var out domain.CatalogPage[domain.ExerciseAttempt]
	if !platform.ValidUUID(id) {
		return out, apperr.ErrBadRequest
	}
	if _, err := s.GetLearningState(ctx, actor, lessonID, id); err != nil {
		return out, err
	}
	if err := ValidateCatalogFilter(&f); err != nil {
		return out, err
	}
	return catalogJSON[domain.CatalogPage[domain.ExerciseAttempt]](ctx, s.db, `with available as(select `+attemptJSON+` item,a.created_at,a.id`+attemptFrom+`where a.learning_attempt_id=$1 and a.student_id=$2),page as(select item from available order by created_at desc,id desc limit $3 offset $4) select jsonb_build_object('items',coalesce((select jsonb_agg(item) from page),'[]'::jsonb),'total',(select count(*) from available),'limit',$3::int,'offset',$4::int)`, id, actor.ID, f.Limit, f.Offset)
}
func (s *Store) CompleteLearning(ctx context.Context, actor domain.User, lessonID, id string, in domain.LearningActionInput) (domain.LearningState, error) {
	var out domain.LearningState
	if !platform.ValidUUID(id) || !platform.ValidUUID(in.RequestID) {
		return out, apperr.ErrBadRequest
	}
	if err := s.learningRunAccess(ctx, actor, lessonID, id); err != nil {
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
	payload, _ := json.Marshal(map[string]string{"lesson_id": lessonID, "id": id})
	previous, err := learningReplay(ctx, tx, actor.ID, in.RequestID, "complete", payload)
	if err != nil {
		return out, err
	}
	var complete, answered bool
	err = tx.QueryRow(ctx, `select completed_at is not null,not exists(select 1 from exercises e where e.version_id=la.version_id and `+learningExerciseScope+` and not exists(select 1 from exercise_attempts a where a.learning_attempt_id=la.id and a.exercise_id=e.id)) from learning_attempts la where la.id=$1 and student_id=$2 and lesson_id=$3`, id, actor.ID, lessonID).Scan(&complete, &answered)
	if err != nil {
		return out, catalogError(err)
	}
	if previous == "" {
		var studyExam bool
		if err = tx.QueryRow(ctx, `select exists(select 1 from study_sessions ss join learning_attempts la on la.id=ss.learning_attempt_id where la.id=$1 and la.mode='exam')`, id).Scan(&studyExam); err != nil {
			return out, err
		}
		if !answered && !studyExam {
			return out, apperr.ErrInvalidState
		}
		if !complete {
			if _, err = tx.Exec(ctx, `update learning_attempts set completed_at=least(clock_timestamp(),coalesce((select deadline_at from study_sessions where learning_attempt_id=$1),clock_timestamp())),player_step='results',theory_seen=true,example_seen=true where id=$1`, id); err != nil {
				return out, err
			}
		}
		if err = learningEvent(ctx, tx, actor.ID, in.RequestID, id, "complete", payload); err != nil {
			return out, err
		}
		if err = syncMaterialAssignment(ctx, tx, id); err != nil {
			return out, err
		}
		var assignmentID, tutorID, title string
		err = tx.QueryRow(ctx, `select a.id::text,a.tutor_id::text,a.title from assignments a join learning_attempts la on la.assignment_id=a.id where la.id=$1`, id).Scan(&assignmentID, &tutorID, &title)
		if err == nil {
			if err = insertNotification(ctx, tx, actor.OrganizationID, tutorID, "Интерактивная работа завершена", title, "/tasks/"+assignmentID, "material-complete:"+id); err != nil {
				return out, err
			}
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return out, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return out, err
	}
	return s.GetLearningState(ctx, actor, lessonID, id)
}
func (s *Store) RevealLearningHint(ctx context.Context, actor domain.User, lessonID, id, exerciseID string, in domain.LearningActionInput) (domain.LearningState, error) {
	var out domain.LearningState
	if !platform.ValidUUID(id) || !platform.ValidUUID(exerciseID) || !platform.ValidUUID(in.RequestID) {
		return out, apperr.ErrBadRequest
	}
	if err := s.learningRunAccess(ctx, actor, lessonID, id); err != nil {
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
	payload, _ := json.Marshal(map[string]string{"lesson_id": lessonID, "id": id, "exercise_id": exerciseID})
	var activeExam bool
	if err = tx.QueryRow(ctx, `select exists(select 1 from learning_attempts where student_id=$1 and lesson_id=$2 and mode='exam' and completed_at is null)`, actor.ID, lessonID).Scan(&activeExam); err != nil {
		return out, err
	}
	if activeExam {
		return out, apperr.ErrInvalidState
	}
	previous, err := learningReplay(ctx, tx, actor.ID, in.RequestID, "hint", payload)
	if err != nil {
		return out, err
	}
	var rowID string
	var available, used int
	var completed bool
	err = tx.QueryRow(ctx, `select e.id::text,jsonb_array_length(k.hints),(select count(*) from learning_hints h where h.learning_attempt_id=la.id and h.exercise_id=e.id),la.completed_at is not null from learning_attempts la join exercises e on e.version_id=la.version_id join exercise_answer_keys k on k.exercise_id=e.id where la.id=$1 and la.student_id=$2 and la.lesson_id=$3 and e.exercise_key=$4 and `+learningExerciseScope, id, actor.ID, lessonID, exerciseID).Scan(&rowID, &available, &used, &completed)
	if err != nil {
		return out, catalogError(err)
	}
	if previous == "" {
		if completed || used >= available {
			return out, apperr.ErrInvalidState
		}
		var mode string
		if err = tx.QueryRow(ctx, `select mode from learning_attempts where id=$1`, id).Scan(&mode); err != nil {
			return out, err
		}
		if mode == "exam" {
			return out, apperr.ErrInvalidState
		}
		if _, err = tx.Exec(ctx, `insert into learning_hints(learning_attempt_id,exercise_id,number) values($1,$2,$3)`, id, rowID, used+1); err != nil {
			return out, err
		}
		if err = learningEvent(ctx, tx, actor.ID, in.RequestID, id, "hint", payload); err != nil {
			return out, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return out, err
	}
	return s.GetLearningState(ctx, actor, lessonID, id)
}

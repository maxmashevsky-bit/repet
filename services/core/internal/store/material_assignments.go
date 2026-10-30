package store

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"tutor-platform/services/core/internal/apperr"
	"tutor-platform/services/core/internal/domain"
	"tutor-platform/services/core/internal/platform"
)

const learningExerciseScope = `(la.assignment_id is null or e.exercise_key in(select unnest(exercise_keys) from material_assignments ma where ma.assignment_id=la.assignment_id))
 and (not exists(select 1 from study_sessions ss where ss.learning_attempt_id=la.id) or e.exercise_key in(select unnest(exercise_keys) from study_sessions ss where ss.learning_attempt_id=la.id))
 and (not exists(select 1 from learning_review_sessions rs where rs.learning_attempt_id=la.id) or e.exercise_key in(select unnest(exercise_keys) from learning_review_sessions rs where rs.learning_attempt_id=la.id))`
const assignmentMaterialJSON = `case when ma.assignment_id is null then null else jsonb_build_object(
 'lesson_id',ma.lesson_id,'version_id',ma.version_id,'exercise_ids',ma.exercise_keys,
 'calendar_lesson_id',ma.calendar_lesson_id,'repeat_of',ma.repeat_of,'mode',ma.mode,
 'attempt',(select ` + learningJSON + ` from learning_attempts la where la.assignment_id=a.id)) end`
const materialAccess = `a.organization_id=$2 and a.kind='material' and a.status<>'archived'
 and (($3='student' and a.student_id=$4) or ($3='tutor' and a.tutor_id=$4))
 and exists(select 1 from relations rel where rel.tutor_id=a.tutor_id and rel.student_id=a.student_id and rel.organization_id=a.organization_id and rel.status='active')`

func (s *Store) GetAssignment(ctx context.Context, actor domain.User, id string) (domain.Assignment, error) {
	item, err := s.getAssignmentForActor(ctx, actor, id)
	if err == nil && item.Kind == "material" {
		err = s.materialAccess(ctx, actor, id)
	}
	return item, err
}
func (s *Store) materialAccess(ctx context.Context, actor domain.User, id string) error {
	if !platform.ValidUUID(id) {
		return apperr.ErrBadRequest
	}
	var found bool
	err := s.db.QueryRow(ctx, `select true from assignments a where a.id=$1 and `+materialAccess, id, actor.OrganizationID, string(actor.Role), actor.ID).Scan(&found)
	return catalogError(err)
}
func (s *Store) learningRunAccess(ctx context.Context, actor domain.User, lessonID, id string) error {
	if actor.Role != domain.RoleStudent {
		return apperr.ErrForbidden
	}
	if id == "current" {
		if err := s.learningAccess(ctx, actor, lessonID); err != nil {
			return err
		}
		var latest string
		if err := s.db.QueryRow(ctx, `select id::text from learning_attempts where student_id=$1 and lesson_id=$2 and assignment_id is null order by number desc limit 1`, actor.ID, lessonID).Scan(&latest); err != nil {
			return catalogError(err)
		}
		return s.learningRunAccess(ctx, actor, lessonID, latest)
	}
	if !platform.ValidUUID(id) || !platform.ValidUUID(lessonID) {
		return apperr.ErrBadRequest
	}
	var assignmentID *string
	var studyID *string
	var sourceID *string
	err := s.db.QueryRow(ctx, `select la.assignment_id::text,ss.id::text,rs.source_run_id::text from learning_attempts la left join study_sessions ss on ss.learning_attempt_id=la.id left join learning_review_sessions rs on rs.learning_attempt_id=la.id where la.id=$1 and la.lesson_id=$2 and la.student_id=$3`, id, lessonID, actor.ID).Scan(&assignmentID, &studyID, &sourceID)
	if err != nil {
		return catalogError(err)
	}
	if assignmentID != nil {
		return s.materialAccess(ctx, actor, *assignmentID)
	}
	if sourceID != nil {
		return s.learningRunAccess(ctx, actor, lessonID, *sourceID)
	}
	if studyID != nil {
		var found bool
		err := s.db.QueryRow(ctx, `select true`+studySessionFrom+`where `+studySessionAccess+` and ss.id=$4`, append(catalogArgs(actor), *studyID)...).Scan(&found)
		return catalogError(err)
	}
	return s.learningAccess(ctx, actor, lessonID)
}
func materialVersion(ctx context.Context, q catalogQuerier, versionID string, ids []string) (domain.LessonVersion, error) {
	v, err := catalogJSON[domain.LessonVersion](ctx, q, `select `+catalogVersionJSON+` from lesson_versions v where v.id=$1 and v.sealed`, versionID)
	if err != nil {
		return v, err
	}
	selected := make([]domain.Exercise, 0, len(ids))
	for _, id := range ids {
		for _, e := range v.Exercises {
			if e.ID == id {
				selected = append(selected, e)
				break
			}
		}
	}
	if len(selected) != len(ids) || len(ids) == 0 {
		return v, apperr.ErrBadRequest
	}
	v.Exercises = selected
	return v, nil
}
func selectedExercises(v domain.LessonVersion, ids []string) ([]string, error) {
	if len(ids) == 0 {
		for _, e := range v.Exercises {
			ids = append(ids, e.ID)
		}
	}
	if len(ids) == 0 || len(ids) > 200 {
		return nil, apperr.ErrBadRequest
	}
	seen := map[string]bool{}
	for _, id := range ids {
		if !platform.ValidUUID(id) || seen[id] {
			return nil, apperr.ErrBadRequest
		}
		seen[id] = true
	}
	out := make([]string, 0, len(ids))
	for _, e := range v.Exercises {
		if seen[e.ID] {
			out = append(out, e.ID)
		}
	}
	if len(out) != len(ids) {
		return nil, apperr.ErrBadRequest
	}
	return out, nil
}
func (s *Store) assignmentSource(ctx context.Context, actor domain.User, lessonID, setID string, ids []string) (domain.LessonVersion, []string, error) {
	var v domain.LessonVersion
	if setID != "" {
		if lessonID != "" || len(ids) > 0 {
			return v, nil, apperr.ErrBadRequest
		}
		set, err := s.getExerciseSet(ctx, actor, setID)
		if err != nil {
			return v, nil, err
		}
		lessonID = set.LessonID
		ids = set.ExerciseIDs
		if _, err = s.GetLearningLesson(ctx, actor, lessonID); err != nil {
			return v, nil, err
		}
		v, err = materialVersion(ctx, s.db, set.VersionID, ids)
		return v, ids, err
	}
	lesson, err := s.GetLearningLesson(ctx, actor, lessonID)
	if err != nil {
		return v, nil, err
	}
	// Authors read drafts by default; assignments always use the sealed publication.
	err = s.db.QueryRow(ctx, `select published_version_id::text from learning_lessons where id=$1 and published_version_id is not null and status='published'`, lesson.ID).Scan(&v.ID)
	if err != nil {
		return v, nil, catalogError(err)
	}
	v, err = catalogJSON[domain.LessonVersion](ctx, s.db, `select `+catalogVersionJSON+` from lesson_versions v where v.id=$1 and v.sealed`, v.ID)
	if err != nil {
		return v, nil, err
	}
	ids, err = selectedExercises(v, ids)
	return v, ids, err
}
func (s *Store) CreateMaterialAssignment(ctx context.Context, actor domain.User, in domain.MaterialAssignmentInput) (domain.Assignment, error) {
	if actor.Role != domain.RoleTutor {
		return domain.Assignment{}, apperr.ErrForbidden
	}
	if !platform.ValidUUID(in.RequestID) || !platform.ValidUUID(in.StudentID) || (!platform.ValidUUID(in.LessonID) && !platform.ValidUUID(in.SetID)) || in.LessonID != "" && in.SetID != "" {
		return domain.Assignment{}, apperr.ErrBadRequest
	}
	if in.Mode == "" {
		in.Mode = "practice"
	}
	if in.Mode != "practice" && in.Mode != "exam" {
		return domain.Assignment{}, apperr.ErrBadRequest
	}
	payload, _ := json.Marshal(in)
	// Return a durable replay even if the source has since been republished.
	if old, err := s.materialReplay(ctx, actor, in.RequestID, payload); err != nil || old.ID != "" {
		return old, err
	}
	v, ids, err := s.assignmentSource(ctx, actor, in.LessonID, in.SetID, in.ExerciseIDs)
	if err != nil {
		return domain.Assignment{}, err
	}
	return s.insertMaterialAssignment(ctx, actor, in, v, ids, "", payload)
}
func (s *Store) materialReplay(ctx context.Context, actor domain.User, requestID string, payload []byte) (domain.Assignment, error) {
	var id string
	var matches bool
	err := s.db.QueryRow(ctx, `select assignment_id::text,payload=$3::jsonb from material_assignments where request_owner=$1 and request_id=$2`, actor.ID, requestID, payload).Scan(&id, &matches)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Assignment{}, nil
	}
	if err != nil {
		return domain.Assignment{}, err
	}
	if !matches {
		return domain.Assignment{}, apperr.ErrConflict
	}
	return s.GetAssignment(ctx, actor, id)
}
func (s *Store) insertMaterialAssignment(ctx context.Context, actor domain.User, in domain.MaterialAssignmentInput, v domain.LessonVersion, ids []string, parent string, payload []byte) (domain.Assignment, error) {
	var out domain.Assignment
	if !in.DueAt.After(time.Now()) || in.CalendarLessonID != "" && !platform.ValidUUID(in.CalendarLessonID) {
		return out, apperr.ErrBadRequest
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `select pg_advisory_xact_lock(hashtextextended($1,0))`, actor.ID+":material:"+in.RequestID); err != nil {
		return out, err
	}
	var oldID string
	var matches bool
	err = tx.QueryRow(ctx, `select assignment_id::text,payload=$3::jsonb from material_assignments where request_owner=$1 and request_id=$2`, actor.ID, in.RequestID, payload).Scan(&oldID, &matches)
	if err == nil {
		if !matches {
			return out, apperr.ErrConflict
		}
		if err = tx.Commit(ctx); err != nil {
			return out, err
		}
		return s.GetAssignment(ctx, actor, oldID)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return out, err
	}
	var relation string
	err = tx.QueryRow(ctx, `select id::text from relations where organization_id=$1 and tutor_id=$2 and student_id=$3 and status='active' for share`, actor.OrganizationID, actor.ID, in.StudentID).Scan(&relation)
	if err != nil {
		return out, catalogError(err)
	}
	if in.CalendarLessonID != "" {
		var calendar string
		err = tx.QueryRow(ctx, `select id::text from lessons where id=$1 and organization_id=$2 and tutor_id=$3 and student_id=$4 and status<>'cancelled' for share`, in.CalendarLessonID, actor.OrganizationID, actor.ID, in.StudentID).Scan(&calendar)
		if err != nil {
			return out, catalogError(err)
		}
	}
	title := v.Title
	if parent != "" {
		title = "Повторение: " + title
	}
	title = materialText(title, 180)
	var id string
	err = tx.QueryRow(ctx, `insert into assignments(id,organization_id,tutor_id,student_id,title,subject,topic,body,status,due_at,max_score,kind) values(gen_random_uuid(),$1,$2,$3,$4,'Математика',$5,$6,'assigned',$7,100,'material') returning id::text`, actor.OrganizationID, actor.ID, in.StudentID, title, materialText(v.Title, 120), v.Summary, in.DueAt).Scan(&id)
	if err != nil {
		return out, err
	}
	var lessonID string
	if err = tx.QueryRow(ctx, `select lesson_id::text from lesson_versions where id=$1 and sealed`, v.ID).Scan(&lessonID); err != nil {
		return out, catalogError(err)
	}
	_, err = tx.Exec(ctx, `insert into material_assignments(assignment_id,lesson_id,version_id,exercise_keys,calendar_lesson_id,repeat_of,mode,request_id,request_owner,payload) values($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, id, lessonID, v.ID, ids, nullableUUID(in.CalendarLessonID), nullableUUID(parent), in.Mode, in.RequestID, actor.ID, payload)
	if err != nil {
		return out, catalogError(err)
	}
	if err = insertNotification(ctx, tx, actor.OrganizationID, in.StudentID, "Новое интерактивное задание", title, "/tasks/"+id, "material:"+id); err != nil {
		return out, err
	}
	if err = tx.Commit(ctx); err != nil {
		return out, err
	}
	s.recordAudit(ctx, actor, "assignment.material_created", "assignment", id, nil)
	return s.GetAssignment(ctx, actor, id)
}
func (s *Store) RepeatMaterialAssignment(ctx context.Context, actor domain.User, id string, in domain.MaterialRepeatInput) (domain.Assignment, error) {
	var out domain.Assignment
	if actor.Role != domain.RoleTutor {
		return out, apperr.ErrForbidden
	}
	if !platform.ValidUUID(in.RequestID) {
		return out, apperr.ErrBadRequest
	}
	item, err := s.GetAssignment(ctx, actor, id)
	if err != nil {
		return out, err
	}
	if item.TutorID != actor.ID || item.Material == nil {
		return out, apperr.ErrNotFound
	}
	payload, _ := json.Marshal(map[string]any{"repeat_of": id, "input": in})
	if old, err := s.materialReplay(ctx, actor, in.RequestID, payload); err != nil || old.ID != "" {
		return old, err
	}
	v, err := materialVersion(ctx, s.db, item.Material.VersionID, item.Material.ExerciseIDs)
	if err != nil {
		return out, err
	}
	return s.insertMaterialAssignment(ctx, actor, domain.MaterialAssignmentInput{RequestID: in.RequestID, StudentID: item.StudentID, DueAt: in.DueAt, CalendarLessonID: in.CalendarLessonID, Mode: item.Material.Mode}, v, item.Material.ExerciseIDs, id, payload)
}
func (s *Store) StartMaterialAssignment(ctx context.Context, actor domain.User, id string, in domain.LearningActionInput) (domain.LearningState, error) {
	var out domain.LearningState
	if actor.Role != domain.RoleStudent {
		return out, apperr.ErrForbidden
	}
	if !platform.ValidUUID(in.RequestID) {
		return out, apperr.ErrBadRequest
	}
	item, err := s.GetAssignment(ctx, actor, id)
	if err != nil {
		return out, err
	}
	if item.Material == nil {
		return out, apperr.ErrNotFound
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx)
	if err = lockLearning(ctx, tx, actor.ID); err != nil {
		return out, err
	}
	payload, _ := json.Marshal(map[string]string{"assignment_id": id})
	runID, err := learningReplay(ctx, tx, actor.ID, in.RequestID, "start", payload)
	if err != nil {
		return out, err
	}
	if runID == "" {
		err = tx.QueryRow(ctx, `select id::text from learning_attempts where assignment_id=$1 and student_id=$2`, id, actor.ID).Scan(&runID)
		if errors.Is(err, pgx.ErrNoRows) {
			var activeExam bool
			if err = tx.QueryRow(ctx, `select exists(select 1 from learning_attempts where student_id=$1 and lesson_id=$2 and mode='exam' and completed_at is null)`, actor.ID, item.Material.LessonID).Scan(&activeExam); err != nil {
				return out, err
			}
			if activeExam {
				return out, apperr.ErrInvalidState
			}
			err = tx.QueryRow(ctx, `insert into learning_attempts(lesson_id,version_id,student_id,number,mode,assignment_id) values($1,$2,$3,(select coalesce(max(number),0)+1 from learning_attempts where student_id=$3 and lesson_id=$1),$4,$5) returning id::text`, item.Material.LessonID, item.Material.VersionID, actor.ID, item.Material.Mode, id).Scan(&runID)
		}
		if err != nil {
			return out, err
		}
		if err = learningEvent(ctx, tx, actor.ID, in.RequestID, runID, "start", payload); err != nil {
			return out, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return out, err
	}
	return s.GetLearningState(ctx, actor, item.Material.LessonID, runID)
}
func (s *Store) MaterialResults(ctx context.Context, actor domain.User, id string, f domain.CatalogFilter) (domain.MaterialResults, error) {
	var out domain.MaterialResults
	item, err := s.GetAssignment(ctx, actor, id)
	if err != nil {
		return out, err
	}
	if item.Material == nil {
		return out, apperr.ErrNotFound
	}
	if err = ValidateCatalogFilter(&f); err != nil {
		return out, err
	}
	out.Assignment = item
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx)
	out.Version, err = materialVersion(ctx, tx, item.Material.VersionID, item.Material.ExerciseIDs)
	if err != nil {
		return out, err
	}
	out.History = domain.CatalogPage[domain.ExerciseAttempt]{Items: []domain.ExerciseAttempt{}, Limit: f.Limit, Offset: f.Offset}
	out.Stats = []domain.AssignmentExerciseStat{}
	if item.Material.Attempt != nil {
		student := actor
		student.ID = item.StudentID
		student.Role = domain.RoleStudent
		state, err := readLearningState(ctx, tx, student, item.Material.LessonID, item.Material.Attempt.ID)
		if err != nil {
			return out, err
		}
		out.State = &state
		out.Stats, err = catalogJSON[[]domain.AssignmentExerciseStat](ctx, tx, `select coalesce(jsonb_agg(item),'[]'::jsonb) from(select jsonb_build_object('exercise_id',e.exercise_key,'attempts',count(*),'mistakes',case when la.mode='exam' and la.completed_at is null then 0 else count(*) filter(where ea.status='incorrect') end) item from exercise_attempts ea join exercises e on e.id=ea.exercise_id join learning_attempts la on la.id=ea.learning_attempt_id where la.id=$1 group by e.exercise_key,la.mode,la.completed_at) stats`, state.Attempt.ID)
		if err != nil {
			return out, err
		}
		out.History, err = catalogJSON[domain.CatalogPage[domain.ExerciseAttempt]](ctx, tx, `with available as(select `+attemptJSON+` item,a.created_at,a.id`+attemptFrom+` where a.learning_attempt_id=$1),page as(select item from available order by created_at desc,id desc limit $2 offset $3) select jsonb_build_object('items',coalesce((select jsonb_agg(item) from page),'[]'::jsonb),'total',(select count(*) from available),'limit',$2::int,'offset',$3::int)`, state.Attempt.ID, f.Limit, f.Offset)
		if err != nil {
			return out, err
		}
	}
	return out, tx.Commit(ctx)
}

func syncMaterialAssignment(ctx context.Context, tx pgx.Tx, runID string) error {
	_, err := tx.Exec(ctx, `update assignments a set status=case when exists(select 1 from (select distinct on(ea.exercise_id) ea.status from exercise_attempts ea where ea.learning_attempt_id=$1 order by ea.exercise_id,ea.attempt_number desc) latest where status='needs_review') then 'submitted' else 'done' end from learning_attempts la where la.id=$1 and la.assignment_id=a.id and la.completed_at is not null`, runID)
	return err
}
func materialText(value string, max int) string {
	value = limit(value, max)
	for !utf8.ValidString(value) {
		value = value[:len(value)-1]
	}
	return value
}
func exerciseResultHref(ctx context.Context, q catalogQuerier, runID, lessonID string) (string, error) {
	href := "/tasks/library/" + lessonID
	if runID == "" {
		return href, nil
	}
	var id *string
	var studyID *string
	if err := q.QueryRow(ctx, `select la.assignment_id::text,ss.id::text from learning_attempts la left join study_sessions ss on ss.learning_attempt_id=la.id where la.id=$1`, runID).Scan(&id, &studyID); err != nil {
		return "", err
	}
	if id != nil {
		href = "/tasks/" + *id
	}
	if studyID != nil {
		href = "/tasks/study/" + *studyID
	}
	return href, nil
}

const setJSON = `jsonb_build_object('id',s.id,'title',s.title,'lesson_id',s.lesson_id,'version_id',s.version_id,'exercise_ids',s.exercise_keys,'created_at',s.created_at)`

func (s *Store) getExerciseSet(ctx context.Context, actor domain.User, id string) (domain.ExerciseSet, error) {
	if actor.Role != domain.RoleTutor {
		return domain.ExerciseSet{}, apperr.ErrForbidden
	}
	if !platform.ValidUUID(id) {
		return domain.ExerciseSet{}, apperr.ErrBadRequest
	}
	return catalogJSON[domain.ExerciseSet](ctx, s.db, `select `+setJSON+` from exercise_sets s where s.id=$1 and s.organization_id=$2 and s.tutor_id=$3`, id, actor.OrganizationID, actor.ID)
}
func (s *Store) ListExerciseSets(ctx context.Context, actor domain.User, f domain.CatalogFilter) (domain.CatalogPage[domain.ExerciseSet], error) {
	var out domain.CatalogPage[domain.ExerciseSet]
	if actor.Role != domain.RoleTutor {
		return out, apperr.ErrForbidden
	}
	if err := ValidateCatalogFilter(&f); err != nil {
		return out, err
	}
	return catalogJSON[domain.CatalogPage[domain.ExerciseSet]](ctx, s.db, `with available as(select `+setJSON+` item,s.created_at,s.id from exercise_sets s where s.organization_id=$1 and s.tutor_id=$2),page as(select item from available order by created_at desc,id desc limit $3 offset $4) select jsonb_build_object('items',coalesce((select jsonb_agg(item) from page),'[]'::jsonb),'total',(select count(*) from available),'limit',$3::int,'offset',$4::int)`, actor.OrganizationID, actor.ID, f.Limit, f.Offset)
}
func (s *Store) SaveExerciseSet(ctx context.Context, actor domain.User, id string, in domain.ExerciseSetInput) (domain.ExerciseSet, error) {
	var out domain.ExerciseSet
	if actor.Role != domain.RoleTutor {
		return out, apperr.ErrForbidden
	}
	if strings.TrimSpace(in.Title) == "" || len(in.Title) > 180 {
		return out, apperr.ErrBadRequest
	}
	if id != "" {
		if _, err := s.getExerciseSet(ctx, actor, id); err != nil {
			return out, err
		}
	}
	v, ids, err := s.assignmentSource(ctx, actor, in.LessonID, "", in.ExerciseIDs)
	if err != nil {
		return out, err
	}
	if id == "" {
		err = s.db.QueryRow(ctx, `insert into exercise_sets(organization_id,tutor_id,title,lesson_id,version_id,exercise_keys) values($1,$2,$3,$4,$5,$6) returning id::text`, actor.OrganizationID, actor.ID, strings.TrimSpace(in.Title), in.LessonID, v.ID, ids).Scan(&id)
	} else {
		_, err = s.db.Exec(ctx, `update exercise_sets set title=$1,lesson_id=$2,version_id=$3,exercise_keys=$4 where id=$5 and organization_id=$6 and tutor_id=$7`, strings.TrimSpace(in.Title), in.LessonID, v.ID, ids, id, actor.OrganizationID, actor.ID)
	}
	if err != nil {
		return out, err
	}
	return s.getExerciseSet(ctx, actor, id)
}
func (s *Store) DeleteExerciseSet(ctx context.Context, actor domain.User, id string) error {
	if _, err := s.getExerciseSet(ctx, actor, id); err != nil {
		return err
	}
	_, err := s.db.Exec(ctx, `delete from exercise_sets where id=$1 and organization_id=$2 and tutor_id=$3`, id, actor.OrganizationID, actor.ID)
	return err
}

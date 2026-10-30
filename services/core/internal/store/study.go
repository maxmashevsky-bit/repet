package store

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"tutor-platform/services/core/internal/apperr"
	"tutor-platform/services/core/internal/domain"
	"tutor-platform/services/core/internal/platform"
)

const studySpecFrom = ` from exam_specifications es join bank_collections bc on bc.id=es.collection_id
 join learning_lessons l on l.id=bc.lesson_id join courses c on c.id=l.course_id join users u on u.id=es.author_id `
const studyCoursePermission = `(c.visibility='organization' or exists(select 1 from relations r where r.organization_id=es.organization_id and r.tutor_id=es.author_id and r.student_id=$2 and r.status='active'))`
const studySpecAccess = `es.organization_id=$1 and (($3='tutor' and es.author_id=$2) or ($3='student' and es.status='published' and c.status='published' and ` + studyCoursePermission + `))`
const studySpecJSON = `jsonb_build_object('id',es.id,'family_id',es.family_id,'version',es.version,'author_id',es.author_id,'author_name',u.display_name,
 'collection_id',es.collection_id,'title',es.title,'direction',es.direction,'year',es.exam_year,'duration_seconds',es.duration_seconds,
 'instructions',es.instructions,'slots',es.slots,'status',es.status,'exercise_count',(select count(distinct (item->>'lesson_id',item->'exercise'->>'id')) from jsonb_array_elements(bc.provenance) item where item->>'direction'=es.direction),
 'topics',coalesce((select jsonb_agg(topic order by topic) from(select distinct item->>'topic' topic from jsonb_array_elements(bc.provenance) item where item->>'direction'=es.direction)t),'[]'::jsonb))`

func validateExamSpecification(in *domain.ExamSpecificationInput) error {
	in.Title = strings.TrimSpace(in.Title)
	if !platform.ValidUUID(in.RequestID) || !platform.ValidUUID(in.CollectionID) || in.ParentID != "" && !platform.ValidUUID(in.ParentID) || !catalogTitle(in.Title) || (in.Direction != "oge" && in.Direction != "ege_basic" && in.Direction != "ege_profile") || in.Year < 1 || in.Year > 9999 || in.DurationSeconds < 60 || in.DurationSeconds > 28800 || len(in.Instructions) > 10000 || strings.ContainsRune(in.Instructions, 0) || len(in.Slots) < 1 || len(in.Slots) > 100 {
		return apperr.ErrBadRequest
	}
	seen := map[int]bool{}
	total := 0
	for _, slot := range in.Slots {
		if slot.Number < 1 || slot.Number > 100 || slot.Count < 1 || slot.Count > 100 || slot.Points < 1 || slot.Points > 100 || seen[slot.Number] {
			return apperr.ErrBadRequest
		}
		seen[slot.Number] = true
		total += slot.Count
	}
	if total > 100 {
		return apperr.ErrBadRequest
	}
	return nil
}
func (s *Store) GetExamSpecification(ctx context.Context, actor domain.User, id string) (domain.ExamSpecification, error) {
	if actor.Role != domain.RoleStudent && actor.Role != domain.RoleTutor {
		return domain.ExamSpecification{}, apperr.ErrForbidden
	}
	if !platform.ValidUUID(id) {
		return domain.ExamSpecification{}, apperr.ErrBadRequest
	}
	return catalogJSON[domain.ExamSpecification](ctx, s.db, `select `+studySpecJSON+studySpecFrom+`where `+studySpecAccess+` and es.id=$4`, append(catalogArgs(actor), id)...)
}
func (s *Store) ListExamSpecifications(ctx context.Context, actor domain.User, f domain.CatalogFilter) (domain.CatalogPage[domain.ExamSpecification], error) {
	if actor.Role != domain.RoleStudent && actor.Role != domain.RoleTutor {
		return domain.CatalogPage[domain.ExamSpecification]{}, apperr.ErrForbidden
	}
	if err := ValidateCatalogFilter(&f); err != nil {
		return domain.CatalogPage[domain.ExamSpecification]{}, err
	}
	return catalogJSON[domain.CatalogPage[domain.ExamSpecification]](ctx, s.db, `with eligible as(select es.id,es.created_at,`+studySpecJSON+` item`+studySpecFrom+`where `+studySpecAccess+`)
 select jsonb_build_object('items',coalesce(jsonb_agg(item order by created_at desc,id),'[]'::jsonb),'total',(select count(*) from eligible),'limit',$4::int,'offset',$5::int)
 from(select * from eligible order by created_at desc,id limit $4 offset $5)p`, append(catalogArgs(actor), f.Limit, f.Offset)...)
}
func (s *Store) CreateExamSpecification(ctx context.Context, actor domain.User, in domain.ExamSpecificationInput) (domain.ExamSpecification, error) {
	var out domain.ExamSpecification
	if actor.Role != domain.RoleTutor {
		return out, apperr.ErrForbidden
	}
	if err := validateExamSpecification(&in); err != nil {
		return out, err
	}
	if _, err := s.GetBankCollection(ctx, actor, in.CollectionID); err != nil {
		return out, err
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `select pg_advisory_xact_lock(hashtextextended($1,12))`, actor.ID+in.RequestID); err != nil {
		return out, err
	}
	payload, _ := json.Marshal(in)
	var old string
	var matches bool
	err = tx.QueryRow(ctx, `select id::text,payload=$3::jsonb from exam_specifications where author_id=$1 and request_id=$2`, actor.ID, in.RequestID, payload).Scan(&old, &matches)
	if err == nil {
		if !matches {
			return out, apperr.ErrConflict
		}
		if err = tx.Commit(ctx); err != nil {
			return out, err
		}
		return s.GetExamSpecification(ctx, actor, old)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return out, err
	}
	family, err := platform.NewUUID()
	if err != nil {
		return out, err
	}
	number := 1
	if in.ParentID != "" {
		if err = tx.QueryRow(ctx, `select family_id::text from exam_specifications where id=$1 and author_id=$2 and organization_id=$3`, in.ParentID, actor.ID, actor.OrganizationID).Scan(&family); err != nil {
			return out, catalogError(err)
		}
		if _, err = tx.Exec(ctx, `select pg_advisory_xact_lock(hashtextextended($1,13))`, family); err != nil {
			return out, err
		}
		if err = tx.QueryRow(ctx, `select max(version)+1 from exam_specifications where family_id=$1`, family).Scan(&number); err != nil {
			return out, err
		}
	}
	slots, _ := json.Marshal(in.Slots)
	var id string
	err = tx.QueryRow(ctx, `insert into exam_specifications(family_id,version,organization_id,author_id,collection_id,title,direction,exam_year,duration_seconds,instructions,slots,request_id,payload) values($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13) returning id::text`, family, number, actor.OrganizationID, actor.ID, in.CollectionID, in.Title, in.Direction, in.Year, in.DurationSeconds, in.Instructions, slots, in.RequestID, payload).Scan(&id)
	if err != nil {
		return out, catalogError(err)
	}
	if err = tx.Commit(ctx); err != nil {
		return out, err
	}
	return s.GetExamSpecification(ctx, actor, id)
}

type studyChoice struct {
	Key, LogicalKey  string
	Number           int
	Topic, Direction string
}

func studyChoices(collection domain.BankCollection) []studyChoice {
	out := make([]studyChoice, 0, len(collection.ExerciseIDs))
	seen := map[string]bool{}
	for i, item := range collection.Provenance {
		if i >= len(collection.ExerciseIDs) {
			break
		}
		logical := item.LessonID + ":" + item.Exercise.ID
		if seen[logical] {
			continue
		}
		seen[logical] = true
		number := 0
		if item.ExamNumber != nil {
			number = *item.ExamNumber
		}
		out = append(out, studyChoice{collection.ExerciseIDs[i], logical, number, item.Topic, item.Direction})
	}
	return out
}
func studyVariant(choices []studyChoice, spec domain.ExamSpecification, seed, mode, topic string, count int, wanted map[string]bool) ([]string, map[string]int, error) {
	pool := append([]studyChoice(nil), choices...)
	sort.Slice(pool, func(i, j int) bool {
		a := sha256.Sum256([]byte(spec.ID + ":" + seed + ":" + pool[i].LogicalKey))
		b := sha256.Sum256([]byte(spec.ID + ":" + seed + ":" + pool[j].LogicalKey))
		return fmt.Sprintf("%x", a) < fmt.Sprintf("%x", b)
	})
	ids := make([]string, 0)
	points := map[string]int{}
	used := map[string]bool{}
	if mode == "practice" {
		for _, item := range pool {
			if item.Direction == spec.Direction && (topic == "" || item.Topic == topic) && (wanted == nil || wanted[item.Key]) {
				ids = append(ids, item.Key)
				points[item.Key] = 1
				if len(ids) == count {
					break
				}
			}
		}
		if len(ids) != count {
			return nil, nil, apperr.ErrInvalidState
		}
		return ids, points, nil
	}
	for _, slot := range spec.Slots {
		n := 0
		for _, item := range pool {
			if item.Direction == spec.Direction && item.Number == slot.Number && !used[item.Key] {
				ids = append(ids, item.Key)
				points[item.Key] = slot.Points
				used[item.Key] = true
				n++
				if n == slot.Count {
					break
				}
			}
		}
		if n != slot.Count {
			return nil, nil, apperr.ErrInvalidState
		}
	}
	return ids, points, nil
}
func (s *Store) SetExamSpecificationStatus(ctx context.Context, actor domain.User, id, status string) (domain.ExamSpecification, error) {
	var out domain.ExamSpecification
	if actor.Role != domain.RoleTutor {
		return out, apperr.ErrForbidden
	}
	if status != "published" && status != "archived" {
		return out, apperr.ErrBadRequest
	}
	spec, err := s.GetExamSpecification(ctx, actor, id)
	if err != nil {
		return out, err
	}
	if status == "published" {
		collection, err := s.GetBankCollection(ctx, actor, spec.CollectionID)
		if err != nil {
			return out, err
		}
		if _, _, err = studyVariant(studyChoices(collection), spec, "validation", "exam", "", 0, nil); err != nil {
			return out, err
		}
	}
	if _, err = s.db.Exec(ctx, `update exam_specifications set status=$2 where id=$1 and author_id=$3 and organization_id=$4`, id, status, actor.ID, actor.OrganizationID); err != nil {
		return out, catalogError(err)
	}
	return s.GetExamSpecification(ctx, actor, id)
}

const studySessionFrom = ` from study_sessions ss join learning_attempts la on la.id=ss.learning_attempt_id
 join exam_specifications es on es.id=ss.specification_id join bank_collections bc on bc.id=es.collection_id
 join learning_lessons l on l.id=bc.lesson_id join courses c on c.id=l.course_id join users u on u.id=es.author_id `
const studySessionAccess = `es.organization_id=$1 and ss.student_id=$2 and $3='student' and ` + studyCoursePermission

func (s *Store) StartStudySession(ctx context.Context, actor domain.User, in domain.StudyStartInput) (domain.StudySession, error) {
	var out domain.StudySession
	if actor.Role != domain.RoleStudent {
		return out, apperr.ErrForbidden
	}
	if !platform.ValidUUID(in.RequestID) || !platform.ValidUUID(in.SpecificationID) || (in.Mode != "practice" && in.Mode != "exam") || len(in.Seed) > 64 || strings.ContainsFunc(in.Seed, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-')
	}) || len(in.Topic) > 400 || in.RepeatSessionID != "" && !platform.ValidUUID(in.RepeatSessionID) || in.Mode == "practice" && (in.Count < 1 || in.Count > 100) || in.Mode == "exam" && (in.Count != 0 || in.Topic != "" || in.RepeatSessionID != "") {
		return out, apperr.ErrBadRequest
	}
	if err := s.expireStudySessions(ctx, actor); err != nil {
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
	var old string
	var matches bool
	err = tx.QueryRow(ctx, `select id::text,payload=$3::jsonb from study_sessions where student_id=$1 and request_id=$2`, actor.ID, in.RequestID, payload).Scan(&old, &matches)
	if err == nil {
		if !matches {
			return out, apperr.ErrConflict
		}
		if err = tx.Commit(ctx); err != nil {
			return out, err
		}
		return s.GetStudySession(ctx, actor, old)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return out, err
	}
	spec, err := catalogJSON[domain.ExamSpecification](ctx, tx, `select `+studySpecJSON+studySpecFrom+`where `+studySpecAccess+` and es.id=$4 and es.status='published' for share of es,c`, append(catalogArgs(actor), in.SpecificationID)...)
	if err != nil {
		return out, err
	}
	collection, err := catalogJSON[domain.BankCollection](ctx, tx, `select `+bankCollectionJSON+` from bank_collections bc join lesson_versions v on v.id=bc.version_id where bc.id=$1`, spec.CollectionID)
	if err != nil {
		return out, err
	}
	var active bool
	if err = tx.QueryRow(ctx, `select exists(select 1 from learning_attempts where student_id=$1 and lesson_id=$2 and assignment_id is null and completed_at is null)`, actor.ID, collection.LessonID).Scan(&active); err != nil {
		return out, err
	}
	if active {
		return out, apperr.ErrConflict
	}
	var wanted map[string]bool
	if in.RepeatSessionID != "" {
		var sourceRun, sourceLesson string
		err = tx.QueryRow(ctx, `select la.id::text,la.lesson_id::text`+studySessionFrom+`where `+studySessionAccess+` and ss.id=$4 and ss.specification_id=$5 and la.completed_at is not null`, append(catalogArgs(actor), in.RepeatSessionID, spec.ID)...).Scan(&sourceRun, &sourceLesson)
		if err != nil {
			return out, catalogError(err)
		}
		state, err := readLearningState(ctx, tx, actor, sourceLesson, sourceRun)
		if err != nil {
			return out, err
		}
		wanted = map[string]bool{}
		for _, ex := range state.Version.Exercises {
			wanted[ex.ID] = true
		}
		for _, a := range state.Answers {
			if a.Status != "incorrect" {
				delete(wanted, a.ExerciseID)
			}
		}
	}
	seed := in.Seed
	if seed == "" {
		seed, err = platform.NewUUID()
		if err != nil {
			return out, err
		}
	}
	keys, points, err := studyVariant(studyChoices(collection), spec, seed, in.Mode, in.Topic, in.Count, wanted)
	if err != nil {
		return out, err
	}
	for _, ex := range collection.Version.Exercises {
		if ex.Kind != "long_text" {
			continue
		}
		selected := false
		for _, key := range keys {
			if key == ex.ID {
				selected = true
				break
			}
		}
		if selected {
			var linked bool
			if err = tx.QueryRow(ctx, `select exists(select 1 from relations where organization_id=$1 and tutor_id=$2 and student_id=$3 and status='active')`, actor.OrganizationID, spec.AuthorID, actor.ID).Scan(&linked); err != nil {
				return out, err
			}
			if !linked {
				return out, apperr.ErrInvalidState
			}
			break
		}
	}
	var runID string
	err = tx.QueryRow(ctx, `insert into learning_attempts(lesson_id,version_id,student_id,number,mode,player_step,theory_seen,example_seen)values($1,$2,$3,(select coalesce(max(number),0)+1 from learning_attempts where student_id=$3 and lesson_id=$1),$4,case when $4='exam' then 'exercise:'||$5 else 'theory' end,$4='exam',$4='exam') returning id::text`, collection.LessonID, collection.VersionID, actor.ID, in.Mode, keys[0]).Scan(&runID)
	if err != nil {
		return out, catalogError(err)
	}
	pointJSON, _ := json.Marshal(points)
	var sid string
	err = tx.QueryRow(ctx, `insert into study_sessions(specification_id,learning_attempt_id,student_id,exercise_keys,exercise_points,seed,deadline_at,request_id,payload)
 select $1,id,$3,$4,$5,$6,case when mode='exam' then started_at+make_interval(secs=>$7) else null end,$8,$9 from learning_attempts where id=$2 returning id::text`, spec.ID, runID, actor.ID, keys, pointJSON, seed, spec.DurationSeconds, in.RequestID, payload).Scan(&sid)
	if err != nil {
		return out, catalogError(err)
	}
	if err = tx.Commit(ctx); err != nil {
		return out, err
	}
	return s.GetStudySession(ctx, actor, sid)
}

func (s *Store) expireStudySessions(ctx context.Context, actor domain.User) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err = lockLearning(ctx, tx, actor.ID); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `update learning_attempts la set completed_at=ss.deadline_at,player_step='results',theory_seen=true,example_seen=true from study_sessions ss where ss.learning_attempt_id=la.id and ss.student_id=$1 and la.completed_at is null and ss.deadline_at<=clock_timestamp()`, actor.ID)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (s *Store) GetStudySession(ctx context.Context, actor domain.User, id string) (domain.StudySession, error) {
	var out domain.StudySession
	if actor.Role != domain.RoleStudent {
		return out, apperr.ErrForbidden
	}
	if !platform.ValidUUID(id) {
		return out, apperr.ErrBadRequest
	}
	var runID, lessonID string
	var points map[string]int
	err := s.db.QueryRow(ctx, `select la.id::text,la.lesson_id::text,ss.seed,ss.deadline_at,ss.exercise_points`+studySessionFrom+`where `+studySessionAccess+` and ss.id=$4`, append(catalogArgs(actor), id)...).Scan(&runID, &lessonID, &out.Seed, &out.DeadlineAt, &points)
	if err != nil {
		return out, catalogError(err)
	}
	if err = s.expireStudySessions(ctx, actor); err != nil {
		return out, err
	}
	out.Specification, err = catalogJSON[domain.ExamSpecification](ctx, s.db, `select `+studySpecJSON+studySessionFrom+`where `+studySessionAccess+` and ss.id=$4`, append(catalogArgs(actor), id)...)
	if err != nil {
		return out, err
	}
	out.State, err = s.GetLearningState(ctx, actor, lessonID, runID)
	if err != nil {
		return out, err
	}
	out.ID = id
	if err = s.db.QueryRow(ctx, `select clock_timestamp()`).Scan(&out.ServerTime); err != nil {
		return out, err
	}
	for _, key := range out.State.Version.Exercises {
		out.MaxPoints += points[key.ID]
	}
	for _, answer := range out.State.Answers {
		if answer.Status == "correct" {
			out.Points += points[answer.ExerciseID]
		}
	}
	return out, nil
}
func (s *Store) ListStudySessions(ctx context.Context, actor domain.User, f domain.CatalogFilter) (domain.CatalogPage[domain.StudyHistoryEntry], error) {
	var out domain.CatalogPage[domain.StudyHistoryEntry]
	if actor.Role != domain.RoleStudent {
		return out, apperr.ErrForbidden
	}
	if err := ValidateCatalogFilter(&f); err != nil {
		return out, err
	}
	if err := s.expireStudySessions(ctx, actor); err != nil {
		return out, err
	}
	return catalogJSON[domain.CatalogPage[domain.StudyHistoryEntry]](ctx, s.db, `with eligible as(select ss.id,ss.created_at,jsonb_build_object('id',ss.id,'title',es.title,'attempt',`+learningJSON+`,'deadline_at',ss.deadline_at,'seed',ss.seed) item`+studySessionFrom+`where `+studySessionAccess+`)
 select jsonb_build_object('items',coalesce(jsonb_agg(item order by created_at desc,id),'[]'::jsonb),'total',(select count(*) from eligible),'limit',$4::int,'offset',$5::int) from(select * from eligible order by created_at desc,id limit $4 offset $5)p`, append(catalogArgs(actor), f.Limit, f.Offset)...)
}
func (s *Store) ListStudyReviews(ctx context.Context, actor domain.User, f domain.CatalogFilter) (domain.CatalogPage[domain.StudyReview], error) {
	if actor.Role != domain.RoleTutor {
		return domain.CatalogPage[domain.StudyReview]{}, apperr.ErrForbidden
	}
	if err := ValidateCatalogFilter(&f); err != nil {
		return domain.CatalogPage[domain.StudyReview]{}, err
	}
	rows, err := s.db.Query(ctx, `select distinct ss.student_id::text`+studySessionFrom+`where es.author_id=$1 and es.organization_id=$2 and la.completed_at is null and ss.deadline_at<=clock_timestamp()`, actor.ID, actor.OrganizationID)
	if err != nil {
		return domain.CatalogPage[domain.StudyReview]{}, err
	}
	students, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return domain.CatalogPage[domain.StudyReview]{}, err
	}
	for _, studentID := range students {
		if err := s.expireStudySessions(ctx, domain.User{ID: studentID}); err != nil {
			return domain.CatalogPage[domain.StudyReview]{}, err
		}
	}
	return catalogJSON[domain.CatalogPage[domain.StudyReview]](ctx, s.db, `with eligible as(select a.id,a.created_at,jsonb_build_object('session_id',ss.id,'lesson_id',l.id,'attempt',`+attemptJSON+`) item`+attemptFrom+`join learning_attempts la on la.id=a.learning_attempt_id join study_sessions ss on ss.learning_attempt_id=la.id join exam_specifications es on es.id=ss.specification_id
 where l.organization_id=$1 and es.author_id=$2 and l.author_id=$2 and a.status='needs_review' and not (`+attemptHidden+`)
 and not exists(select 1 from exercise_attempts newer where newer.learning_attempt_id=a.learning_attempt_id and newer.exercise_id=a.exercise_id and newer.attempt_number>a.attempt_number)
 and exists(select 1 from relations r where r.organization_id=$1 and r.tutor_id=$2 and r.student_id=a.student_id and r.status='active'))
 select jsonb_build_object('items',coalesce(jsonb_agg(item order by created_at,id),'[]'::jsonb),'total',(select count(*) from eligible),'limit',$3::int,'offset',$4::int) from(select * from eligible order by created_at,id limit $3 offset $4)p`, actor.OrganizationID, actor.ID, f.Limit, f.Offset)
}

package store

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"tutor-platform/services/core/internal/apperr"
	"tutor-platform/services/core/internal/domain"
	"tutor-platform/services/core/internal/exercise"
	"tutor-platform/services/core/internal/platform"
)

const catalogOwner = `(c.author_id=$2 and $3='tutor')`
const maxCatalogVersion = 1<<31 - 1
const catalogCourseAccess = `c.organization_id=$1 and (` + catalogOwner + ` or
 (c.status='published' and $3 in ('student','tutor') and (c.visibility='organization' or
 ($3='student' and exists(select 1 from relations r where r.organization_id=c.organization_id
 and r.tutor_id=c.author_id and r.student_id=$2 and r.status='active')))))`

// Every ancestor must be published; hiding a branch also hides its descendants.
const catalogTopicPublished = `not exists(with recursive ancestors as (
 select id,parent_id,status from topics where id=v.topic_id
 union all select t.id,t.parent_id,t.status from topics t join ancestors a on a.parent_id=t.id
) select 1 from ancestors where status<>'published')`
const catalogLessonAccess = catalogCourseAccess + ` and (` + catalogOwner + ` or
 (not l.assignment_only and l.status='published' and v.sealed and v.id=l.published_version_id and ` + catalogTopicPublished + `))`
const catalogLessonFrom = ` from learning_lessons l join courses c on c.id=l.course_id
 join users u on u.id=l.author_id join lesson_versions v on v.lesson_id=l.id `
const catalogCurrentVersion = `v.id=case when ` + catalogOwner + ` then
 (select id from lesson_versions where lesson_id=l.id and number=0) else l.published_version_id end`
const catalogVersionJSON = `(to_jsonb(v)-'search_vector'-'sealed'-'course_id'-'lesson_id') || jsonb_build_object('exercises',
 coalesce((select jsonb_agg(jsonb_build_object('id',e.exercise_key,'kind',e.kind,'prompt',e.prompt,
 'options',e.options,'difficulty',e.difficulty,'position',e.position,'config',e.config,'media',e.media,'hint_count',coalesce((select jsonb_array_length(k.hints) from exercise_answer_keys k where k.exercise_id=e.id),0)) order by e.position,e.exercise_key)
 from exercises e where e.version_id=v.id),'[]'::jsonb))`
const catalogLessonJSON = `jsonb_build_object('id',l.id,'course_id',l.course_id,'author_id',l.author_id,
 'author_name',u.display_name,'status',l.status,'has_unpublished_changes',
 (` + catalogOwner + ` and l.draft_revision is distinct from l.published_revision),'version',` + catalogVersionJSON + `,
 'provenance',(select to_jsonb(p)-'lesson_id' from learning_lesson_provenance p where p.lesson_id=l.id))`
const catalogCourseJSON = `(to_jsonb(c)-'organization_id') || jsonb_build_object('author_name',u.display_name)`

type catalogQuerier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func catalogJSON[T any](ctx context.Context, q catalogQuerier, sql string, args ...any) (T, error) {
	var result T
	var data []byte
	if err := q.QueryRow(ctx, sql, args...).Scan(&data); err != nil {
		return result, catalogError(err)
	}
	err := json.Unmarshal(data, &result)
	return result, err
}

func catalogError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return apperr.ErrNotFound
	}
	var dbErr *pgconn.PgError
	if errors.As(err, &dbErr) {
		if dbErr.ConstraintName == "study_answer_window" {
			return apperr.ErrInvalidState
		}
		switch dbErr.Code {
		case "23503", "23514", "22P02", "22P05", "22003", "22021", "22001":
			return apperr.ErrBadRequest
		case "23505":
			return apperr.ErrConflict
		}
	}
	return err
}

func catalogTrack(track string) bool {
	return track == "school" || track == "olympiad" || track == "oge" || track == "ege"
}
func catalogStatus(status string) bool {
	return status == "draft" || status == "published" || status == "archived"
}
func catalogTitle(title string) bool {
	n := utf8.RuneCountInString(strings.TrimSpace(title))
	return n > 0 && n <= 200
}
func catalogGrades(min, max, difficulty int) bool {
	return min >= 1 && max >= min && max <= 11 && difficulty >= 1 && difficulty <= 5
}

func ValidateCatalogFilter(f *domain.CatalogFilter) error {
	f.Query = strings.TrimSpace(f.Query)
	if f.Limit == 0 {
		f.Limit = 20
	}
	if f.Limit < 1 || f.Limit > 100 || f.Offset < 0 || f.Offset > 100000 || len(f.Query) > 200 || f.Grade < 0 || f.Grade > 11 || f.Difficulty < 0 || f.Difficulty > 5 {
		return apperr.ErrBadRequest
	}
	if f.Track != "" && !catalogTrack(f.Track) || f.Status != "" && !catalogStatus(f.Status) {
		return apperr.ErrBadRequest
	}
	for _, id := range []string{f.AuthorID, f.CourseID, f.TopicID} {
		if id != "" && !platform.ValidUUID(id) {
			return apperr.ErrBadRequest
		}
	}
	return nil
}

func catalogArgs(actor domain.User) []any {
	return []any{actor.OrganizationID, actor.ID, string(actor.Role)}
}

func (s *Store) catalogWrite(ctx context.Context, actor domain.User, courseID string, fn func(pgx.Tx) error) error {
	if actor.Role != domain.RoleTutor {
		return apperr.ErrForbidden
	}
	if !platform.ValidUUID(courseID) {
		return apperr.ErrBadRequest
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var id string
	if err := tx.QueryRow(ctx, `select id::text from courses where id=$1 and organization_id=$2 and author_id=$3 for update`, courseID, actor.OrganizationID, actor.ID).Scan(&id); err != nil {
		return catalogError(err)
	}
	if err := fn(tx); err != nil {
		return catalogError(err)
	}
	return tx.Commit(ctx)
}

func (s *Store) CreateCourse(ctx context.Context, actor domain.User, in domain.CourseInput) (domain.Course, error) {
	if actor.Role != domain.RoleTutor {
		return domain.Course{}, apperr.ErrForbidden
	}
	if err := validateCourse(&in); err != nil {
		return domain.Course{}, err
	}
	var id string
	err := s.db.QueryRow(ctx, `insert into courses(organization_id,author_id,title,description,track,grade_min,grade_max,difficulty,visibility)
 values($1,$2,$3,$4,$5,$6,$7,$8,$9) returning id::text`, actor.OrganizationID, actor.ID, in.Title, in.Description, in.Track, in.GradeMin, in.GradeMax, in.Difficulty, in.Visibility).Scan(&id)
	if err != nil {
		return domain.Course{}, catalogError(err)
	}
	return s.GetCourse(ctx, actor, id)
}

func validateCourse(in *domain.CourseInput) error {
	in.Title = strings.TrimSpace(in.Title)
	if in.Visibility == "" {
		in.Visibility = "linked_students"
	}
	if !catalogTitle(in.Title) || len(in.Description) > 10000 || !catalogTrack(in.Track) || !catalogGrades(in.GradeMin, in.GradeMax, in.Difficulty) || in.Visibility != "linked_students" && in.Visibility != "organization" {
		return apperr.ErrBadRequest
	}
	return nil
}

func (s *Store) UpdateCourse(ctx context.Context, actor domain.User, id string, in domain.CourseInput) (domain.Course, error) {
	if err := validateCourse(&in); err != nil {
		return domain.Course{}, err
	}
	err := s.catalogWrite(ctx, actor, id, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `update courses set title=$2,description=$3,track=$4,grade_min=$5,grade_max=$6,difficulty=$7,visibility=$8,updated_at=now() where id=$1`, id, in.Title, in.Description, in.Track, in.GradeMin, in.GradeMax, in.Difficulty, in.Visibility)
		return err
	})
	if err != nil {
		return domain.Course{}, err
	}
	return s.GetCourse(ctx, actor, id)
}

func (s *Store) SetCourseStatus(ctx context.Context, actor domain.User, id, status string) (domain.Course, error) {
	if status != "published" && status != "archived" {
		return domain.Course{}, apperr.ErrBadRequest
	}
	err := s.catalogWrite(ctx, actor, id, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `update courses set status=$2,updated_at=now() where id=$1`, id, status)
		return err
	})
	if err != nil {
		return domain.Course{}, err
	}
	return s.GetCourse(ctx, actor, id)
}

func (s *Store) GetCourse(ctx context.Context, actor domain.User, id string) (domain.Course, error) {
	if !platform.ValidUUID(id) {
		return domain.Course{}, apperr.ErrBadRequest
	}
	return catalogJSON[domain.Course](ctx, s.db, `select `+catalogCourseJSON+` from courses c join users u on u.id=c.author_id where `+catalogCourseAccess+` and c.id=$4`, append(catalogArgs(actor), id)...)
}

func (s *Store) ListCourses(ctx context.Context, actor domain.User, f domain.CatalogFilter) (domain.CatalogPage[domain.Course], error) {
	if err := ValidateCatalogFilter(&f); err != nil {
		return domain.CatalogPage[domain.Course]{}, err
	}
	args := append(catalogArgs(actor), f.Query, f.Track, f.Grade, f.Difficulty, f.AuthorID, f.Status, f.Limit, f.Offset)
	return catalogJSON[domain.CatalogPage[domain.Course]](ctx, s.db, `with eligible as (select c.created_at,c.id,`+catalogCourseJSON+` as item
 from courses c join users u on u.id=c.author_id where `+catalogCourseAccess+`
 and ($4='' or strpos(lower(c.title||' '||c.description),lower($4))>0) and ($5='' or c.track=$5)
 and ($6=0 or $6 between c.grade_min and c.grade_max) and ($7=0 or c.difficulty=$7)
 and ($8='' or c.author_id=nullif($8,'')::uuid) and ($9='' or c.status=$9))
 select jsonb_build_object('items',coalesce(jsonb_agg(item order by created_at desc,id),'[]'::jsonb),'total',(select count(*) from eligible),'limit',$10::int,'offset',$11::int)
 from (select * from eligible order by created_at desc,id limit $10 offset $11) page`, args...)
}

func validateTopic(in *domain.TopicInput) error {
	in.Title = strings.TrimSpace(in.Title)
	if !catalogTitle(in.Title) || len(in.Description) > 10000 || in.Position < 0 || in.Position > 1000000 || in.ParentID != nil && !platform.ValidUUID(*in.ParentID) {
		return apperr.ErrBadRequest
	}
	return nil
}

func catalogParent(ctx context.Context, tx pgx.Tx, courseID, id string, parentID *string) error {
	if parentID == nil {
		return nil
	}
	var valid bool
	err := tx.QueryRow(ctx, `with recursive ancestors as (select id,parent_id from topics where id=$1 and course_id=$2
 union all select t.id,t.parent_id from topics t join ancestors a on a.parent_id=t.id)
 select exists(select 1 from ancestors) and not exists(select 1 from ancestors where id=nullif($3,'')::uuid)`, *parentID, courseID, id).Scan(&valid)
	if err != nil {
		return err
	}
	if !valid {
		return apperr.ErrBadRequest
	}
	return nil
}

func (s *Store) CreateTopic(ctx context.Context, actor domain.User, courseID string, in domain.TopicInput) (domain.Topic, error) {
	if err := validateTopic(&in); err != nil {
		return domain.Topic{}, err
	}
	var id string
	err := s.catalogWrite(ctx, actor, courseID, func(tx pgx.Tx) error {
		if err := catalogParent(ctx, tx, courseID, "", in.ParentID); err != nil {
			return err
		}
		return tx.QueryRow(ctx, `insert into topics(course_id,parent_id,title,description,position) values($1,$2,$3,$4,$5) returning id::text`, courseID, in.ParentID, in.Title, in.Description, in.Position).Scan(&id)
	})
	if err != nil {
		return domain.Topic{}, err
	}
	return s.GetTopic(ctx, actor, id)
}

func (s *Store) topicCourse(ctx context.Context, id string) (string, error) {
	if !platform.ValidUUID(id) {
		return "", apperr.ErrBadRequest
	}
	var courseID string
	err := s.db.QueryRow(ctx, `select course_id::text from topics where id=$1`, id).Scan(&courseID)
	return courseID, catalogError(err)
}

func (s *Store) UpdateTopic(ctx context.Context, actor domain.User, id string, in domain.TopicInput) (domain.Topic, error) {
	if actor.Role != domain.RoleTutor {
		return domain.Topic{}, apperr.ErrForbidden
	}
	if err := validateTopic(&in); err != nil {
		return domain.Topic{}, err
	}
	courseID, err := s.topicCourse(ctx, id)
	if err != nil {
		return domain.Topic{}, err
	}
	err = s.catalogWrite(ctx, actor, courseID, func(tx pgx.Tx) error {
		if err := catalogParent(ctx, tx, courseID, id, in.ParentID); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `update topics set parent_id=$2,title=$3,description=$4,position=$5,updated_at=now() where id=$1`, id, in.ParentID, in.Title, in.Description, in.Position)
		return err
	})
	if err != nil {
		return domain.Topic{}, err
	}
	return s.GetTopic(ctx, actor, id)
}

func (s *Store) SetTopicStatus(ctx context.Context, actor domain.User, id, status string) (domain.Topic, error) {
	if actor.Role != domain.RoleTutor {
		return domain.Topic{}, apperr.ErrForbidden
	}
	if status != "published" && status != "archived" {
		return domain.Topic{}, apperr.ErrBadRequest
	}
	courseID, err := s.topicCourse(ctx, id)
	if err != nil {
		return domain.Topic{}, err
	}
	err = s.catalogWrite(ctx, actor, courseID, func(tx pgx.Tx) error {
		if status == "published" {
			var ready bool
			err := tx.QueryRow(ctx, `with recursive ancestors as (
 select parent_id from topics where id=$1 union all select t.parent_id from topics t join ancestors a on t.id=a.parent_id)
 select (select status='published' from courses where id=$2) and not exists(select 1 from topics where id in(select parent_id from ancestors) and status<>'published')`, id, courseID).Scan(&ready)
			if err != nil {
				return err
			}
			if !ready {
				return apperr.ErrInvalidState
			}
		}
		_, err := tx.Exec(ctx, `update topics set status=$2,updated_at=now() where id=$1`, id, status)
		return err
	})
	if err != nil {
		return domain.Topic{}, err
	}
	return s.GetTopic(ctx, actor, id)
}

const catalogTopicAccess = catalogCourseAccess + ` and (` + catalogOwner + ` or not exists(with recursive ancestors as (
 select id,parent_id,status from topics where id=t.id union all select p.id,p.parent_id,p.status from topics p join ancestors a on p.id=a.parent_id)
 select 1 from ancestors where status<>'published'))`

func (s *Store) GetTopic(ctx context.Context, actor domain.User, id string) (domain.Topic, error) {
	if !platform.ValidUUID(id) {
		return domain.Topic{}, apperr.ErrBadRequest
	}
	return catalogJSON[domain.Topic](ctx, s.db, `select to_jsonb(t) from topics t join courses c on c.id=t.course_id where `+catalogTopicAccess+` and t.id=$4`, append(catalogArgs(actor), id)...)
}

func (s *Store) ListTopics(ctx context.Context, actor domain.User, courseID string, f domain.CatalogFilter) (domain.CatalogPage[domain.Topic], error) {
	if !platform.ValidUUID(courseID) {
		return domain.CatalogPage[domain.Topic]{}, apperr.ErrBadRequest
	}
	if _, err := s.GetCourse(ctx, actor, courseID); err != nil {
		return domain.CatalogPage[domain.Topic]{}, err
	}
	if err := ValidateCatalogFilter(&f); err != nil {
		return domain.CatalogPage[domain.Topic]{}, err
	}
	return catalogJSON[domain.CatalogPage[domain.Topic]](ctx, s.db, `with eligible as (select t.position,t.id,to_jsonb(t) item from topics t join courses c on c.id=t.course_id
 where `+catalogTopicAccess+` and c.id=$4 and ($5='' or strpos(lower(t.title||' '||t.description),lower($5))>0) and ($6='' or t.status=$6))
 select jsonb_build_object('items',coalesce(jsonb_agg(item order by position,id),'[]'::jsonb),'total',(select count(*) from eligible),'limit',$7::int,'offset',$8::int)
 from(select * from eligible order by position,id limit $7 offset $8)page`, append(catalogArgs(actor), courseID, f.Query, f.Status, f.Limit, f.Offset)...)
}

func validateLearningLesson(in *domain.LearningLessonInput) error {
	if e := in.Example; e != nil {
		if !catalogTitle(e.Title) || strings.TrimSpace(e.Explanation) == "" || len(e.Explanation) > 10000 || strings.ContainsRune(e.Explanation, 0) || e.Min < -1000 || e.Max > 1000 || e.Max <= e.Min || e.Max-e.Min > 100 || len(e.Steps) < 2 || len(e.Steps) > 12 {
			return apperr.ErrBadRequest
		}
		for _, step := range e.Steps {
			if strings.TrimSpace(step.Text) == "" || len(step.Text) > 2000 || strings.ContainsRune(step.Text, 0) || step.Value < e.Min || step.Value > e.Max {
				return apperr.ErrBadRequest
			}
		}
	}
	in.Title = strings.TrimSpace(in.Title)
	if !platform.ValidUUID(in.TopicID) || !catalogTitle(in.Title) || len(in.Summary) > 10000 || len(in.Content) > 250000 || !catalogTrack(in.Track) || !catalogGrades(in.GradeMin, in.GradeMax, in.Difficulty) {
		return apperr.ErrBadRequest
	}
	return nil
}

func (s *Store) CreateLearningLesson(ctx context.Context, actor domain.User, courseID string, in domain.LearningLessonInput) (domain.LearningLesson, error) {
	if err := validateLearningLesson(&in); err != nil {
		return domain.LearningLesson{}, err
	}
	var id string
	err := s.catalogWrite(ctx, actor, courseID, func(tx pgx.Tx) error {
		var err error
		id, err = insertLearningLesson(ctx, tx, actor, courseID, in)
		return err
	})
	if err != nil {
		return domain.LearningLesson{}, err
	}
	return s.GetLearningLesson(ctx, actor, id)
}

func insertLearningLesson(ctx context.Context, tx pgx.Tx, actor domain.User, courseID string, in domain.LearningLessonInput) (string, error) {
	var id string
	if err := tx.QueryRow(ctx, `insert into learning_lessons(organization_id,course_id,author_id) values($1,$2,$3) returning id::text`, actor.OrganizationID, courseID, actor.ID).Scan(&id); err != nil {
		return "", err
	}
	example, _ := json.Marshal(in.Example)
	_, err := tx.Exec(ctx, `insert into lesson_versions(course_id,lesson_id,topic_id,number,title,summary,content,track,grade_min,grade_max,difficulty,example) values($1,$2,$3,0,$4,$5,$6,$7,$8,$9,$10,nullif($11::jsonb,'null'::jsonb))`, courseID, id, in.TopicID, in.Title, in.Summary, in.Content, in.Track, in.GradeMin, in.GradeMax, in.Difficulty, example)
	return id, err
}

func (s *Store) learningCourse(ctx context.Context, actor domain.User, id string) (string, error) {
	if actor.Role != domain.RoleTutor {
		return "", apperr.ErrForbidden
	}
	if !platform.ValidUUID(id) {
		return "", apperr.ErrBadRequest
	}
	var courseID string
	err := s.db.QueryRow(ctx, `select course_id::text from learning_lessons where id=$1 and organization_id=$2 and author_id=$3`, id, actor.OrganizationID, actor.ID).Scan(&courseID)
	return courseID, catalogError(err)
}

func (s *Store) UpdateLearningLesson(ctx context.Context, actor domain.User, id string, in domain.LearningLessonInput) (domain.LearningLesson, error) {
	if err := validateLearningLesson(&in); err != nil {
		return domain.LearningLesson{}, err
	}
	courseID, err := s.learningCourse(ctx, actor, id)
	if err != nil {
		return domain.LearningLesson{}, err
	}
	err = s.catalogWrite(ctx, actor, courseID, func(tx pgx.Tx) error {
		example, _ := json.Marshal(in.Example)
		_, err := tx.Exec(ctx, `update lesson_versions set topic_id=$2,title=$3,summary=$4,content=$5,track=$6,grade_min=$7,grade_max=$8,difficulty=$9,example=nullif($10::jsonb,'null'::jsonb) where lesson_id=$1 and number=0`, id, in.TopicID, in.Title, in.Summary, in.Content, in.Track, in.GradeMin, in.GradeMax, in.Difficulty, example)
		if err != nil {
			return err
		}
		return bumpLearningDraft(ctx, tx, id)
	})
	if err != nil {
		return domain.LearningLesson{}, err
	}
	return s.GetLearningLesson(ctx, actor, id)
}

func bumpLearningDraft(ctx context.Context, tx pgx.Tx, id string) error {
	_, err := tx.Exec(ctx, `update learning_lessons set draft_revision=draft_revision+1,updated_at=now() where id=$1`, id)
	return err
}

func (s *Store) GetLearningLesson(ctx context.Context, actor domain.User, id string) (domain.LearningLesson, error) {
	if !platform.ValidUUID(id) {
		return domain.LearningLesson{}, apperr.ErrBadRequest
	}
	return catalogJSON[domain.LearningLesson](ctx, s.db, `select `+catalogLessonJSON+catalogLessonFrom+`where `+catalogLessonAccess+` and `+catalogCurrentVersion+` and l.id=$4`, append(catalogArgs(actor), id)...)
}

func (s *Store) ListLearningLessons(ctx context.Context, actor domain.User, f domain.CatalogFilter) (domain.CatalogPage[domain.LearningLesson], error) {
	if err := ValidateCatalogFilter(&f); err != nil {
		return domain.CatalogPage[domain.LearningLesson]{}, err
	}
	return catalogJSON[domain.CatalogPage[domain.LearningLesson]](ctx, s.db, `with eligible as (select l.created_at,l.id,`+catalogLessonJSON+` item`+catalogLessonFrom+`
 where `+catalogLessonAccess+` and `+catalogCurrentVersion+`
 and ($4='' or v.search_vector @@ websearch_to_tsquery('russian',$4) or strpos(lower(v.title||' '||v.summary||' '||v.content),lower($4))>0)
 and ($5='' or v.track=$5) and ($6=0 or $6 between v.grade_min and v.grade_max) and ($7=0 or v.difficulty=$7)
 and ($8='' or l.author_id=nullif($8,'')::uuid) and ($9='' or c.id=nullif($9,'')::uuid)
 and ($10='' or v.topic_id in(with recursive descendants as(select id from topics where id=nullif($10,'')::uuid
 union all select t.id from topics t join descendants d on t.parent_id=d.id)select id from descendants))
 and ($11='' or l.status=$11))
 select jsonb_build_object('items',coalesce(jsonb_agg(item order by created_at desc,id),'[]'::jsonb),'total',(select count(*) from eligible),'limit',$12::int,'offset',$13::int)
 from(select * from eligible order by created_at desc,id limit $12 offset $13)page`, append(catalogArgs(actor), f.Query, f.Track, f.Grade, f.Difficulty, f.AuthorID, f.CourseID, f.TopicID, f.Status, f.Limit, f.Offset)...)
}

func validateExercise(in *domain.ExerciseInput) error {
	if len(in.AnswerKey.Hints) > 5 {
		return apperr.ErrBadRequest
	}
	for _, hint := range in.AnswerKey.Hints {
		if strings.TrimSpace(hint) == "" || len(hint) > 4000 || strings.ContainsRune(hint, 0) {
			return apperr.ErrBadRequest
		}
	}
	in.Prompt = strings.TrimSpace(in.Prompt)
	in.AnswerKey.Answer = strings.TrimSpace(in.AnswerKey.Answer)
	if in.Kind == "long_text" {
		in.AnswerKey.Answer = "[manual]"
	}
	if in.Options == nil {
		in.Options = []string{}
	}
	if len(in.Config) == 0 {
		in.Config = json.RawMessage(`{}`)
	}
	if in.Prompt == "" || len(in.Prompt) > 20000 || in.Difficulty < 1 || in.Difficulty > 5 || in.Position < 0 || in.Position > 1000000 || in.AnswerKey.Answer == "" || len(in.AnswerKey.Answer) > 4096 || len(in.AnswerKey.Explanation) > 10000 {
		return apperr.ErrBadRequest
	}
	switch in.Kind {
	case "text", "numeric", "long_text", "number_line", "grid_select", "matching", "ordering", "geometry":
		if len(in.Options) != 0 {
			return apperr.ErrBadRequest
		}
	case "single_choice", "multiple_choice":
		if len(in.Options) < 2 || len(in.Options) > 12 {
			return apperr.ErrBadRequest
		}
		seen := map[string]bool{}
		for _, option := range in.Options {
			if strings.TrimSpace(option) == "" || len(option) > 1000 || seen[option] {
				return apperr.ErrBadRequest
			}
			seen[option] = true
		}
	default:
		return apperr.ErrBadRequest
	}
	return exercise.ValidateExerciseKey(domain.Exercise{Kind: in.Kind, Options: in.Options, Config: in.Config}, in.AnswerKey.Answer)
}

func (s *Store) SaveExercise(ctx context.Context, actor domain.User, lessonID, exerciseID string, in domain.ExerciseInput) (domain.Exercise, error) {
	if err := validateExercise(&in); err != nil {
		return domain.Exercise{}, err
	}
	courseID, err := s.learningCourse(ctx, actor, lessonID)
	if err != nil {
		return domain.Exercise{}, err
	}
	if exerciseID != "" && !platform.ValidUUID(exerciseID) {
		return domain.Exercise{}, apperr.ErrBadRequest
	}
	var result domain.Exercise
	err = s.catalogWrite(ctx, actor, courseID, func(tx pgx.Tx) error {
		if err := s.validatePDFMedia(ctx, tx, actor, in.Media); err != nil {
			return err
		}
		media, _ := json.Marshal(in.Media)
		if in.Media == nil {
			media = []byte(`[]`)
		}
		options, err := json.Marshal(in.Options)
		if err != nil {
			return err
		}
		var rowID, key string
		if exerciseID == "" {
			err = tx.QueryRow(ctx, `insert into exercises(version_id,kind,prompt,options,difficulty,position,config,media)
 select id,$2,$3,$4,$5,$6,$7,$8 from lesson_versions where lesson_id=$1 and number=0 returning id::text,exercise_key::text`, lessonID, in.Kind, in.Prompt, options, in.Difficulty, in.Position, in.Config, media).Scan(&rowID, &key)
		} else {
			err = tx.QueryRow(ctx, `update exercises e set kind=$3,prompt=$4,options=$5,difficulty=$6,position=$7,config=$8,media=$9 from lesson_versions v
 where e.version_id=v.id and v.lesson_id=$1 and v.number=0 and e.exercise_key=$2 returning e.id::text,e.exercise_key::text`, lessonID, exerciseID, in.Kind, in.Prompt, options, in.Difficulty, in.Position, in.Config, media).Scan(&rowID, &key)
		}
		if err != nil {
			return err
		}
		hints, _ := json.Marshal(in.AnswerKey.Hints)
		if in.AnswerKey.Hints == nil {
			hints = []byte(`[]`)
		}
		if _, err = tx.Exec(ctx, `insert into exercise_answer_keys(exercise_id,answer,explanation,hints) values($1,$2,$3,$4)
 on conflict(exercise_id) do update set answer=excluded.answer,explanation=excluded.explanation,hints=excluded.hints`, rowID, in.AnswerKey.Answer, in.AnswerKey.Explanation, hints); err != nil {
			return err
		}
		result = domain.Exercise{ID: key, Kind: in.Kind, Prompt: in.Prompt, Options: in.Options, Difficulty: in.Difficulty, Position: in.Position, Config: in.Config, HintCount: len(in.AnswerKey.Hints), Media: in.Media}
		return bumpLearningDraft(ctx, tx, lessonID)
	})
	return result, err
}

func (s *Store) PublishLearningLesson(ctx context.Context, actor domain.User, id string) (domain.LearningLesson, error) {
	var versionID string
	courseID, err := s.learningCourse(ctx, actor, id)
	if err != nil {
		return domain.LearningLesson{}, err
	}
	err = s.catalogWrite(ctx, actor, courseID, func(tx pgx.Tx) error {
		var ready bool
		err := tx.QueryRow(ctx, `select c.status='published' and `+catalogTopicPublished+`
 and (length(trim(v.content))>0 or exists(select 1 from exercises where version_id=v.id))
 and (not exists(select 1 from html_conversions h where h.learning_lesson_id=l.id) or exists(select 1 from exercises where version_id=v.id))
 and not exists(select 1 from pdf_lesson_sources p where p.lesson_id=l.id and
 (p.reviewed_revision is distinct from l.draft_revision or not exists(select 1 from exercises where version_id=v.id)))
 and not exists(select 1 from exercises e left join exercise_answer_keys k on k.exercise_id=e.id where e.version_id=v.id and k.exercise_id is null)
 and not (l.status='published' and l.draft_revision=l.published_revision)
 from learning_lessons l join courses c on c.id=l.course_id join lesson_versions v on v.lesson_id=l.id and v.number=0 where l.id=$1`, id).Scan(&ready)
		if err != nil {
			return err
		}
		if !ready {
			return apperr.ErrInvalidState
		}
		err = tx.QueryRow(ctx, `insert into lesson_versions(course_id,lesson_id,topic_id,number,title,summary,content,track,grade_min,grade_max,difficulty,published_at,example)
 select course_id,lesson_id,topic_id,(select coalesce(max(number),0)+1 from lesson_versions where lesson_id=$1),title,summary,content,track,grade_min,grade_max,difficulty,now(),example
 from lesson_versions where lesson_id=$1 and number=0 returning id::text`, id).Scan(&versionID)
		if err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `insert into exercises(version_id,exercise_key,kind,prompt,options,difficulty,position,config,media)
 select $2,e.exercise_key,e.kind,e.prompt,e.options,e.difficulty,e.position,e.config,e.media from exercises e join lesson_versions v on v.id=e.version_id where v.lesson_id=$1 and v.number=0`, id, versionID); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `insert into exercise_answer_keys(exercise_id,answer,explanation,hints)
 select dst.id,k.answer,k.explanation,k.hints from exercises src join lesson_versions v on v.id=src.version_id
 join exercise_answer_keys k on k.exercise_id=src.id join exercises dst on dst.version_id=$2 and dst.exercise_key=src.exercise_key where v.lesson_id=$1 and v.number=0`, id, versionID); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `update lesson_versions set sealed=true where id=$1`, versionID); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `update learning_lessons set published_version_id=$2,published_revision=draft_revision,status='published',updated_at=now() where id=$1`, id, versionID)
		return err
	})
	if err != nil {
		return domain.LearningLesson{}, err
	}
	return catalogJSON[domain.LearningLesson](ctx, s.db, `select `+catalogLessonJSON+catalogLessonFrom+`where `+catalogLessonAccess+` and l.id=$4 and v.id=$5`, append(catalogArgs(actor), id, versionID)...)
}

func (s *Store) ArchiveLearningLesson(ctx context.Context, actor domain.User, id string) (domain.LearningLesson, error) {
	courseID, err := s.learningCourse(ctx, actor, id)
	if err != nil {
		return domain.LearningLesson{}, err
	}
	err = s.catalogWrite(ctx, actor, courseID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `update learning_lessons set status='archived',updated_at=now() where id=$1`, id)
		return err
	})
	if err != nil {
		return domain.LearningLesson{}, err
	}
	return s.GetLearningLesson(ctx, actor, id)
}

func (s *Store) ListLearningExercises(ctx context.Context, actor domain.User, id string, f domain.CatalogFilter) (domain.CatalogPage[domain.Exercise], error) {
	if !platform.ValidUUID(id) {
		return domain.CatalogPage[domain.Exercise]{}, apperr.ErrBadRequest
	}
	if _, err := s.GetLearningLesson(ctx, actor, id); err != nil {
		return domain.CatalogPage[domain.Exercise]{}, err
	}
	if err := ValidateCatalogFilter(&f); err != nil {
		return domain.CatalogPage[domain.Exercise]{}, err
	}
	return catalogJSON[domain.CatalogPage[domain.Exercise]](ctx, s.db, `with eligible as (select e.position,e.exercise_key,
 jsonb_build_object('id',e.exercise_key,'kind',e.kind,'prompt',e.prompt,'options',e.options,'difficulty',e.difficulty,'position',e.position,'config',e.config,'media',e.media) item`+catalogLessonFrom+`
 join exercises e on e.version_id=v.id where `+catalogLessonAccess+` and `+catalogCurrentVersion+`
 and l.id=$4 and ($5='' or strpos(lower(e.prompt),lower($5))>0) and ($6=0 or e.difficulty=$6))
 select jsonb_build_object('items',coalesce(jsonb_agg(item order by position,exercise_key),'[]'::jsonb),'total',(select count(*) from eligible),'limit',$7::int,'offset',$8::int)
 from (select * from eligible order by position,exercise_key limit $7 offset $8)page`, append(catalogArgs(actor), id, f.Query, f.Difficulty, f.Limit, f.Offset)...)
}

func (s *Store) GetLearningExercise(ctx context.Context, actor domain.User, id, exerciseID string) (domain.Exercise, error) {
	if !platform.ValidUUID(id) || !platform.ValidUUID(exerciseID) {
		return domain.Exercise{}, apperr.ErrBadRequest
	}
	return catalogJSON[domain.Exercise](ctx, s.db, `select jsonb_build_object('id',e.exercise_key,'kind',e.kind,'prompt',e.prompt,'options',e.options,'difficulty',e.difficulty,'position',e.position,'config',e.config,'media',e.media)`+catalogLessonFrom+`
 join exercises e on e.version_id=v.id where `+catalogLessonAccess+` and `+catalogCurrentVersion+` and l.id=$4 and e.exercise_key=$5`, append(catalogArgs(actor), id, exerciseID)...)
}

func (s *Store) GetLearningVersion(ctx context.Context, actor domain.User, id string, number int) (domain.LearningLesson, error) {
	if !platform.ValidUUID(id) || number < 0 || number > maxCatalogVersion {
		return domain.LearningLesson{}, apperr.ErrBadRequest
	}
	return catalogJSON[domain.LearningLesson](ctx, s.db, `select `+catalogLessonJSON+catalogLessonFrom+`where `+catalogLessonAccess+` and l.id=$4 and v.number=$5`, append(catalogArgs(actor), id, number)...)
}

func (s *Store) ListLearningVersions(ctx context.Context, actor domain.User, id string, f domain.CatalogFilter) (domain.CatalogPage[domain.LessonVersion], error) {
	if !platform.ValidUUID(id) {
		return domain.CatalogPage[domain.LessonVersion]{}, apperr.ErrBadRequest
	}
	if _, err := s.GetLearningLesson(ctx, actor, id); err != nil {
		return domain.CatalogPage[domain.LessonVersion]{}, err
	}
	if err := ValidateCatalogFilter(&f); err != nil {
		return domain.CatalogPage[domain.LessonVersion]{}, err
	}
	return catalogJSON[domain.CatalogPage[domain.LessonVersion]](ctx, s.db, `with eligible as(select v.number,`+catalogVersionJSON+` item`+catalogLessonFrom+`
 where `+catalogLessonAccess+` and l.id=$4 and v.number>0 and v.sealed)
 select jsonb_build_object('items',coalesce(jsonb_agg(item order by number desc),'[]'::jsonb),'total',(select count(*) from eligible),'limit',$5::int,'offset',$6::int)
 from(select * from eligible order by number desc limit $5 offset $6)page`, append(catalogArgs(actor), id, f.Limit, f.Offset)...)
}

func (s *Store) GetExerciseAnswerKeys(ctx context.Context, actor domain.User, id string, number int) ([]domain.ExerciseAnswerKey, error) {
	if _, err := s.learningCourse(ctx, actor, id); err != nil {
		return nil, err
	}
	if number < 0 || number > maxCatalogVersion {
		return nil, apperr.ErrBadRequest
	}
	if _, err := s.GetLearningVersion(ctx, actor, id, number); err != nil {
		return nil, err
	}
	return catalogJSON[[]domain.ExerciseAnswerKey](ctx, s.db, `select coalesce(jsonb_agg(jsonb_build_object('exercise_id',e.exercise_key,'answer',k.answer,'explanation',k.explanation,'hints',k.hints) order by e.position,e.exercise_key),'[]'::jsonb)
 from lesson_versions v join exercises e on e.version_id=v.id join exercise_answer_keys k on k.exercise_id=e.id where v.lesson_id=$1 and v.number=$2`, id, number)
}

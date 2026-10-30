package store

import (
	"context"
	"encoding/json"
	"errors"
	"path"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"tutor-platform/services/core/internal/apperr"
	"tutor-platform/services/core/internal/domain"
	"tutor-platform/services/core/internal/htmlcontent"
	"tutor-platform/services/core/internal/platform"
)

const htmlVersionJSON = `(to_jsonb(v)-'source'-'lesson_id'-'request_id'-'request_hash')`
const htmlLessonJSON = `jsonb_build_object('id',l.id,'course_id',l.course_id,'author_id',l.author_id,'author_name',u.display_name,'status',l.status,'has_unpublished_changes',l.current_version_id is distinct from l.published_version_id,'version',` + htmlVersionJSON + `)`
const htmlFrom = ` from html_lessons l join courses c on c.id=l.course_id join users u on u.id=l.author_id join html_lesson_versions v on v.lesson_id=l.id `
const htmlAccess = catalogCourseAccess + ` and (` + catalogOwner + ` or (l.status='published' and v.id=l.published_version_id and v.published_at is not null and ` + catalogTopicPublished + `))`
const htmlCurrent = `v.id=case when ` + catalogOwner + ` then l.current_version_id else l.published_version_id end`
const assignmentHTMLJSON = `(select jsonb_build_object('lesson_id',hl.lesson_id,'version_id',hl.version_id) from html_assignment_links hl where hl.assignment_id=a.id)`

func (s *Store) GetHTMLLesson(ctx context.Context, actor domain.User, id, versionID, assignmentID string) (domain.HTMLLesson, error) {
	if !platform.ValidUUID(id) || versionID != "" && !platform.ValidUUID(versionID) {
		return domain.HTMLLesson{}, apperr.ErrBadRequest
	}
	if assignmentID != "" {
		a, err := s.getAssignmentForActor(ctx, actor, assignmentID)
		if err != nil {
			return domain.HTMLLesson{}, err
		}
		if a.HTML == nil || a.HTML.LessonID != id || versionID != "" && versionID != a.HTML.VersionID || a.Status == "archived" {
			return domain.HTMLLesson{}, apperr.ErrNotFound
		}
		return catalogJSON[domain.HTMLLesson](ctx, s.db, `select `+htmlLessonJSON+htmlFrom+`where l.id=$1 and v.id=$2 and l.organization_id=$3`, id, a.HTML.VersionID, actor.OrganizationID)
	}
	condition := htmlCurrent
	args := append(catalogArgs(actor), id)
	if versionID != "" {
		condition = `v.id=$5 and (` + catalogOwner + ` or v.id=l.published_version_id)`
		args = append(args, versionID)
	}
	return catalogJSON[domain.HTMLLesson](ctx, s.db, `select `+htmlLessonJSON+htmlFrom+`where `+htmlAccess+` and l.id=$4 and `+condition, args...)
}

func (s *Store) ListHTMLLessons(ctx context.Context, actor domain.User, f domain.CatalogFilter) (domain.CatalogPage[domain.HTMLLesson], error) {
	if f.Limit < 1 || f.Limit > 100 || f.Offset < 0 || f.Offset > 100000 {
		return domain.CatalogPage[domain.HTMLLesson]{}, apperr.ErrBadRequest
	}
	return catalogJSON[domain.CatalogPage[domain.HTMLLesson]](ctx, s.db, `with eligible as(select l.created_at,l.id,`+htmlLessonJSON+` item`+htmlFrom+`where `+htmlAccess+` and `+htmlCurrent+`
 and ($4='' or position(lower($4) in lower(v.title))>0) and ($5='' or l.course_id::text=$5) and ($6='' or v.topic_id::text=$6)
 and ($7::int=0 or $7 between v.grade_min and v.grade_max) and ($8='' or v.track=$8) and ($9::int=0 or v.difficulty=$9)
 and ($10='' or l.status=$10) and ($11='' or l.author_id::text=$11)), page as(select item from eligible order by created_at desc,id limit $12 offset $13)
 select jsonb_build_object('items',coalesce((select jsonb_agg(item) from page),'[]'::jsonb),'total',(select count(*) from eligible),'limit',$12::int,'offset',$13::int)`, actor.OrganizationID, actor.ID, string(actor.Role), f.Query, f.CourseID, f.TopicID, f.Grade, f.Track, f.Difficulty, f.Status, f.AuthorID, f.Limit, f.Offset)
}

func (s *Store) ImportHTMLLesson(ctx context.Context, actor domain.User, id string, in domain.HTMLImportInput) (domain.HTMLLesson, error) {
	var out domain.HTMLLesson
	if actor.Role != domain.RoleTutor {
		return out, apperr.ErrForbidden
	}
	metadata, _, err := htmlcontent.Inspect(in.Source)
	if err != nil {
		return out, err
	}
	in.Title = strings.TrimSpace(in.Title)
	in.Filename = strings.TrimSpace(in.Filename)
	if in.Title == "" {
		in.Title = strings.TrimSpace(metadata.Title)
	}
	if in.Title == "" {
		in.Title = strings.TrimSuffix(in.Filename, path.Ext(in.Filename))
	}
	if !platform.ValidUUID(in.RequestID) || !platform.ValidUUID(in.CourseID) || !platform.ValidUUID(in.TopicID) || id != "" && !platform.ValidUUID(id) || !catalogTitle(in.Title) || len(in.Title) > 180 || len(in.Summary) > 10000 || !catalogTrack(in.Track) || !catalogGrades(in.GradeMin, in.GradeMax, in.Difficulty) || !validPlanText(in.Filename, 180) || strings.ContainsAny(in.Filename, "/\\") || !strings.EqualFold(path.Ext(in.Filename), ".html") && !strings.EqualFold(path.Ext(in.Filename), ".htm") || len(in.License) > 2000 || len(in.SourceLabel) > 1000 || (in.RightsStatus != "own" && in.RightsStatus != "permission" && in.RightsStatus != "open_license") || in.RightsStatus != "own" && strings.TrimSpace(in.License) == "" {
		return out, apperr.ErrBadRequest
	}
	raw, _ := json.Marshal(in)
	requestHash := htmlcontent.SHA256(id + string(raw))
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `select pg_advisory_xact_lock(hashtextextended($1,0))`, actor.ID+":html-upload"); err != nil {
		return out, err
	}
	var previous, previousVersion, hash string
	err = tx.QueryRow(ctx, `select l.id::text,v.id::text,v.request_hash from html_lesson_versions v join html_lessons l on l.id=v.lesson_id where l.organization_id=$1 and l.author_id=$2 and v.request_id=$3`, actor.OrganizationID, actor.ID, in.RequestID).Scan(&previous, &previousVersion, &hash)
	if err == nil {
		if hash != requestHash {
			return out, apperr.ErrConflict
		}
		return s.GetHTMLLesson(ctx, actor, previous, previousVersion, "")
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return out, err
	}
	var course string
	err = tx.QueryRow(ctx, `select c.id::text from courses c join topics t on t.course_id=c.id where c.id=$1 and c.organization_id=$2 and c.author_id=$3 and t.id=$4 for share of c,t`, in.CourseID, actor.OrganizationID, actor.ID, in.TopicID).Scan(&course)
	if err != nil {
		return out, catalogError(err)
	}
	var size, count int64
	if err = tx.QueryRow(ctx, `select coalesce(sum(octet_length(v.source)),0),count(*)`+htmlFrom+`where l.author_id=$1 and l.organization_id=$2`, actor.ID, actor.OrganizationID).Scan(&size, &count); err != nil {
		return out, err
	}
	if size+int64(len(in.Source)) > 16<<20 || count >= 1000 {
		return out, apperr.ErrConflict
	}
	if id == "" {
		err = tx.QueryRow(ctx, `insert into html_lessons(organization_id,author_id,course_id) values($1,$2,$3) returning id::text`, actor.OrganizationID, actor.ID, in.CourseID).Scan(&id)
	} else {
		var found string
		err = tx.QueryRow(ctx, `select id::text from html_lessons where id=$1 and organization_id=$2 and author_id=$3 and course_id=$4 for update`, id, actor.OrganizationID, actor.ID, in.CourseID).Scan(&found)
	}
	if err != nil {
		return out, catalogError(err)
	}
	meta, _ := json.Marshal(metadata)
	var version string
	err = tx.QueryRow(ctx, `insert into html_lesson_versions(lesson_id,topic_id,number,title,summary,track,grade_min,grade_max,difficulty,filename,source,sha256,metadata,rights_status,license,source_label,request_id,request_hash)
 values($1,$2,(select coalesce(max(number),0)+1 from html_lesson_versions where lesson_id=$1),$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17) returning id::text`, id, in.TopicID, in.Title, in.Summary, in.Track, in.GradeMin, in.GradeMax, in.Difficulty, in.Filename, in.Source, htmlcontent.SHA256(in.Source), meta, in.RightsStatus, in.License, in.SourceLabel, in.RequestID, requestHash).Scan(&version)
	if err != nil {
		return out, err
	}
	if _, err = tx.Exec(ctx, `update html_lessons set current_version_id=$2 where id=$1`, id, version); err != nil {
		return out, err
	}
	if err = tx.Commit(ctx); err != nil {
		return out, err
	}
	return s.GetHTMLLesson(ctx, actor, id, "", "")
}

func (s *Store) PublishHTMLLesson(ctx context.Context, actor domain.User, id, version string) (domain.HTMLLesson, error) {
	var out domain.HTMLLesson
	if actor.Role != domain.RoleTutor {
		return out, apperr.ErrForbidden
	}
	if !platform.ValidUUID(id) || !platform.ValidUUID(version) {
		return out, apperr.ErrBadRequest
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx)
	var allowed bool
	err = tx.QueryRow(ctx, `select c.status='published' and `+catalogTopicPublished+htmlFrom+`where l.id=$4 and l.organization_id=$1 and l.author_id=$2 and $3='tutor' and v.id=l.current_version_id and v.id=$5 for update of l`, actor.OrganizationID, actor.ID, string(actor.Role), id, version).Scan(&allowed)
	if err != nil {
		return out, catalogError(err)
	}
	if !allowed {
		return out, apperr.ErrInvalidState
	}
	if _, err = tx.Exec(ctx, `update html_lesson_versions set published_at=now() where id=$1 and published_at is null`, version); err != nil {
		return out, err
	}
	if _, err = tx.Exec(ctx, `update html_lessons set published_version_id=$2,status='published' where id=$1`, id, version); err != nil {
		return out, err
	}
	if err = tx.Commit(ctx); err != nil {
		return out, err
	}
	return s.GetHTMLLesson(ctx, actor, id, "", "")
}
func (s *Store) ArchiveHTMLLesson(ctx context.Context, actor domain.User, id string) error {
	if actor.Role != domain.RoleTutor {
		return apperr.ErrForbidden
	}
	if !platform.ValidUUID(id) {
		return apperr.ErrBadRequest
	}
	r, err := s.db.Exec(ctx, `update html_lessons set status='archived' where id=$1 and author_id=$2 and organization_id=$3`, id, actor.ID, actor.OrganizationID)
	if err == nil && r.RowsAffected() == 0 {
		return apperr.ErrNotFound
	}
	return err
}
func (s *Store) HTMLVersions(ctx context.Context, actor domain.User, id string) ([]domain.HTMLVersion, error) {
	l, err := s.GetHTMLLesson(ctx, actor, id, "", "")
	if err != nil {
		return nil, err
	}
	if actor.Role != domain.RoleTutor || l.AuthorID != actor.ID {
		return nil, apperr.ErrForbidden
	}
	page, err := catalogJSON[domain.CatalogPage[domain.HTMLVersion]](ctx, s.db, `select jsonb_build_object('items',coalesce(jsonb_agg(`+htmlVersionJSON+` order by number desc),'[]'::jsonb)) from html_lesson_versions v where lesson_id=$1`, id)
	return page.Items, err
}
func (s *Store) HTMLOriginal(ctx context.Context, actor domain.User, id, version string) (string, error) {
	l, err := s.GetHTMLLesson(ctx, actor, id, version, "")
	if err != nil {
		return "", err
	}
	if actor.Role != domain.RoleTutor || l.AuthorID != actor.ID {
		return "", apperr.ErrForbidden
	}
	var source string
	err = s.db.QueryRow(ctx, `select source from html_lesson_versions where id=$1`, l.Version.ID).Scan(&source)
	return source, err
}
func (s *Store) CreateHTMLPreview(ctx context.Context, actor domain.User, id, version, assignment string) (string, error) {
	l, err := s.GetHTMLLesson(ctx, actor, id, version, assignment)
	if err != nil {
		return "", err
	}
	token, err := platform.RandomToken(32)
	if err != nil {
		return "", err
	}
	_, err = s.db.Exec(ctx, `delete from html_preview_grants where token_hash in(select token_hash from html_preview_grants where expires_at<now() limit 1000)`)
	if err != nil {
		return "", err
	}
	_, err = s.db.Exec(ctx, `insert into html_preview_grants(token_hash,user_id,lesson_id,version_id,assignment_id,expires_at) values($1,$2,$3,$4,nullif($5,'')::uuid,now()+interval '5 minutes')`, platform.TokenHash(token), actor.ID, id, l.Version.ID, assignment)
	return token, err
}
func (s *Store) ReadHTMLPreview(ctx context.Context, token string) (string, error) {
	if len(token) < 32 || len(token) > 128 {
		return "", apperr.ErrNotFound
	}
	var user, id, version string
	var assignment *string
	err := s.db.QueryRow(ctx, `select user_id::text,lesson_id::text,version_id::text,assignment_id::text from html_preview_grants where token_hash=$1 and expires_at>now()`, platform.TokenHash(token)).Scan(&user, &id, &version, &assignment)
	if err != nil {
		return "", catalogError(err)
	}
	actor, err := s.GetUser(ctx, user)
	if err != nil {
		return "", err
	}
	a := ""
	if assignment != nil {
		a = *assignment
	}
	if _, err = s.GetHTMLLesson(ctx, actor, id, version, a); err != nil {
		return "", err
	}
	var source string
	err = s.db.QueryRow(ctx, `select source from html_lesson_versions where id=$1`, version).Scan(&source)
	return source, err
}

func (s *Store) AssignHTMLLesson(ctx context.Context, actor domain.User, id string, in domain.HTMLAssignInput) (domain.Assignment, error) {
	var out domain.Assignment
	if actor.Role != domain.RoleTutor {
		return out, apperr.ErrForbidden
	}
	if !platform.ValidUUID(id) || !platform.ValidUUID(in.StudentID) || !platform.ValidUUID(in.RequestID) {
		return out, apperr.ErrBadRequest
	}
	payload, _ := json.Marshal(in)
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `select pg_advisory_xact_lock(hashtextextended($1,0))`, actor.ID+":html-assign:"+in.RequestID); err != nil {
		return out, err
	}
	var old string
	var same bool
	err = tx.QueryRow(ctx, `select a.id::text,h.payload=$4::jsonb and h.lesson_id=$3 from html_assignment_links h join assignments a on a.id=h.assignment_id where a.tutor_id=$1 and h.request_id=$2`, actor.ID, in.RequestID, id, payload).Scan(&old, &same)
	if err == nil {
		if !same {
			return out, apperr.ErrConflict
		}
		return s.GetAssignment(ctx, actor, old)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return out, err
	}
	if !in.DueAt.After(time.Now()) {
		return out, apperr.ErrBadRequest
	}
	var relation string
	err = tx.QueryRow(ctx, `select id::text from relations where organization_id=$1 and tutor_id=$2 and student_id=$3 and status='active' for share`, actor.OrganizationID, actor.ID, in.StudentID).Scan(&relation)
	if err != nil {
		return out, catalogError(err)
	}
	l, err := catalogJSON[domain.HTMLLesson](ctx, tx, `select `+htmlLessonJSON+htmlFrom+`where `+htmlAccess+` and l.status='published' and v.id=l.published_version_id and l.id=$4 for share of l,v`, actor.OrganizationID, actor.ID, string(actor.Role), id)
	if err != nil {
		return out, err
	}
	var assignment string
	err = tx.QueryRow(ctx, `insert into assignments(id,organization_id,tutor_id,student_id,title,subject,topic,body,due_at,max_score,status) values(gen_random_uuid(),$1,$2,$3,$4,'Математика','HTML-урок',$5,$6,100,'assigned') returning id::text`, actor.OrganizationID, actor.ID, in.StudentID, l.Version.Title, "HTML-урок. Результаты встроенного кода не являются проверенными ответами. Работа оценивается преподавателем вручную.", in.DueAt).Scan(&assignment)
	if err != nil {
		return out, err
	}
	if _, err = tx.Exec(ctx, `insert into html_assignment_links(assignment_id,lesson_id,version_id,request_id,payload) values($1,$2,$3,$4,$5)`, assignment, id, l.Version.ID, in.RequestID, payload); err != nil {
		return out, err
	}
	if err = insertNotification(ctx, tx, actor.OrganizationID, in.StudentID, "Новый HTML-урок", l.Version.Title, "/tasks/"+assignment, "html-assignment:"+assignment); err != nil {
		return out, err
	}
	if err = tx.Commit(ctx); err != nil {
		return out, err
	}
	return s.GetAssignment(ctx, actor, assignment)
}

func (s *Store) ConvertHTMLLesson(ctx context.Context, actor domain.User, id, version string) (domain.LearningLesson, error) {
	l, err := s.GetHTMLLesson(ctx, actor, id, version, "")
	if err != nil {
		return domain.LearningLesson{}, err
	}
	source, err := s.HTMLOriginal(ctx, actor, id, l.Version.ID)
	if err != nil {
		return domain.LearningLesson{}, err
	}
	_, text, err := htmlcontent.Inspect(source)
	if err != nil {
		return domain.LearningLesson{}, err
	}
	in := domain.LearningLessonInput{TopicID: l.Version.TopicID, Title: l.Version.Title, Summary: l.Version.Summary, Content: text, Track: l.Version.Track, GradeMin: l.Version.GradeMin, GradeMax: l.Version.GradeMax, Difficulty: l.Version.Difficulty}
	if err = validateLearningLesson(&in); err != nil {
		return domain.LearningLesson{}, err
	}
	var draftID string
	err = s.catalogWrite(ctx, actor, l.CourseID, func(tx pgx.Tx) error {
		var err error
		draftID, err = insertLearningLesson(ctx, tx, actor, l.CourseID, in)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `insert into html_conversions(learning_lesson_id,html_version_id) values($1,$2)`, draftID, l.Version.ID)
		return err
	})
	if err != nil {
		return domain.LearningLesson{}, err
	}
	return s.GetLearningLesson(ctx, actor, draftID)
}

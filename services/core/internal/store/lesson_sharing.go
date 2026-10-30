package store

import (
	"context"
	"errors"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"tutor-platform/services/core/internal/apperr"
	"tutor-platform/services/core/internal/domain"
	"tutor-platform/services/core/internal/platform"
)

const shareJSON = `jsonb_build_object('id',g.id,'lesson_id',g.lesson_id,'version_id',g.version_id,'version_number',v.number,'title',v.title,'author_name',g.author_name,'recipient_email',recipient.email,'allow_copy',g.allow_copy,'license',g.license,'created_at',g.created_at,'revoked_at',g.revoked_at)`
const shareFrom = ` from lesson_shares g join learning_lessons l on l.id=g.lesson_id join courses c on c.id=l.course_id join lesson_versions v on v.id=g.version_id join users recipient on recipient.id=g.recipient_id `
const incomingShareAccess = `g.organization_id=$1 and g.recipient_id=$2 and $3='tutor' and g.revoked_at is null and l.status='published' and not l.assignment_only and c.status='published' and v.sealed and ` + catalogTopicPublished

func shareFilter(f *domain.CatalogFilter) error {
	if err := ValidateCatalogFilter(f); err != nil {
		return err
	}
	if f.Track != "" || f.Grade != 0 || f.Difficulty != 0 || f.AuthorID != "" || f.CourseID != "" || f.TopicID != "" || f.Status != "" {
		return apperr.ErrBadRequest
	}
	return nil
}

func (s *Store) CreateLessonShare(ctx context.Context, actor domain.User, lessonID string, in domain.LessonShareInput) (domain.LessonShare, error) {
	courseID, err := s.learningCourse(ctx, actor, lessonID)
	if err != nil {
		return domain.LessonShare{}, err
	}
	email := strings.ToLower(strings.TrimSpace(in.RecipientEmail))
	if !in.RightsConfirmed || len(email) < 3 || len(email) > 254 || !strings.Contains(email, "@") {
		return domain.LessonShare{}, apperr.ErrBadRequest
	}
	var out domain.LessonShare
	err = s.catalogWrite(ctx, actor, courseID, func(tx pgx.Tx) error {
		var versionID, recipientID, authorName string
		err := tx.QueryRow(ctx, `select v.id::text,recipient.id::text,author.display_name from learning_lessons l join courses c on c.id=l.course_id join lesson_versions v on v.id=l.published_version_id join users author on author.id=l.author_id join users recipient on recipient.organization_id=l.organization_id and recipient.email=$2 and recipient.role='tutor' and recipient.id<>l.author_id where l.id=$1 and l.status='published' and not l.assignment_only and c.status='published' and v.sealed and `+catalogTopicPublished+` and not exists(select 1 from learning_lesson_provenance p where p.lesson_id=l.id) and not exists(select 1 from exercises e where e.version_id=v.id and e.media<>'[]'::jsonb)`, lessonID, email).Scan(&versionID, &recipientID, &authorName)
		if err != nil {
			return err
		}
		var existingID, existingVersion string
		var existingCopy bool
		err = tx.QueryRow(ctx, `select id::text,version_id::text,allow_copy from lesson_shares where lesson_id=$1 and recipient_id=$2 and revoked_at is null`, lessonID, recipientID).Scan(&existingID, &existingVersion, &existingCopy)
		if err == nil {
			if existingVersion != versionID || existingCopy != in.AllowCopy {
				return apperr.ErrConflict
			}
			out, err = catalogJSON[domain.LessonShare](ctx, tx, `select `+shareJSON+shareFrom+`where g.id=$1`, existingID)
			return err
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		var count int
		if err = tx.QueryRow(ctx, `select count(*) from lesson_shares where lesson_id=$1 and revoked_at is null`, lessonID).Scan(&count); err != nil {
			return err
		}
		if count >= 1000 {
			return apperr.ErrInvalidState
		}
		license := "view_only"
		if in.AllowCopy {
			license = "personal_copy_permission"
		}
		var id string
		err = tx.QueryRow(ctx, `insert into lesson_shares(organization_id,lesson_id,version_id,grantor_id,recipient_id,author_name,allow_copy,license) values($1,$2,$3,$4,$5,$6,$7,$8) returning id::text`, actor.OrganizationID, lessonID, versionID, actor.ID, recipientID, authorName, in.AllowCopy, license).Scan(&id)
		if err != nil {
			return err
		}
		out, err = catalogJSON[domain.LessonShare](ctx, tx, `select `+shareJSON+shareFrom+`where g.id=$1`, id)
		return err
	})
	return out, err
}

func (s *Store) ListLessonShares(ctx context.Context, actor domain.User, lessonID string, f domain.CatalogFilter) (domain.CatalogPage[domain.LessonShare], error) {
	if actor.Role != domain.RoleTutor {
		return domain.CatalogPage[domain.LessonShare]{}, apperr.ErrForbidden
	}
	if err := shareFilter(&f); err != nil {
		return domain.CatalogPage[domain.LessonShare]{}, err
	}
	predicate := incomingShareAccess + ` and $4::text=''`
	if lessonID != "" {
		if _, err := s.learningCourse(ctx, actor, lessonID); err != nil {
			return domain.CatalogPage[domain.LessonShare]{}, err
		}
		predicate = `g.organization_id=$1 and g.grantor_id=$2 and $3='tutor' and g.lesson_id=nullif($4,'')::uuid`
	}
	return catalogJSON[domain.CatalogPage[domain.LessonShare]](ctx, s.db, `with eligible as(select g.id,g.created_at,`+shareJSON+` item`+shareFrom+`where `+predicate+` and ($5='' or strpos(lower(v.title),lower($5))>0)) select jsonb_build_object('items',coalesce(jsonb_agg(item order by created_at desc,id),'[]'::jsonb),'total',(select count(*) from eligible),'limit',$6::int,'offset',$7::int) from(select * from eligible order by created_at desc,id limit $6 offset $7)page`, actor.OrganizationID, actor.ID, string(actor.Role), lessonID, f.Query, f.Limit, f.Offset)
}

func (s *Store) GetSharedLesson(ctx context.Context, actor domain.User, id string) (domain.SharedLesson, error) {
	if actor.Role != domain.RoleTutor {
		return domain.SharedLesson{}, apperr.ErrForbidden
	}
	if !platform.ValidUUID(id) {
		return domain.SharedLesson{}, apperr.ErrBadRequest
	}
	return catalogJSON[domain.SharedLesson](ctx, s.db, `select jsonb_build_object('share',`+shareJSON+`,'lesson',`+catalogLessonJSON+`)`+shareFrom+`join users u on u.id=l.author_id where `+incomingShareAccess+` and g.id=$4`, append(catalogArgs(actor), id)...)
}

func (s *Store) RevokeLessonShare(ctx context.Context, actor domain.User, lessonID, id string) error {
	courseID, err := s.learningCourse(ctx, actor, lessonID)
	if err != nil {
		return err
	}
	if !platform.ValidUUID(id) {
		return apperr.ErrBadRequest
	}
	return s.catalogWrite(ctx, actor, courseID, func(tx pgx.Tx) error {
		var revoked bool
		err := tx.QueryRow(ctx, `select revoked_at is not null from lesson_shares where id=$1 and lesson_id=$2 and grantor_id=$3 and organization_id=$4 for update`, id, lessonID, actor.ID, actor.OrganizationID).Scan(&revoked)
		if err != nil || revoked {
			return err
		}
		_, err = tx.Exec(ctx, `update lesson_shares set revoked_at=clock_timestamp() where id=$1`, id)
		return err
	})
}

func (s *Store) CopySharedLesson(ctx context.Context, actor domain.User, shareID string, in domain.LessonCopyInput) (domain.LearningLesson, error) {
	if actor.Role != domain.RoleTutor {
		return domain.LearningLesson{}, apperr.ErrForbidden
	}
	for _, id := range []string{shareID, in.RequestID, in.CourseID, in.TopicID} {
		if !platform.ValidUUID(id) {
			return domain.LearningLesson{}, apperr.ErrBadRequest
		}
	}
	in.Title = strings.TrimSpace(in.Title)
	if !catalogTitle(in.Title) {
		return domain.LearningLesson{}, apperr.ErrBadRequest
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return domain.LearningLesson{}, err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `select pg_advisory_xact_lock(hashtextextended($1,82374125))`, actor.ID+in.RequestID); err != nil {
		return domain.LearningLesson{}, err
	}
	var priorID, priorShare, priorCourse, priorTopic, priorTitle string
	err = tx.QueryRow(ctx, `select lesson_id::text,share_id::text,course_id::text,topic_id::text,title from lesson_copy_requests where recipient_id=$1 and request_id=$2`, actor.ID, in.RequestID).Scan(&priorID, &priorShare, &priorCourse, &priorTopic, &priorTitle)
	if err == nil {
		if priorShare != shareID || priorCourse != in.CourseID || priorTopic != in.TopicID || priorTitle != in.Title {
			return domain.LearningLesson{}, apperr.ErrConflict
		}
		if err = tx.Commit(ctx); err != nil {
			return domain.LearningLesson{}, err
		}
		return s.GetLearningLesson(ctx, actor, priorID)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return domain.LearningLesson{}, err
	}
	var sourceCourse string
	err = tx.QueryRow(ctx, `select l.course_id::text from lesson_shares g join learning_lessons l on l.id=g.lesson_id where g.id=$1 and g.recipient_id=$2 and g.organization_id=$3 and g.revoked_at is null`, shareID, actor.ID, actor.OrganizationID).Scan(&sourceCourse)
	if err != nil {
		return domain.LearningLesson{}, catalogError(err)
	}
	// Grant, publication and destination edits serialize on courses in a stable order.
	ids := []string{sourceCourse, in.CourseID}
	sort.Strings(ids)
	for _, id := range ids {
		var locked string
		if err = tx.QueryRow(ctx, `select id::text from courses where id=$1 and organization_id=$2 for update`, id, actor.OrganizationID).Scan(&locked); err != nil {
			return domain.LearningLesson{}, catalogError(err)
		}
	}
	var target string
	err = tx.QueryRow(ctx, `select c.id::text from courses c join topics t on t.course_id=c.id where c.id=$1 and c.organization_id=$2 and c.author_id=$3 and t.id=$4 and t.status<>'archived' and c.status<>'archived'`, in.CourseID, actor.OrganizationID, actor.ID, in.TopicID).Scan(&target)
	if err != nil {
		return domain.LearningLesson{}, catalogError(err)
	}
	shared, err := catalogJSON[domain.SharedLesson](ctx, tx, `select jsonb_build_object('share',`+shareJSON+`,'lesson',`+catalogLessonJSON+`)`+shareFrom+`join users u on u.id=l.author_id where `+incomingShareAccess+` and g.id=$4`, append(catalogArgs(actor), shareID)...)
	if err != nil {
		return domain.LearningLesson{}, err
	}
	if !shared.Share.AllowCopy {
		return domain.LearningLesson{}, apperr.ErrForbidden
	}
	input := shared.Lesson.Version.LearningLessonInput
	input.TopicID = in.TopicID
	input.Title = in.Title
	id, err := insertLearningLesson(ctx, tx, actor, in.CourseID, input)
	if err != nil {
		return domain.LearningLesson{}, catalogError(err)
	}
	var draftID string
	if err = tx.QueryRow(ctx, `select id::text from lesson_versions where lesson_id=$1 and number=0`, id).Scan(&draftID); err != nil {
		return domain.LearningLesson{}, err
	}
	// Generate new public exercise keys; private answers never cross an HTTP response.
	for _, source := range shared.Lesson.Version.Exercises {
		if len(source.Media) > 0 {
			return domain.LearningLesson{}, apperr.ErrInvalidState
		}
		var newID, oldID string
		err = tx.QueryRow(ctx, `insert into exercises(version_id,kind,prompt,options,difficulty,position,config,media) select $1,kind,prompt,options,difficulty,position,config,media from exercises where version_id=$2 and exercise_key=$3 returning id::text`, draftID, shared.Share.VersionID, source.ID).Scan(&newID)
		if err != nil {
			return domain.LearningLesson{}, err
		}
		if err = tx.QueryRow(ctx, `select id::text from exercises where version_id=$1 and exercise_key=$2`, shared.Share.VersionID, source.ID).Scan(&oldID); err != nil {
			return domain.LearningLesson{}, err
		}
		if _, err = tx.Exec(ctx, `insert into exercise_answer_keys(exercise_id,answer,explanation,hints) select $1,answer,explanation,hints from exercise_answer_keys where exercise_id=$2`, newID, oldID); err != nil {
			return domain.LearningLesson{}, err
		}
	}
	_, err = tx.Exec(ctx, `insert into learning_lesson_provenance(lesson_id,source_lesson_id,source_version_id,source_version_number,original_author_id,original_author_name,source_title,license) values($1,$2,$3,$4,$5,$6,$7,$8)`, id, shared.Lesson.ID, shared.Share.VersionID, shared.Share.VersionNumber, shared.Lesson.AuthorID, shared.Share.AuthorName, shared.Share.Title, shared.Share.License)
	if err != nil {
		return domain.LearningLesson{}, catalogError(err)
	}
	_, err = tx.Exec(ctx, `insert into lesson_copy_requests(recipient_id,request_id,share_id,course_id,topic_id,title,lesson_id) values($1,$2,$3,$4,$5,$6,$7)`, actor.ID, in.RequestID, shareID, in.CourseID, in.TopicID, in.Title, id)
	if err != nil {
		return domain.LearningLesson{}, catalogError(err)
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.LearningLesson{}, err
	}
	return s.GetLearningLesson(ctx, actor, id)
}

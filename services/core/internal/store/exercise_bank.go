package store

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"tutor-platform/services/core/internal/apperr"
	"tutor-platform/services/core/internal/domain"
	"tutor-platform/services/core/internal/platform"
)

const bankFrom = ` from bank_exercises b join exercises e on e.id=b.exercise_id
 join lesson_versions v on v.id=e.version_id join learning_lessons l on l.id=v.lesson_id
 join courses c on c.id=l.course_id join users u on u.id=b.author_id `
const bankAccess = `b.organization_id=$1 and $3='tutor' and ` + catalogCourseAccess + `
 and not l.assignment_only and l.status='published' and c.status='published' and v.sealed and ` + catalogTopicPublished
const bankJSON = `(to_jsonb(b)-'organization_id'-'exercise_id'-'search_text'-'search_vector') ||
 jsonb_build_object('author_name',u.display_name,'lesson_id',l.id,'version_id',v.id,'exercise',
 jsonb_build_object('id',e.exercise_key,'kind',e.kind,'prompt',e.prompt,'options',e.options,
 'difficulty',e.difficulty,'position',e.position,'config',e.config,'media',e.media,
 'hint_count',coalesce((select jsonb_array_length(k.hints) from exercise_answer_keys k where k.exercise_id=e.id),0)))`
const bankWhere = ` and ($4='' or b.search_vector @@ plainto_tsquery('russian',$4))
 and ($5='' or b.direction=$5) and ($6::int=0 or b.grade=$6) and ($7::int=0 or b.exam_number=$7)
 and ($8='' or lower(b.topic)=lower($8)) and ($9='' or lower(b.subtopic)=lower($9))
 and ($10::int=0 or e.difficulty=$10) and ($11='' or e.kind=$11)`

func bankDirection(v string) bool {
	return v == "school" || v == "olympiad" || v == "oge" || v == "ege_basic" || v == "ege_profile"
}
func bankKind(v string) bool {
	switch v {
	case "text", "numeric", "single_choice", "multiple_choice", "long_text", "number_line", "grid_select", "matching", "ordering", "geometry":
		return true
	}
	return false
}
func ValidateBankFilter(f *domain.BankFilter) error {
	f.Query, f.Topic, f.Subtopic = strings.TrimSpace(f.Query), strings.TrimSpace(f.Topic), strings.TrimSpace(f.Subtopic)
	if f.Limit == 0 {
		f.Limit = 20
	}
	if f.Limit < 1 || f.Limit > 100 || f.Offset < 0 || f.Offset > 100000 || len(f.Query) > 200 || len(f.Topic) > 400 || len(f.Subtopic) > 400 || f.Grade != 0 && (f.Grade < 5 || f.Grade > 11) || f.ExamNumber < 0 || f.ExamNumber > 100 || f.Difficulty < 0 || f.Difficulty > 5 || f.Direction != "" && !bankDirection(f.Direction) || f.Kind != "" && !bankKind(f.Kind) {
		return apperr.ErrBadRequest
	}
	return nil
}
func bankArgs(actor domain.User, f domain.BankFilter) []any {
	return append(catalogArgs(actor), f.Query, f.Direction, f.Grade, f.ExamNumber, f.Topic, f.Subtopic, f.Difficulty, f.Kind)
}
func (s *Store) ListBankExercises(ctx context.Context, actor domain.User, f domain.BankFilter) (domain.CatalogPage[domain.BankExercise], error) {
	var out domain.CatalogPage[domain.BankExercise]
	if actor.Role != domain.RoleTutor {
		return out, apperr.ErrForbidden
	}
	if err := ValidateBankFilter(&f); err != nil {
		return out, err
	}
	return catalogJSON[domain.CatalogPage[domain.BankExercise]](ctx, s.db, `with eligible as (select b.id,b.created_at,`+bankJSON+` item`+bankFrom+`where `+bankAccess+bankWhere+`)
 select jsonb_build_object('items',coalesce(jsonb_agg(item order by created_at desc,id),'[]'::jsonb),'total',(select count(*) from eligible),'limit',$12::int,'offset',$13::int)
 from(select * from eligible order by created_at desc,id limit $12 offset $13) page`, append(bankArgs(actor, f), f.Limit, f.Offset)...)
}
func (s *Store) BankFacets(ctx context.Context, actor domain.User, f domain.BankFilter) (domain.BankFacets, error) {
	if actor.Role != domain.RoleTutor {
		return domain.BankFacets{}, apperr.ErrForbidden
	}
	if err := ValidateBankFilter(&f); err != nil {
		return domain.BankFacets{}, err
	}
	topic := f.Topic
	f.Topic, f.Subtopic = "", ""
	return catalogJSON[domain.BankFacets](ctx, s.db, `with eligible as(select b.topic,b.subtopic`+bankFrom+`where `+bankAccess+bankWhere+`)
 select jsonb_build_object('topics',coalesce((select jsonb_agg(topic order by topic) from(select distinct topic from eligible)t),'[]'::jsonb),
 'subtopics',coalesce((select jsonb_agg(subtopic order by subtopic) from(select distinct subtopic from eligible where subtopic<>'' and ($12='' or lower(topic)=lower($12)))t),'[]'::jsonb))`, append(bankArgs(actor, f), topic)...)
}
func bankGet(ctx context.Context, q catalogQuerier, actor domain.User, id string) (domain.BankExercise, error) {
	return catalogJSON[domain.BankExercise](ctx, q, `select `+bankJSON+bankFrom+`where `+bankAccess+` and b.id=$4`, append(catalogArgs(actor), id)...)
}
func (s *Store) GetBankExercise(ctx context.Context, actor domain.User, id string) (domain.BankExercise, error) {
	if actor.Role != domain.RoleTutor {
		return domain.BankExercise{}, apperr.ErrForbidden
	}
	if !platform.ValidUUID(id) {
		return domain.BankExercise{}, apperr.ErrBadRequest
	}
	return bankGet(ctx, s.db, actor, id)
}
func (s *Store) BankAnswerKey(ctx context.Context, actor domain.User, id string) (domain.ExerciseAnswerKey, error) {
	if _, err := s.GetBankExercise(ctx, actor, id); err != nil {
		return domain.ExerciseAnswerKey{}, err
	}
	return catalogJSON[domain.ExerciseAnswerKey](ctx, s.db, `select jsonb_build_object('exercise_id',e.exercise_key,'answer',k.answer,'explanation',k.explanation,'hints',k.hints)`+bankFrom+`join exercise_answer_keys k on k.exercise_id=e.id where `+bankAccess+` and b.id=$4`, append(catalogArgs(actor), id)...)
}
func (s *Store) CreateBankExercise(ctx context.Context, actor domain.User, in domain.BankExerciseInput) (domain.BankExercise, error) {
	var out domain.BankExercise
	if actor.Role != domain.RoleTutor {
		return out, apperr.ErrForbidden
	}
	in.Topic, in.Subtopic = strings.TrimSpace(in.Topic), strings.TrimSpace(in.Subtopic)
	if !platform.ValidUUID(in.LessonID) || !platform.ValidUUID(in.ExerciseID) || in.Grade < 5 || in.Grade > 11 || !bankDirection(in.Direction) || !catalogTitle(in.Topic) || len(in.Subtopic) > 400 || strings.ContainsRune(in.Subtopic, 0) || !validPlanText(in.Source, 1000) || !validPlanText(in.Attribution, 1000) || !validPlanText(in.License, 1000) || (in.RightsStatus != "own" && in.RightsStatus != "permission" && in.RightsStatus != "open_license") {
		return out, apperr.ErrBadRequest
	}
	exam := in.Direction == "oge" || strings.HasPrefix(in.Direction, "ege_")
	if exam != (in.ExamNumber != nil) || in.ExamNumber != nil && (*in.ExamNumber < 1 || *in.ExamNumber > 100) {
		return out, apperr.ErrBadRequest
	}
	courseID, err := s.learningCourse(ctx, actor, in.LessonID)
	if err != nil {
		return out, err
	}
	var id string
	err = s.catalogWrite(ctx, actor, courseID, func(tx pgx.Tx) error {
		var rowID, prompt string
		err := tx.QueryRow(ctx, `select e.id::text,e.prompt`+catalogLessonFrom+`join exercises e on e.version_id=v.id where `+catalogLessonAccess+` and c.status='published' and l.status='published' and not l.assignment_only and v.id=l.published_version_id and v.sealed and `+catalogTopicPublished+` and l.id=$4 and e.exercise_key=$5 and $6::int between v.grade_min and v.grade_max`, append(catalogArgs(actor), in.LessonID, in.ExerciseID, in.Grade)...).Scan(&rowID, &prompt)
		if err != nil {
			return err
		}
		err = tx.QueryRow(ctx, `insert into bank_exercises(organization_id,author_id,exercise_id,grade,direction,topic,subtopic,exam_number,source,attribution,rights_status,license,search_text)
 values($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13) on conflict(author_id,exercise_id) do nothing returning id::text`, actor.OrganizationID, actor.ID, rowID, in.Grade, in.Direction, in.Topic, in.Subtopic, in.ExamNumber, in.Source, in.Attribution, in.RightsStatus, in.License, prompt).Scan(&id)
		if errors.Is(err, pgx.ErrNoRows) {
			return apperr.ErrConflict
		}
		return err
	})
	if err != nil {
		return out, err
	}
	return s.GetBankExercise(ctx, actor, id)
}

const bankCollectionJSON = `jsonb_build_object('id',bc.id,'lesson_id',bc.lesson_id,'version_id',bc.version_id,'title',v.title,
 'exercise_ids',bc.exercise_keys,'provenance',bc.provenance,'version',` + catalogVersionJSON + `,'created_at',bc.created_at)`
const bankCollectionFrom = ` from bank_collections bc join lesson_versions v on v.id=bc.version_id where bc.organization_id=$1 and bc.tutor_id=$2 `

func (s *Store) GetBankCollection(ctx context.Context, actor domain.User, id string) (domain.BankCollection, error) {
	if actor.Role != domain.RoleTutor {
		return domain.BankCollection{}, apperr.ErrForbidden
	}
	if !platform.ValidUUID(id) {
		return domain.BankCollection{}, apperr.ErrBadRequest
	}
	return catalogJSON[domain.BankCollection](ctx, s.db, `select `+bankCollectionJSON+bankCollectionFrom+`and bc.id=$3`, actor.OrganizationID, actor.ID, id)
}
func (s *Store) ListBankCollections(ctx context.Context, actor domain.User, f domain.CatalogFilter) (domain.CatalogPage[domain.BankCollection], error) {
	if actor.Role != domain.RoleTutor {
		return domain.CatalogPage[domain.BankCollection]{}, apperr.ErrForbidden
	}
	if err := ValidateCatalogFilter(&f); err != nil {
		return domain.CatalogPage[domain.BankCollection]{}, err
	}
	return catalogJSON[domain.CatalogPage[domain.BankCollection]](ctx, s.db, `with eligible as(select bc.id,bc.created_at,jsonb_build_object('id',bc.id,'title',v.title,'lesson_id',bc.lesson_id,'version_id',bc.version_id,'exercise_ids',bc.exercise_keys,'provenance','[]'::jsonb,'created_at',bc.created_at) item`+bankCollectionFrom+`)
 select jsonb_build_object('items',coalesce(jsonb_agg(item order by created_at desc,id),'[]'::jsonb),'total',(select count(*) from eligible),'limit',$3::int,'offset',$4::int)
 from(select * from eligible order by created_at desc,id limit $3 offset $4)p`, actor.OrganizationID, actor.ID, f.Limit, f.Offset)
}
func (s *Store) CreateBankCollection(ctx context.Context, actor domain.User, in domain.BankCollectionInput) (domain.BankCollection, error) {
	var out domain.BankCollection
	if actor.Role != domain.RoleTutor {
		return out, apperr.ErrForbidden
	}
	in.RequestID = strings.ToLower(in.RequestID)
	if !platform.ValidUUID(in.RequestID) || !platform.ValidUUID(in.CourseID) || !platform.ValidUUID(in.TopicID) || !catalogTitle(in.Title) || len(in.BankIDs) < 1 || len(in.BankIDs) > 100 {
		return out, apperr.ErrBadRequest
	}
	seen := map[string]bool{}
	for i, id := range in.BankIDs {
		id = strings.ToLower(id)
		if !platform.ValidUUID(id) || seen[id] {
			return out, apperr.ErrBadRequest
		}
		seen[id] = true
		in.BankIDs[i] = id
	}
	payload, _ := json.Marshal(in)
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `select pg_advisory_xact_lock(hashtextextended($1,10))`, actor.ID+in.RequestID); err != nil {
		return out, err
	}
	var old string
	var matches bool
	err = tx.QueryRow(ctx, `select id::text,payload=$3::jsonb from bank_collections where tutor_id=$1 and request_id=$2`, actor.ID, in.RequestID, payload).Scan(&old, &matches)
	if err == nil {
		if !matches {
			return out, apperr.ErrConflict
		}
		if err = tx.Commit(ctx); err != nil {
			return out, err
		}
		return s.GetBankCollection(ctx, actor, old)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return out, err
	}
	// Course locks use a stable order, including foreign shared sources and the destination.
	rows, err := tx.Query(ctx, `select c0.id from courses c0 where (c0.id=$4 and c0.organization_id=$1 and c0.author_id=$2)
 or c0.id in(select c.id`+bankFrom+`where `+bankAccess+` and b.id=any($5::uuid[])) order by c0.id for update`, append(catalogArgs(actor), in.CourseID, in.BankIDs)...)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return out, err
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, err
	}
	var ready bool
	err = tx.QueryRow(ctx, `select c.author_id=$2 and c.organization_id=$3 and c.status='published' and not exists(with recursive ancestors as(select id,parent_id,status from topics where id=$4 and course_id=c.id union all select t.id,t.parent_id,t.status from topics t join ancestors a on a.parent_id=t.id)select 1 from ancestors where status<>'published') and exists(select 1 from topics where id=$4 and course_id=c.id) from courses c where c.id=$1`, in.CourseID, actor.ID, actor.OrganizationID, in.TopicID).Scan(&ready)
	if err != nil {
		return out, catalogError(err)
	}
	if !ready {
		return out, apperr.ErrNotFound
	}
	items := make([]domain.BankExercise, 0, len(in.BankIDs))
	minGrade, maxGrade := 11, 5
	for _, id := range in.BankIDs {
		item, e := bankGet(ctx, tx, actor, id)
		if e != nil {
			return out, e
		}
		items = append(items, item)
		minGrade = min(minGrade, item.Grade)
		maxGrade = max(maxGrade, item.Grade)
	}
	lessonID, err := insertLearningLesson(ctx, tx, actor, in.CourseID, domain.LearningLessonInput{TopicID: in.TopicID, Title: strings.TrimSpace(in.Title), Summary: "Подборка упражнений", Content: "", Track: "school", GradeMin: minGrade, GradeMax: maxGrade, Difficulty: 1})
	if err != nil {
		return out, catalogError(err)
	}
	var versionID, draftID string
	err = tx.QueryRow(ctx, `insert into lesson_versions(course_id,lesson_id,topic_id,number,title,summary,content,track,grade_min,grade_max,difficulty,published_at)select course_id,lesson_id,topic_id,1,title,summary,content,track,grade_min,grade_max,difficulty,now() from lesson_versions where lesson_id=$1 and number=0 returning id::text`, lessonID).Scan(&versionID)
	if err != nil {
		return out, err
	}
	if err = tx.QueryRow(ctx, `select id::text from lesson_versions where lesson_id=$1 and number=0`, lessonID).Scan(&draftID); err != nil {
		return out, err
	}
	keys := make([]string, 0, len(items))
	for pos, item := range items {
		key, e := platform.NewUUID()
		if e != nil {
			return out, e
		}
		keys = append(keys, key)
		for _, dst := range []string{draftID, versionID} {
			var copied string
			err = tx.QueryRow(ctx, `insert into exercises(version_id,exercise_key,kind,prompt,options,difficulty,position,config,media) select $1,$2,e.kind,e.prompt,e.options,e.difficulty,$3,e.config,e.media from exercises e join bank_exercises b on b.exercise_id=e.id where b.id=$4 returning id::text`, dst, key, pos, item.ID).Scan(&copied)
			if err != nil {
				return out, err
			}
			if _, err = tx.Exec(ctx, `insert into exercise_answer_keys(exercise_id,answer,explanation,hints) select $1,k.answer,k.explanation,k.hints from exercise_answer_keys k join bank_exercises b on b.exercise_id=k.exercise_id where b.id=$2`, copied, item.ID); err != nil {
				return out, err
			}
		}
	}
	if _, err = tx.Exec(ctx, `update lesson_versions set sealed=true where id=$1`, versionID); err != nil {
		return out, err
	}
	if _, err = tx.Exec(ctx, `update learning_lessons set assignment_only=true,status='published',published_version_id=$2,published_revision=draft_revision where id=$1`, lessonID, versionID); err != nil {
		return out, err
	}
	provenance, _ := json.Marshal(items)
	var id string
	err = tx.QueryRow(ctx, `insert into bank_collections(organization_id,tutor_id,lesson_id,version_id,exercise_keys,provenance,request_id,payload) values($1,$2,$3,$4,$5,$6,$7,$8) returning id::text`, actor.OrganizationID, actor.ID, lessonID, versionID, keys, provenance, in.RequestID, payload).Scan(&id)
	if err != nil {
		return out, catalogError(err)
	}
	if err = tx.Commit(ctx); err != nil {
		return out, err
	}
	return s.GetBankCollection(ctx, actor, id)
}

func bankChildRequest(batch, student string) string {
	h := sha256.Sum256([]byte("repet-bank-assignment:" + batch + ":" + student))
	h[6] = (h[6] & 15) | 80
	h[8] = (h[8] & 63) | 128
	return fmt.Sprintf("%x-%x-%x-%x-%x", h[:4], h[4:6], h[6:8], h[8:10], h[10:16])
}
func (s *Store) AssignBankCollection(ctx context.Context, actor domain.User, id string, in domain.BankAssignInput) ([]domain.BankAssignResult, error) {
	collection, err := s.GetBankCollection(ctx, actor, id)
	if err != nil {
		return nil, err
	}
	in.RequestID = strings.ToLower(in.RequestID)
	if !platform.ValidUUID(in.RequestID) || len(in.StudentIDs) < 1 || len(in.StudentIDs) > 50 || (in.Mode != "practice" && in.Mode != "exam") {
		return nil, apperr.ErrBadRequest
	}
	seen := map[string]bool{}
	for i, sid := range in.StudentIDs {
		sid = strings.ToLower(sid)
		if !platform.ValidUUID(sid) || seen[sid] {
			return nil, apperr.ErrBadRequest
		}
		seen[sid] = true
		in.StudentIDs[i] = sid
	}
	sort.Strings(in.StudentIDs)
	payload, _ := json.Marshal(in)
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `select pg_advisory_xact_lock(hashtextextended($1,11))`, actor.ID+in.RequestID); err != nil {
		return nil, err
	}
	var matches bool
	err = tx.QueryRow(ctx, `select collection_id=$3 and payload=$4::jsonb from bank_assignment_batches where tutor_id=$1 and request_id=$2`, actor.ID, in.RequestID, id, payload).Scan(&matches)
	if errors.Is(err, pgx.ErrNoRows) {
		if !in.DueAt.After(time.Now()) {
			return nil, apperr.ErrBadRequest
		}
		var count int
		if err = tx.QueryRow(ctx, `select count(*) from relations where organization_id=$1 and tutor_id=$2 and student_id=any($3::uuid[]) and status='active'`, actor.OrganizationID, actor.ID, in.StudentIDs).Scan(&count); err != nil {
			return nil, err
		}
		if count != len(in.StudentIDs) {
			return nil, apperr.ErrNotFound
		}
		_, err = tx.Exec(ctx, `insert into bank_assignment_batches(tutor_id,request_id,collection_id,payload) values($1,$2,$3,$4)`, actor.ID, in.RequestID, id, payload)
	} else if err == nil && !matches {
		return nil, apperr.ErrConflict
	}
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	// Each child has a deterministic nonce. A partial batch is explicit and safe to retry.
	results := make([]domain.BankAssignResult, 0, len(in.StudentIDs))
	for _, sid := range in.StudentIDs {
		input := domain.MaterialAssignmentInput{RequestID: bankChildRequest(in.RequestID, sid), StudentID: sid, DueAt: in.DueAt, Mode: in.Mode}
		childPayload, _ := json.Marshal(struct {
			CollectionID string                         `json:"collection_id"`
			Input        domain.MaterialAssignmentInput `json:"input"`
		}{id, input})
		item, e := s.materialReplay(ctx, actor, input.RequestID, childPayload)
		if e == nil && item.ID == "" {
			item, e = s.insertMaterialAssignment(ctx, actor, input, collection.Version, collection.ExerciseIDs, "", childPayload)
		}
		result := domain.BankAssignResult{StudentID: sid}
		if e != nil {
			result.Error = "Не удалось назначить. Проверьте связь с учеником и повторите запрос."
		} else {
			result.Assignment = &item
		}
		results = append(results, result)
	}
	return results, nil
}

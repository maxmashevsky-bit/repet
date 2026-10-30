package store

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"image"
	"image/draw"
	"image/png"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"tutor-platform/services/core/internal/apperr"
	"tutor-platform/services/core/internal/domain"
	"tutor-platform/services/core/internal/pdfextract"
	"tutor-platform/services/core/internal/platform"
)

const pdfOwner = `p.organization_id=$2 and p.author_id=$3`
const pdfImportMetadataJSON = `(to_jsonb(p)-'organization_id'-'author_id'-'source_key'-'request_id'-'request_hash'-'lease_until'-'lease_token'-'extraction_attempts')`
const pdfImportJSON = pdfImportMetadataJSON + ` || jsonb_build_object(
 'pages',coalesce((select jsonb_agg(to_jsonb(pg)-'import_id'-'reviewed_by' order by number) from pdf_pages pg where pg.import_id=p.id),'[]'::jsonb),
 'assets',coalesce((select jsonb_agg(to_jsonb(a)-'import_id'-'object_key'-'sha256' order by page_number,kind,ordinal) from pdf_assets a where a.import_id=p.id),'[]'::jsonb),
 'lesson_ids',coalesce((select jsonb_agg(lesson_id) from pdf_lesson_sources where import_id=p.id),'[]'::jsonb))`

func (s *Store) ImportPDF(ctx context.Context, actor domain.User, in domain.PDFImportInput) (domain.PDFImport, error) {
	if actor.Role != domain.RoleTutor {
		return domain.PDFImport{}, apperr.ErrForbidden
	}
	if s.objects == nil || s.pdf == nil {
		return domain.PDFImport{}, apperr.ErrInvalidState
	}
	if !platform.ValidUUID(in.RequestID) || len(in.Filename) > 200 || filepath.Base(in.Filename) != in.Filename || strings.ContainsAny(in.Filename, "\\\x00") || !strings.EqualFold(filepath.Ext(in.Filename), ".pdf") ||
		(in.RightsStatus != "own" && in.RightsStatus != "permission" && in.RightsStatus != "open_license") || strings.TrimSpace(in.SourceLabel) == "" || len(in.SourceLabel) > 1000 || strings.TrimSpace(in.License) == "" || len(in.License) > 1000 ||
		len(in.DataBase64) > base64.StdEncoding.EncodedLen(pdfextract.MaxPDFBytes) {
		return domain.PDFImport{}, apperr.ErrBadRequest
	}
	data, err := base64.StdEncoding.Strict().DecodeString(in.DataBase64)
	if err != nil || len(data) > pdfextract.MaxPDFBytes || !bytes.HasPrefix(data, []byte("%PDF-")) {
		return domain.PDFImport{}, apperr.ErrBadRequest
	}
	payload, _ := json.Marshal(in)
	sum := sha256.Sum256(payload)
	hash := hex.EncodeToString(sum[:])
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return domain.PDFImport{}, err
	}
	defer tx.Rollback(ctx)
	// Per-author lock serializes quota checks and idempotent uploads, including concurrent requests.
	if _, err = tx.Exec(ctx, `select pg_advisory_xact_lock(hashtextextended($1,9))`, actor.ID); err != nil {
		return domain.PDFImport{}, err
	}
	var id, oldHash string
	err = tx.QueryRow(ctx, `select id::text,request_hash from pdf_imports where author_id=$1 and request_id=$2`, actor.ID, in.RequestID).Scan(&id, &oldHash)
	if err == nil {
		if oldHash != hash {
			return domain.PDFImport{}, apperr.ErrConflict
		}
		return s.getPDF(ctx, tx, actor, id)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return domain.PDFImport{}, err
	}
	var allowed bool
	if err = tx.QueryRow(ctx, `select count(*)<100 and coalesce(sum(source_bytes),0)+$2<=104857600 from pdf_imports where author_id=$1`, actor.ID, len(data)).Scan(&allowed); err != nil {
		return domain.PDFImport{}, err
	}
	if !allowed {
		return domain.PDFImport{}, apperr.ErrInvalidState
	}
	if err = s.pdfStorageQuota(ctx, tx, actor.ID, int64(len(data))); err != nil {
		return domain.PDFImport{}, err
	}
	obj, err := s.objects.Put(ctx, bytes.NewReader(data), pdfextract.MaxPDFBytes)
	if err != nil {
		return domain.PDFImport{}, err
	}
	committed := false
	defer func() {
		if !committed {
			s.cleanupPDFObjects(tx, actor.ID, []string{obj.Key})
		}
	}()
	err = tx.QueryRow(ctx, `insert into pdf_imports(organization_id,author_id,filename,source_key,sha256,source_bytes,rights_status,license,source_label,request_id,request_hash,status) values($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,'queued') returning id::text`, actor.OrganizationID, actor.ID, in.Filename, obj.Key, obj.SHA256, obj.Size, in.RightsStatus, in.License, in.SourceLabel, in.RequestID, hash).Scan(&id)
	if err != nil {
		return domain.PDFImport{}, catalogError(err)
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.PDFImport{}, err
	}
	committed = true
	s.recordAudit(ctx, actor, "pdf.imported", "pdf_import", id, map[string]any{"bytes": obj.Size})
	return s.GetPDFImport(ctx, actor, id)
}
func (s *Store) getPDF(ctx context.Context, q catalogQuerier, actor domain.User, id string) (domain.PDFImport, error) {
	if actor.Role != domain.RoleTutor {
		return domain.PDFImport{}, apperr.ErrForbidden
	}
	if !platform.ValidUUID(id) {
		return domain.PDFImport{}, apperr.ErrBadRequest
	}
	return catalogJSON[domain.PDFImport](ctx, q, `select `+pdfImportJSON+` from pdf_imports p where p.id=$1 and `+pdfOwner, id, actor.OrganizationID, actor.ID)
}
func (s *Store) GetPDFImport(ctx context.Context, actor domain.User, id string) (domain.PDFImport, error) {
	return s.getPDF(ctx, s.db, actor, id)
}
func (s *Store) ListPDFImports(ctx context.Context, actor domain.User, f domain.CatalogFilter) (domain.CatalogPage[domain.PDFImport], error) {
	if actor.Role != domain.RoleTutor {
		return domain.CatalogPage[domain.PDFImport]{}, apperr.ErrForbidden
	}
	if err := ValidateCatalogFilter(&f); err != nil {
		return domain.CatalogPage[domain.PDFImport]{}, err
	}
	return catalogJSON[domain.CatalogPage[domain.PDFImport]](ctx, s.db, `with eligible as(select p.id,p.created_at,`+pdfImportMetadataJSON+` || jsonb_build_object('pages','[]'::jsonb,'assets','[]'::jsonb,'lesson_ids','[]'::jsonb) item from pdf_imports p where p.organization_id=$1 and p.author_id=$2),page as(select * from eligible order by created_at desc,id limit $3 offset $4) select jsonb_build_object('items',coalesce((select jsonb_agg(item order by created_at desc,id) from page),'[]'::jsonb),'total',(select count(*) from eligible),'limit',$3::int,'offset',$4::int)`, actor.OrganizationID, actor.ID, f.Limit, f.Offset)
}
func (s *Store) PDFOriginal(ctx context.Context, actor domain.User, id string) (io.ReadCloser, error) {
	if _, err := s.GetPDFImport(ctx, actor, id); err != nil {
		return nil, err
	}
	if s.objects == nil {
		return nil, apperr.ErrInvalidState
	}
	var key string
	if err := s.db.QueryRow(ctx, `select source_key from pdf_imports where id=$1`, id).Scan(&key); err != nil {
		return nil, err
	}
	return s.objects.Open(ctx, key)
}

func (s *Store) CropPDFFigure(ctx context.Context, actor domain.User, id string, page, x, y, width, height int) (domain.PDFImport, error) {
	if _, err := s.GetPDFImport(ctx, actor, id); err != nil {
		return domain.PDFImport{}, err
	}
	if s.objects == nil {
		return domain.PDFImport{}, apperr.ErrInvalidState
	}
	if page < 1 || page > 10 || x < 0 || y < 0 || width < 1 || height < 1 || x > 3000 || y > 3000 || width > 3000 || height > 3000 {
		return domain.PDFImport{}, apperr.ErrBadRequest
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return domain.PDFImport{}, err
	}
	defer tx.Rollback(ctx)
	var locked bool
	if err = tx.QueryRow(ctx, `select true from pdf_imports where id=$1 for update`, id).Scan(&locked); err != nil {
		return domain.PDFImport{}, err
	}
	var key string
	if err = tx.QueryRow(ctx, `select a.object_key from pdf_pages pg join pdf_assets a on a.id=pg.image_id where pg.import_id=$1 and pg.number=$2`, id, page).Scan(&key); err != nil {
		return domain.PDFImport{}, catalogError(err)
	}
	var ordinal int
	if err = tx.QueryRow(ctx, `select coalesce(max(ordinal),-1)+1 from pdf_assets where import_id=$1 and kind='figure'`, id).Scan(&ordinal); err != nil {
		return domain.PDFImport{}, err
	}
	if ordinal >= 64 {
		return domain.PDFImport{}, apperr.ErrInvalidState
	}
	body, err := s.objects.Open(ctx, key)
	if err != nil {
		return domain.PDFImport{}, err
	}
	img, err := png.Decode(body)
	body.Close()
	if err != nil {
		return domain.PDFImport{}, err
	}
	rect := image.Rect(x, y, x+width, y+height)
	if !rect.In(img.Bounds()) {
		return domain.PDFImport{}, apperr.ErrBadRequest
	}
	crop := image.NewNRGBA(image.Rect(0, 0, width, height))
	draw.Draw(crop, crop.Bounds(), img, rect.Min, draw.Src)
	var data bytes.Buffer
	if err = png.Encode(&data, crop); err != nil {
		return domain.PDFImport{}, err
	}
	if data.Len() > 2<<20 {
		return domain.PDFImport{}, apperr.ErrBadRequest
	}
	if err = s.pdfStorageQuota(ctx, tx, actor.ID, int64(data.Len())); err != nil {
		return domain.PDFImport{}, err
	}
	obj, err := s.objects.Put(ctx, bytes.NewReader(data.Bytes()), 2<<20)
	if err != nil {
		return domain.PDFImport{}, err
	}
	committed := false
	defer func() {
		if !committed {
			s.cleanupPDFObjects(tx, actor.ID, []string{obj.Key})
		}
	}()
	if _, err = tx.Exec(ctx, `insert into pdf_assets(import_id,kind,page_number,ordinal,object_key,sha256,bytes,width,height) values($1,'figure',$2,$3,$4,$5,$6,$7,$8)`, id, page, ordinal, obj.Key, obj.SHA256, obj.Size, width, height); err != nil {
		return domain.PDFImport{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.PDFImport{}, err
	}
	committed = true
	return s.GetPDFImport(ctx, actor, id)
}

// The author lock resolves uncertain COMMIT outcomes before deleting any unreferenced object.
func (s *Store) cleanupPDFObjects(original pgx.Tx, author string, keys []string) {
	if len(keys) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = original.Rollback(ctx)
	tx, err := s.db.Begin(ctx)
	if err != nil {
		slog.Error("PDF cleanup deferred", "error", err)
		return
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `select pg_advisory_xact_lock(hashtextextended($1,9))`, author); err != nil {
		slog.Error("PDF cleanup deferred", "error", err)
		return
	}
	for _, key := range keys {
		var referenced bool
		if err = tx.QueryRow(ctx, `select exists(select 1 from pdf_imports where source_key=$1 union all select 1 from pdf_assets where object_key=$1)`, key).Scan(&referenced); err != nil {
			slog.Error("PDF cleanup deferred", "error", err)
			return
		}
		if !referenced {
			if err = s.objects.Delete(ctx, key); err != nil {
				slog.Error("PDF object cleanup failed", "error", err)
			}
		}
	}
}
func (s *Store) pdfStorageQuota(ctx context.Context, tx pgx.Tx, author string, additional int64) error {
	if _, err := tx.Exec(ctx, `select pg_advisory_xact_lock(hashtextextended($1,9))`, author); err != nil {
		return err
	}
	var allowed bool
	err := tx.QueryRow(ctx, `select coalesce((select sum(a.bytes) from pdf_assets a join pdf_imports p on p.id=a.import_id where p.author_id=$1),0)+coalesce((select sum(source_bytes) from pdf_imports where author_id=$1),0)+$2<=536870912`, author, additional).Scan(&allowed)
	if err != nil {
		return err
	}
	if !allowed {
		return apperr.ErrInvalidState
	}
	return nil
}
func (s *Store) ReviewPDFPage(ctx context.Context, actor domain.User, id string, number int, text string) (domain.PDFImport, error) {
	if _, err := s.GetPDFImport(ctx, actor, id); err != nil {
		return domain.PDFImport{}, err
	}
	if number < 1 || number > pdfextract.MaxPages || len(text) > 100000 || !utf8.ValidString(text) || strings.ContainsRune(text, 0) {
		return domain.PDFImport{}, apperr.ErrBadRequest
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return domain.PDFImport{}, err
	}
	defer tx.Rollback(ctx)
	// Use the same course locks as catalog edits and publication before invalidating linked drafts.
	lockedCourses, err := tx.Query(ctx, `select c.id from courses c where c.id in(select l.course_id from learning_lessons l join pdf_lesson_sources src on src.lesson_id=l.id where src.import_id=$1) order by c.id for update`, id)
	if err != nil {
		return domain.PDFImport{}, err
	}
	for lockedCourses.Next() {
	}
	err = lockedCourses.Err()
	lockedCourses.Close()
	if err != nil {
		return domain.PDFImport{}, err
	}
	var valid bool
	if err = tx.QueryRow(ctx, `select not ocr_required or length(trim($3))>0 from pdf_pages where import_id=$1 and number=$2 for update`, id, number, text).Scan(&valid); err != nil {
		return domain.PDFImport{}, catalogError(err)
	}
	if !valid {
		return domain.PDFImport{}, apperr.ErrBadRequest
	}
	if _, err = tx.Exec(ctx, `update pdf_pages set corrected_text=$3,reviewed_at=now(),reviewed_by=$4 where import_id=$1 and number=$2`, id, number, text, actor.ID); err != nil {
		return domain.PDFImport{}, err
	}
	if _, err = tx.Exec(ctx, `update pdf_lesson_sources set reviewed_revision=null,reviewed_at=null,reviewed_by=null where import_id=$1`, id); err != nil {
		return domain.PDFImport{}, err
	}
	if _, err = tx.Exec(ctx, `update learning_lessons set draft_revision=draft_revision+1,updated_at=now() where id in(select lesson_id from pdf_lesson_sources where import_id=$1)`, id); err != nil {
		return domain.PDFImport{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.PDFImport{}, err
	}
	return s.GetPDFImport(ctx, actor, id)
}

func (s *Store) CreatePDFLesson(ctx context.Context, actor domain.User, id, courseID string, in domain.LearningLessonInput) (domain.LearningLesson, error) {
	if err := validateLearningLesson(&in); err != nil {
		return domain.LearningLesson{}, err
	}
	var lessonID string
	err := s.catalogWrite(ctx, actor, courseID, func(tx pgx.Tx) error {
		source, err := s.getPDF(ctx, tx, actor, id)
		if err != nil {
			return err
		}
		if source.Status != "needs_review" && source.Status != "ocr_required" {
			return apperr.ErrInvalidState
		}
		lessonID, err = insertLearningLesson(ctx, tx, actor, courseID, in)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `insert into pdf_lesson_sources(lesson_id,import_id) values($1,$2)`, lessonID, id)
		return err
	})
	if err != nil {
		return domain.LearningLesson{}, err
	}
	return s.GetLearningLesson(ctx, actor, lessonID)
}
func (s *Store) PDFLessonSource(ctx context.Context, actor domain.User, lessonID string) (map[string]any, error) {
	if _, err := s.learningCourse(ctx, actor, lessonID); err != nil {
		return nil, err
	}
	var id string
	var revision int64
	var reviewed *int64
	err := s.db.QueryRow(ctx, `select p.import_id::text,l.draft_revision,p.reviewed_revision from pdf_lesson_sources p join learning_lessons l on l.id=p.lesson_id where l.id=$1`, lessonID).Scan(&id, &revision, &reviewed)
	if err != nil {
		return nil, catalogError(err)
	}
	source, err := s.GetPDFImport(ctx, actor, id)
	return map[string]any{"source": source, "revision": revision, "reviewed": reviewed != nil && *reviewed == revision}, err
}
func (s *Store) ApprovePDFLesson(ctx context.Context, actor domain.User, lessonID string, revision int64, confirmed bool) error {
	if !confirmed {
		return apperr.ErrBadRequest
	}
	courseID, err := s.learningCourse(ctx, actor, lessonID)
	if err != nil {
		return err
	}
	return s.catalogWrite(ctx, actor, courseID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `update pdf_lesson_sources p set reviewed_revision=l.draft_revision,reviewed_at=now(),reviewed_by=$2 from learning_lessons l where l.id=$1 and p.lesson_id=l.id and l.draft_revision=$3
  and exists(select 1 from pdf_pages where import_id=p.import_id) and not exists(select 1 from pdf_pages where import_id=p.import_id and reviewed_at is null)
  and exists(select 1 from exercises e join lesson_versions v on v.id=e.version_id where v.lesson_id=l.id and v.number=0)`, lessonID, actor.ID, revision)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return apperr.ErrInvalidState
		}
		return nil
	})
}

func (s *Store) validatePDFMedia(ctx context.Context, tx pgx.Tx, actor domain.User, media []domain.ExerciseMedia) error {
	if len(media) > 4 {
		return apperr.ErrBadRequest
	}
	seen := map[string]bool{}
	for _, m := range media {
		if !platform.ValidUUID(m.AssetID) || strings.TrimSpace(m.Alt) == "" || len(m.Alt) > 1000 || strings.ContainsRune(m.Alt, 0) || seen[m.AssetID] {
			return apperr.ErrBadRequest
		}
		seen[m.AssetID] = true
		var allowed bool
		err := tx.QueryRow(ctx, `select true from pdf_assets a join pdf_imports p on p.id=a.import_id where a.id=$1 and p.organization_id=$2 and p.author_id=$3 and a.kind='figure'`, m.AssetID, actor.OrganizationID, actor.ID).Scan(&allowed)
		if err != nil {
			return catalogError(err)
		}
	}
	return nil
}
func (s *Store) PDFAsset(ctx context.Context, actor domain.User, id string) (io.ReadCloser, error) {
	if !platform.ValidUUID(id) {
		return nil, apperr.ErrBadRequest
	}
	if s.objects == nil {
		return nil, apperr.ErrInvalidState
	}
	// Never expose source pages to students. A figure must appear in an available version or selected assignment exercise.
	var key string
	err := s.db.QueryRow(ctx, `select a.object_key from pdf_assets a join pdf_imports p on p.id=a.import_id where a.id=$4 and p.organization_id=$1 and (
 ($3='tutor' and p.author_id=$2) or (a.kind='figure' and exists(select 1 `+catalogLessonFrom+` join exercises e on e.version_id=v.id where `+catalogLessonAccess+` and v.sealed and e.media @> jsonb_build_array(jsonb_build_object('asset_id',$4::text))))
 or (a.kind='figure' and $3='student' and exists(select 1 `+catalogLessonFrom+` join learning_attempts la on la.lesson_id=l.id and la.student_id=$2 and la.assignment_id is null join exercises e on e.version_id=la.version_id where `+catalogLessonAccess+` and e.media @> jsonb_build_array(jsonb_build_object('asset_id',$4::text))))
 or (a.kind='figure' and exists(select 1 from assignments ass join material_assignments ma on ma.assignment_id=ass.id join exercises e on e.version_id=ma.version_id and e.exercise_key=any(ma.exercise_keys) join relations r on r.tutor_id=ass.tutor_id and r.student_id=ass.student_id and r.organization_id=ass.organization_id and r.status='active'
 where ass.organization_id=$1 and ass.status<>'archived' and (($3='student' and ass.student_id=$2) or ($3='tutor' and ass.tutor_id=$2)) and e.media @> jsonb_build_array(jsonb_build_object('asset_id',$4::text))))
 or (a.kind='figure' and exists(select 1 `+studySessionFrom+`join exercises e on e.version_id=la.version_id and e.exercise_key=any(ss.exercise_keys) where `+studySessionAccess+` and e.media @> jsonb_build_array(jsonb_build_object('asset_id',$4::text)))))`, append(catalogArgs(actor), id)...).Scan(&key)
	if err != nil {
		return nil, catalogError(err)
	}
	return s.objects.Open(ctx, key)
}

// Durable leases allow a restart to retry extraction without duplicating derived assets.
func (s *Store) ProcessPDFImport(ctx context.Context) (bool, error) {
	if s.pdf == nil || s.objects == nil {
		return false, apperr.ErrInvalidState
	}
	token, err := platform.NewUUID()
	if err != nil {
		return false, err
	}
	var id, key string
	err = s.db.QueryRow(ctx, `with job as(select id from pdf_imports where (status='queued' and (lease_until is null or lease_until<now()) or status='extracting' and lease_until<now()) and extraction_attempts<3 order by created_at,id for update skip locked limit 1)
 update pdf_imports p set status='extracting',lease_until=now()+interval '2 minutes',lease_token=$1,extraction_attempts=extraction_attempts+1 from job where p.id=job.id returning p.id::text,p.source_key`, token).Scan(&id, &key)
	if errors.Is(err, pgx.ErrNoRows) {
		_, err = s.db.Exec(ctx, `update pdf_imports set status='failed',failure='Обработка прервана. Загрузите документ повторно.',lease_token=null,lease_until=null where status='extracting' and lease_until<now() and extraction_attempts>=3`)
		return false, err
	}
	if err != nil {
		return false, err
	}
	jobCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	body, err := s.objects.Open(jobCtx, key)
	var source []byte
	if err == nil {
		source, err = io.ReadAll(io.LimitReader(body, pdfextract.MaxPDFBytes+1))
		body.Close()
	}
	var result pdfextract.Result
	if err == nil {
		result, err = s.pdf.Extract(jobCtx, source)
	}
	if err != nil {
		if ctx.Err() != nil {
			return true, ctx.Err()
		}
		_, writeErr := s.db.Exec(ctx, `update pdf_imports set status=case when $3 or extraction_attempts>=3 then 'failed' else 'queued' end,
 failure=case when $3 or extraction_attempts>=3 then 'Не удалось извлечь PDF. Проверьте формат, отсутствие пароля и лимит 10 страниц.' else '' end,
 lease_token=null,lease_until=now()+interval '15 seconds' where id=$1 and lease_token=$2`, id, token, errors.Is(err, pdfextract.ErrInvalidPDF))
		return true, writeErr
	}
	tx, err := s.db.Begin(jobCtx)
	if err != nil {
		return true, err
	}
	defer tx.Rollback(jobCtx)
	var locked bool
	if err = tx.QueryRow(jobCtx, `select true from pdf_imports where id=$1 and lease_token=$2 and status='extracting' for update`, id, token).Scan(&locked); err != nil {
		return true, catalogError(err)
	}
	status := "needs_review"
	var author string
	if err = tx.QueryRow(jobCtx, `select author_id::text from pdf_imports where id=$1`, id).Scan(&author); err != nil {
		return true, err
	}
	committed := false
	staged := []string{}
	defer func() {
		if !committed {
			s.cleanupPDFObjects(tx, author, staged)
		}
	}()
	putImage := func(data []byte, kind string, page, ordinal int) (string, error) {
		normalized, e := pdfextract.NormalizePNG(data)
		if e != nil {
			return "", e
		}
		cfg, e := png.DecodeConfig(bytes.NewReader(normalized))
		if e != nil {
			return "", e
		}
		if e = s.pdfStorageQuota(jobCtx, tx, author, int64(len(normalized))); e != nil {
			return "", e
		}
		obj, e := s.objects.Put(jobCtx, bytes.NewReader(normalized), 2<<20)
		if e != nil {
			return "", e
		}
		staged = append(staged, obj.Key)
		var assetID string
		e = tx.QueryRow(jobCtx, `insert into pdf_assets(import_id,kind,page_number,ordinal,object_key,sha256,bytes,width,height) values($1,$2,$3,$4,$5,$6,$7,$8,$9) returning id::text`, id, kind, page, ordinal, obj.Key, obj.SHA256, obj.Size, cfg.Width, cfg.Height).Scan(&assetID)
		return assetID, e
	}
	if len(result.Pages) == 0 || len(result.Pages) > 10 {
		return true, pdfextract.ErrInvalidPDF
	}
	for _, page := range result.Pages {
		assetID, e := putImage(page.PNG, "page", page.Number, 0)
		if e != nil {
			return true, e
		}
		ocr := strings.TrimSpace(page.Text) == ""
		if ocr {
			status = "ocr_required"
		}
		if _, err = tx.Exec(jobCtx, `insert into pdf_pages(import_id,number,original_text,image_id,ocr_required) values($1,$2,$3,$4,$5)`, id, page.Number, page.Text, assetID, ocr); err != nil {
			return true, err
		}
	}
	for _, f := range result.Figures {
		if _, err = putImage(f.PNG, "figure", f.Page, f.Number); err != nil {
			return true, err
		}
	}
	warnings, _ := json.Marshal(result.Warnings)
	if result.Warnings == nil {
		warnings = []byte(`[]`)
	}
	if _, err = tx.Exec(jobCtx, `update pdf_imports set status=$3,warnings=$4,lease_token=null,lease_until=null where id=$1 and lease_token=$2`, id, token, status, warnings); err != nil {
		return true, err
	}
	err = tx.Commit(jobCtx)
	committed = err == nil
	return true, err
}
func (s *Store) RunPDFQueue(ctx context.Context, log *slog.Logger) {
	for ctx.Err() == nil {
		worked, err := s.ProcessPDFImport(ctx)
		if err != nil && ctx.Err() == nil {
			log.Error("PDF queue processing failed", "error", err)
		}
		if !worked || err != nil {
			select {
			case <-ctx.Done():
				return
			case <-time.After(2 * time.Second):
			}
		}
	}
}

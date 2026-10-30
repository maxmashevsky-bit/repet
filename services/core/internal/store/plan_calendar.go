package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"tutor-platform/services/core/internal/apperr"
	"tutor-platform/services/core/internal/domain"
	"tutor-platform/services/core/internal/platform"
)

func (s *Store) PublishPlanCalendar(ctx context.Context, actor domain.User, in domain.PlanCalendarInput) (domain.PlanCalendarResult, error) {
	var out domain.PlanCalendarResult
	if actor.Role != domain.RoleTutor {
		return out, apperr.ErrForbidden
	}
	if !platform.ValidUUID(in.RequestID) || in.Revision < 1 {
		return out, apperr.ErrBadRequest
	}
	loc, err := time.LoadLocation(actor.Timezone)
	if err != nil {
		return out, apperr.ErrBadRequest
	}
	from, err := time.ParseInLocation("2006-01-02", in.From, loc)
	if err != nil {
		return out, apperr.ErrBadRequest
	}
	to, err := time.ParseInLocation("2006-01-02", in.To, loc)
	if err != nil || to.Before(from) || to.After(from.AddDate(0, 0, 83)) {
		return out, apperr.ErrBadRequest
	}
	payload, _ := json.Marshal(in)
	hash := sha256.Sum256(payload)
	fingerprint := hex.EncodeToString(hash[:])
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `select pg_advisory_xact_lock(hashtextextended($1,1523))`, actor.ID); err != nil {
		return out, err
	}
	var oldHash string
	var result []byte
	err = tx.QueryRow(ctx, `select payload_hash,result from plan_calendar_exports where tutor_id=$1 and request_id=$2 and organization_id=$3`, actor.ID, in.RequestID, actor.OrganizationID).Scan(&oldHash, &result)
	if err == nil {
		if oldHash != fingerprint {
			return out, apperr.ErrConflict
		}
		if err = json.Unmarshal(result, &out); err != nil {
			return out, err
		}
		return out, tx.Commit(ctx)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return out, err
	}
	now := time.Now().In(loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	if from.Before(today) || to.After(today.AddDate(1, 0, 0)) {
		return out, apperr.ErrBadRequest
	}
	var raw []byte
	var revision int64
	if err = tx.QueryRow(ctx, `select document,revision from tutor_plans where tutor_id=$1 and organization_id=$2 for update`, actor.ID, actor.OrganizationID).Scan(&raw, &revision); err != nil {
		return out, catalogError(err)
	}
	if revision != in.Revision {
		return out, apperr.ErrConflict
	}
	var plan domain.TutorPlan
	if err = json.Unmarshal(raw, &plan); err != nil {
		return out, err
	}
	students := map[string]domain.PlanStudent{}
	for _, student := range plan.Students {
		students[student.ID] = student
	}
	out.From, out.To = in.From, in.To
	for day := from; !day.After(to); day = day.AddDate(0, 0, 1) {
		weekday := int(day.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		for _, slot := range plan.Lessons {
			if slot.Weekday != weekday {
				continue
			}
			if out.Created+out.Existing+out.Unlinked+out.Paused >= 2000 {
				return domain.PlanCalendarResult{}, apperr.ErrBadRequest
			}
			if slot.Paused {
				out.Paused++
				continue
			}
			student := students[slot.StudentID]
			if student.RelationID == "" {
				out.Unlinked++
				continue
			}
			var exists bool
			if err = tx.QueryRow(ctx, `select exists(select 1 from plan_calendar_occurrences where tutor_id=$1 and plan_lesson_key=$2 and local_date=$3::date)`, actor.ID, slot.ID, day.Format("2006-01-02")).Scan(&exists); err != nil {
				return out, err
			}
			if exists {
				out.Existing++
				continue
			}
			var studentID string
			if err = tx.QueryRow(ctx, `select student_id::text from relations where id=$1 and tutor_id=$2 and organization_id=$3 and status='active' for share`, student.RelationID, actor.ID, actor.OrganizationID).Scan(&studentID); err != nil {
				return domain.PlanCalendarResult{}, catalogError(err)
			}
			start := time.Date(day.Year(), day.Month(), day.Day(), slot.StartMinute/60, slot.StartMinute%60, 0, 0, loc)
			end := start.Add(time.Duration(slot.Duration) * time.Minute)
			if start.Hour() != slot.StartMinute/60 || start.Minute() != slot.StartMinute%60 {
				return domain.PlanCalendarResult{}, apperr.ErrBadRequest
			}
			if err = tx.QueryRow(ctx, `select exists(select 1 from lessons where organization_id=$1 and (tutor_id=$2 or student_id=$3) and status='scheduled' and starts_at<$5 and ends_at>$4)`, actor.OrganizationID, actor.ID, studentID, start, end).Scan(&exists); err != nil {
				return out, err
			}
			if exists {
				return domain.PlanCalendarResult{}, apperr.ErrConflict
			}
			id, e := platform.NewUUID()
			if e != nil {
				return out, e
			}
			title := student.Subject
			if title == "" {
				title = "Занятие"
			}
			if _, err = tx.Exec(ctx, `insert into lessons(id,organization_id,relation_id,tutor_id,student_id,title,starts_at,ends_at,status,created_by)values($1,$2,$3,$4,$5,$6,$7,$8,'scheduled',$4)`, id, actor.OrganizationID, student.RelationID, actor.ID, studentID, title, start, end); err != nil {
				return out, catalogError(err)
			}
			if _, err = tx.Exec(ctx, `insert into plan_calendar_occurrences(tutor_id,plan_lesson_key,local_date,lesson_id)values($1,$2,$3::date,$4)`, actor.ID, slot.ID, day.Format("2006-01-02"), id); err != nil {
				return out, err
			}
			if err = insertNotification(ctx, tx, actor.OrganizationID, studentID, "Новое занятие", title, "/calendar?date="+start.In(loc).Format("2006-01-02"), "lesson:"+id+":created"); err != nil {
				return out, err
			}
			out.Created++
		}
	}
	result, err = json.Marshal(out)
	if err != nil {
		return out, err
	}
	if _, err = tx.Exec(ctx, `insert into plan_calendar_exports(tutor_id,request_id,organization_id,payload_hash,result)values($1,$2,$3,$4,$5)`, actor.ID, in.RequestID, actor.OrganizationID, fingerprint, result); err != nil {
		return out, err
	}
	if err = tx.Commit(ctx); err != nil {
		return out, err
	}
	s.recordAudit(ctx, actor, "planner.calendar_published", "tutor_plan", actor.ID, map[string]any{"created": out.Created, "from": out.From, "to": out.To})
	return out, nil
}

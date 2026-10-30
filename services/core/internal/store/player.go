package store

import (
	"context"
	"encoding/json"
	"strings"
	"tutor-platform/services/core/internal/apperr"
	"tutor-platform/services/core/internal/domain"
	"tutor-platform/services/core/internal/platform"
)

func playerCanVisit(state domain.LearningState, step string) bool {
	if step == "theory" {
		return true
	}
	if step == "results" {
		return state.Attempt.CompletedAt != nil
	}
	if step == "example" {
		return state.Version.Example != nil && (state.Attempt.TheorySeen || state.Attempt.PlayerStep == "theory")
	}
	ready := state.Attempt.TheorySeen || state.Attempt.PlayerStep == "theory"
	if state.Version.Example != nil {
		ready = state.Attempt.TheorySeen && (state.Attempt.ExampleSeen || state.Attempt.PlayerStep == "example")
	}
	if !ready {
		return false
	}
	for _, e := range state.Version.Exercises {
		if step == "exercise:"+e.ID {
			return true
		}
		answered := false
		for _, a := range state.Answers {
			if a.ExerciseID == e.ID {
				answered = true
				break
			}
		}
		if !answered {
			return false
		}
	}
	return false
}
func (s *Store) NavigateLearning(ctx context.Context, actor domain.User, lessonID, id string, in domain.LearningNavigateInput) (domain.LearningState, error) {
	var out domain.LearningState
	if !platform.ValidUUID(id) || !platform.ValidUUID(in.RequestID) || len(in.Step) > 60 {
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
	state, err := readLearningState(ctx, tx, actor, lessonID, id)
	if err != nil {
		return out, err
	}
	payload, _ := json.Marshal(map[string]string{"lesson_id": lessonID, "id": id, "step": in.Step})
	previous, err := learningReplay(ctx, tx, actor.ID, in.RequestID, "navigate", payload)
	if err != nil {
		return out, err
	}
	if previous == "" {
		var studyExam bool
		if err = tx.QueryRow(ctx, `select exists(select 1 from study_sessions ss where ss.learning_attempt_id=$1 and ss.deadline_at is not null)`, id).Scan(&studyExam); err != nil {
			return out, err
		}
		allowed := playerCanVisit(state, in.Step)
		if studyExam {
			for _, ex := range state.Version.Exercises {
				if in.Step == "exercise:"+ex.ID {
					allowed = true
				}
			}
		}
		if !allowed {
			return out, apperr.ErrInvalidState
		}
		theory := state.Attempt.TheorySeen || in.Step != "theory"
		example := state.Attempt.ExampleSeen || strings.HasPrefix(in.Step, "exercise:") || in.Step == "results"
		if _, err = tx.Exec(ctx, `update learning_attempts set player_step=$2,theory_seen=$3,example_seen=$4 where id=$1`, id, in.Step, theory, example); err != nil {
			return out, err
		}
		if err = learningEvent(ctx, tx, actor.ID, in.RequestID, id, "navigate", payload); err != nil {
			return out, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return out, err
	}
	return s.GetLearningState(ctx, actor, lessonID, id)
}
func (s *Store) LearningSolution(ctx context.Context, actor domain.User, lessonID, id, exerciseID string) (domain.LearningSolution, error) {
	out := domain.LearningSolution{ExerciseID: exerciseID}
	if !platform.ValidUUID(id) || !platform.ValidUUID(exerciseID) {
		return out, apperr.ErrBadRequest
	}
	state, err := s.GetLearningState(ctx, actor, lessonID, id)
	if err != nil {
		return out, err
	}
	var activeExam bool
	if err = s.db.QueryRow(ctx, `select exists(select 1 from learning_attempts where student_id=$1 and lesson_id=$2 and mode='exam' and completed_at is null)`, actor.ID, lessonID).Scan(&activeExam); err != nil {
		return out, err
	}
	if activeExam {
		return out, apperr.ErrInvalidState
	}
	if state.Attempt.Mode == "exam" && state.Attempt.CompletedAt == nil {
		return out, apperr.ErrInvalidState
	}
	var latest *domain.ExerciseAttempt
	for i := range state.Answers {
		if state.Answers[i].ExerciseID == exerciseID {
			latest = &state.Answers[i]
			break
		}
	}
	if latest == nil {
		return out, apperr.ErrInvalidState
	}
	if latest.Kind == "long_text" && latest.ReviewedAt == nil {
		return out, apperr.ErrInvalidState
	}
	var key, text string
	err = s.db.QueryRow(ctx, `select k.answer,k.explanation from exercises e join exercise_answer_keys k on k.exercise_id=e.id where e.version_id=$1 and e.exercise_key=$2`, state.Version.ID, exerciseID).Scan(&key, &text)
	if err != nil {
		return out, catalogError(err)
	}
	if strings.TrimSpace(text) == "" {
		if latest.Kind == "long_text" {
			text = latest.Feedback
		} else {
			var options []string
			if json.Unmarshal([]byte(key), &options) == nil {
				key = strings.Join(options, ", ")
			}
			text = "Ответ: " + key
		}
	}
	out.Text = text
	out.Available = strings.TrimSpace(text) != ""
	return out, nil
}

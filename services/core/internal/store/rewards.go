package store

import (
	"context"
	"tutor-platform/services/core/internal/apperr"
	"tutor-platform/services/core/internal/domain"
)

func (s *Store) GetRewardPreference(ctx context.Context, actor domain.User, studentID string) (domain.RewardPreference, error) {
	if actor.Role != domain.RoleTutor {
		return domain.RewardPreference{}, apperr.ErrForbidden
	}
	if _, err := s.analyticsAccess(ctx, actor, studentID); err != nil {
		return domain.RewardPreference{}, err
	}
	return catalogJSON[domain.RewardPreference](ctx, s.db, `select jsonb_build_object('enabled',coalesce(p.enabled,true),'enabled_from',p.enabled_from,'updated_at',p.updated_at) from(select 1)seed left join learning_reward_preferences p on p.tutor_id=$1 and p.student_id=$2 and p.organization_id=$3`, actor.ID, studentID, actor.OrganizationID)
}
func (s *Store) SetRewardPreference(ctx context.Context, actor domain.User, studentID string, in domain.RewardPreferenceInput) (domain.RewardPreference, error) {
	if actor.Role != domain.RoleTutor {
		return domain.RewardPreference{}, apperr.ErrForbidden
	}
	if in.Enabled == nil {
		return domain.RewardPreference{}, apperr.ErrBadRequest
	}
	if _, err := s.analyticsAccess(ctx, actor, studentID); err != nil {
		return domain.RewardPreference{}, err
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return domain.RewardPreference{}, err
	}
	defer tx.Rollback(ctx)
	if err = lockLearning(ctx, tx, studentID); err != nil {
		return domain.RewardPreference{}, err
	}
	var relation string
	if err = tx.QueryRow(ctx, `select id::text from relations where tutor_id=$1 and student_id=$2 and organization_id=$3 and status='active' for share`, actor.ID, studentID, actor.OrganizationID).Scan(&relation); err != nil {
		return domain.RewardPreference{}, catalogError(err)
	}
	_, err = tx.Exec(ctx, `insert into learning_reward_preferences(tutor_id,student_id,organization_id,enabled,enabled_from)values($1,$2,$3,$4,case when $4 then null else clock_timestamp() end)
 on conflict(tutor_id,student_id) do update set enabled=excluded.enabled,
 enabled_from=case when excluded.enabled and not learning_reward_preferences.enabled then clock_timestamp() else learning_reward_preferences.enabled_from end,
 updated_at=case when learning_reward_preferences.enabled<>excluded.enabled then clock_timestamp() else learning_reward_preferences.updated_at end`, actor.ID, studentID, actor.OrganizationID, *in.Enabled)
	if err != nil {
		return domain.RewardPreference{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.RewardPreference{}, err
	}
	return s.GetRewardPreference(ctx, actor, studentID)
}

// A click or one correct guess never earns XP: require clean recall on two UTC days.
const rewardEvidence = `with evidence as (
 select ea.id,ea.created_at,ea.reviewed_at,ea.status,e.kind,la.completed_at,
 coalesce(a.tutor_id,es.author_id,c.author_id) tutor_id,
 coalesce((provenance.item->>'lesson_id')::uuid,la.lesson_id) source_lesson_id,
 coalesce((provenance.item->'exercise'->>'id')::uuid,e.exercise_key) source_exercise_key,
 row_number()over(partition by la.id,e.id order by ea.attempt_number) first_rank,
 exists(select 1 from learning_hints h where h.learning_attempt_id=la.id and h.exercise_id=e.id) hinted
 ` + analyticsRunFrom + analyticsTopicFrom + ` join exercise_attempts ea on ea.learning_attempt_id=la.id and ea.exercise_id=e.id
 where ` + analyticsRunScope + ` and la.completed_at is not null
), clean as(select e.* from evidence e left join learning_reward_preferences p on p.tutor_id=e.tutor_id and p.student_id=$1 and p.organization_id=$2
 where e.first_rank=1 and e.status='correct' and e.kind in ('numeric','text','long_text') and not e.hinted
 and coalesce(p.enabled,true) and (p.enabled_from is null or e.created_at>=p.enabled_from)),
 qualified as(select tutor_id,source_lesson_id,source_exercise_key,max(greatest(created_at,coalesce(reviewed_at,created_at),completed_at)) earned_at,
 (array_agg(id order by created_at desc,id))[1] evidence_attempt_id
 from clean group by tutor_id,source_lesson_id,source_exercise_key having count(distinct (created_at at time zone 'UTC')::date)>=2 and max(created_at)-min(created_at)>=interval '24 hours') `

func rewardLevel(xp int) (int, int) { return 1 + xp/100, xp % 100 }
func (s *Store) LearningRewards(ctx context.Context, actor domain.User, studentID string, f domain.CatalogFilter) (domain.RewardSummary, error) {
	var out domain.RewardSummary
	if f.Limit < 1 || f.Limit > 100 || f.Offset < 0 || f.Offset > 100000 {
		return out, apperr.ErrBadRequest
	}
	if _, err := s.analyticsAccess(ctx, actor, studentID); err != nil {
		return out, err
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx)
	if err = lockLearning(ctx, tx, studentID); err != nil {
		return out, err
	}
	if _, err = analyticsAccess(ctx, tx, actor, studentID); err != nil {
		return out, err
	}
	_, err = tx.Exec(ctx, rewardEvidence+`insert into learning_reward_awards(organization_id,student_id,tutor_id,source_lesson_id,source_exercise_key,evidence_attempt_id,earned_at)
 select distinct on(source_lesson_id,source_exercise_key) $2,$1,tutor_id,source_lesson_id,source_exercise_key,evidence_attempt_id,earned_at from qualified order by source_lesson_id,source_exercise_key,earned_at,tutor_id
 on conflict(student_id,source_lesson_id,source_exercise_key) do nothing`, studentID, actor.OrganizationID, string(actor.Role), actor.ID)
	if err != nil {
		return out, err
	}
	const available = ` from learning_reward_awards a left join learning_reward_preferences p on p.tutor_id=a.tutor_id and p.student_id=a.student_id and p.organization_id=a.organization_id where a.student_id=$1 and a.organization_id=$2 and ($3='student' or a.tutor_id=$4) and coalesce(p.enabled,true)`
	out, err = catalogJSON[domain.RewardSummary](ctx, tx, `select jsonb_build_object('enabled',case when $3='tutor' then coalesce((select enabled from learning_reward_preferences where tutor_id=$4 and student_id=$1 and organization_id=$2),true) else not exists(select 1 from relations where student_id=$1 and organization_id=$2 and status='active') or exists(select 1 from relations r left join learning_reward_preferences p on p.tutor_id=r.tutor_id and p.student_id=r.student_id and p.organization_id=r.organization_id where r.student_id=$1 and r.organization_id=$2 and r.status='active' and coalesce(p.enabled,true)) end,
 'xp',(select coalesce(sum(a.points),0)`+available+`),'weekly_xp',(select coalesce(sum(a.points),0)`+available+` and a.earned_at>=date_trunc('week',clock_timestamp() at time zone 'UTC') at time zone 'UTC'),
 'weekly_achievements',(select count(*)`+available+` and a.earned_at>=date_trunc('week',clock_timestamp() at time zone 'UTC') at time zone 'UTC'),
 'awards',jsonb_build_object('items',coalesce((select jsonb_agg(item order by earned_at desc,id) from(select a.id,a.earned_at,jsonb_build_object('id',a.id,'title','Уверенное повторение','points',a.points,'earned_at',a.earned_at) item`+available+` order by a.earned_at desc,a.id limit $5 offset $6)p),'[]'::jsonb),'total',(select count(*)`+available+`),'limit',$5::int,'offset',$6::int))`, studentID, actor.OrganizationID, string(actor.Role), actor.ID, f.Limit, f.Offset)
	if err != nil {
		return out, err
	}
	out.LevelTarget = 100
	out.Level, out.LevelProgress = rewardLevel(out.XP)
	if !out.Enabled {
		out.XP = 0
		out.Level = 0
		out.LevelProgress = 0
		out.WeeklyXP = 0
		out.WeeklyAchievements = 0
		out.Awards.Items = []domain.RewardAward{}
		out.Awards.Total = 0
	}
	return out, tx.Commit(ctx)
}

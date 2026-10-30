alter table lesson_versions add column example jsonb check(example is null or jsonb_typeof(example)='object');
alter table learning_attempts add column mode text not null default 'practice' check(mode in ('practice','exam'));
alter table learning_attempts add column player_step text not null default 'theory';
alter table learning_attempts add column theory_seen boolean not null default false;
alter table learning_attempts add column example_seen boolean not null default false;
update learning_attempts la set theory_seen=true,example_seen=true,player_step=case when completed_at is not null then 'results' else coalesce((
 select 'exercise:'||e.exercise_key::text from exercises e where e.version_id=la.version_id
 and not exists(select 1 from exercise_attempts a where a.learning_attempt_id=la.id and a.exercise_id=e.id)
 order by e.position,e.exercise_key limit 1),'theory') end;
alter table learning_events drop constraint learning_events_kind_check;
alter table learning_events add constraint learning_events_kind_check check(kind in ('start','complete','hint','navigate'));

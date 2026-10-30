do $$ begin
 if exists(select 1 from learning_reward_awards) or exists(select 1 from learning_reward_preferences) then
  raise exception 'learning rewards data exists; export before downgrade';
 end if;
end $$;
drop table learning_reward_awards;
drop table learning_reward_preferences;

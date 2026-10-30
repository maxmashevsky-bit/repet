drop table if exists exercise_answer_keys;
drop table if exists exercises;
alter table learning_lessons drop constraint if exists learning_lessons_id_published_version_id_fkey;
drop table if exists lesson_versions;
drop table if exists learning_lessons;
drop table if exists topics;
drop table if exists courses;
drop function if exists protect_learning_snapshot();

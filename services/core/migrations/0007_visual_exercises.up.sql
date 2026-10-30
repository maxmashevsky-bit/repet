alter table exercises add column config jsonb not null default '{}'::jsonb check (jsonb_typeof(config)='object');
alter table exercises drop constraint exercises_kind_check;
alter table exercises add constraint exercises_kind_check check (kind in ('text','numeric','single_choice','multiple_choice','long_text','number_line','grid_select','matching','ordering','geometry'));

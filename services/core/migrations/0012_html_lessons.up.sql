create table html_lessons (
 id uuid primary key default gen_random_uuid(),
 organization_id uuid not null references organizations(id),
 author_id uuid not null references users(id),
 course_id uuid not null references courses(id),
 status text not null default 'draft' check(status in ('draft','published','archived')),
 current_version_id uuid,
 published_version_id uuid,
 created_at timestamptz not null default now()
);
create table html_lesson_versions (
 id uuid primary key default gen_random_uuid(),
 lesson_id uuid not null references html_lessons(id),
 topic_id uuid not null references topics(id),
 number integer not null check(number>0),
 title text not null check(length(title) between 1 and 180),
 summary text not null,
 track text not null check(track in ('school','olympiad','oge','ege')),
 grade_min integer not null check(grade_min between 1 and 11),
 grade_max integer not null check(grade_max between grade_min and 11),
 difficulty integer not null check(difficulty between 1 and 5),
 filename text not null,
 source text not null check(octet_length(source) between 1 and 524288),
 sha256 text not null,
 metadata jsonb not null,
 rights_status text not null check(rights_status in ('own','permission','open_license')),
 license text not null,
 source_label text not null,
 request_id uuid not null,
 request_hash text not null,
 created_at timestamptz not null default now(),
 published_at timestamptz,
 unique(lesson_id,number), unique(lesson_id,request_id), unique(lesson_id,id)
);
alter table html_lessons add foreign key(id,current_version_id) references html_lesson_versions(lesson_id,id);
alter table html_lessons add foreign key(id,published_version_id) references html_lesson_versions(lesson_id,id);
create index html_lessons_author_idx on html_lessons(organization_id,author_id,created_at desc,id);
create index html_lessons_course_idx on html_lessons(course_id,status);
create index html_versions_filters_idx on html_lesson_versions(track,grade_min,grade_max,difficulty);
create table html_assignment_links (
 assignment_id uuid primary key references assignments(id),
 lesson_id uuid not null,
 version_id uuid not null,
 request_id uuid not null,
 payload jsonb not null,
 foreign key(lesson_id,version_id) references html_lesson_versions(lesson_id,id)
);
create table html_preview_grants (
 token_hash text primary key,
 user_id uuid not null references users(id),
 lesson_id uuid not null,
 version_id uuid not null,
 assignment_id uuid references html_assignment_links(assignment_id),
 expires_at timestamptz not null,
 foreign key(lesson_id,version_id) references html_lesson_versions(lesson_id,id)
);
create index html_preview_expiry_idx on html_preview_grants(expires_at);
create table html_conversions (
 learning_lesson_id uuid primary key references learning_lessons(id),
 html_version_id uuid not null references html_lesson_versions(id)
);
create function protect_html_version() returns trigger language plpgsql as $$ begin
 if tg_op='DELETE' or (to_jsonb(new)-'published_at') is distinct from (to_jsonb(old)-'published_at')
 or old.published_at is not null or new.published_at is null then
 raise exception 'HTML source versions are immutable' using errcode='23514';
 end if;
 return new;
end $$;
create trigger html_version_immutable before update or delete on html_lesson_versions for each row execute function protect_html_version();

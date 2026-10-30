create table teaching_preferences (
 tutor_id uuid primary key references users(id),
 organization_id uuid not null references organizations(id),
 revision bigint not null check(revision>0),
 subjects text[] not null check(cardinality(subjects) between 1 and 10),
 grades integer[] not null check(cardinality(grades) between 1 and 11 and grades<@array[1,2,3,4,5,6,7,8,9,10,11]),
 default_duration_minutes integer not null check(default_duration_minutes between 1 and 240),
 updated_at timestamptz not null default clock_timestamp()
);

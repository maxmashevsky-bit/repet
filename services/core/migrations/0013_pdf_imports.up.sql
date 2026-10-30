create table pdf_imports (
 id uuid primary key default gen_random_uuid(),
 organization_id uuid not null references organizations(id),
 author_id uuid not null references users(id),
 filename text not null,
 source_key text not null unique,
 sha256 text not null,
 source_bytes bigint not null check(source_bytes between 1 and 5242880),
 rights_status text not null check(rights_status in ('own','permission','open_license')),
 license text not null,
 source_label text not null,
 request_id uuid not null,
 request_hash text not null,
 status text not null check(status in ('queued','extracting','needs_review','ocr_required','failed')),
 lease_until timestamptz,
 lease_token uuid,
 extraction_attempts integer not null default 0 check(extraction_attempts between 0 and 3),
 warnings jsonb not null default '[]'::jsonb,
 failure text not null default '',
 created_at timestamptz not null default now(),
 unique(author_id,request_id)
);
create index pdf_imports_owner_idx on pdf_imports(organization_id,author_id,created_at desc,id);
create index pdf_imports_queue_idx on pdf_imports(status,lease_until,created_at);
create table pdf_assets (
 id uuid primary key default gen_random_uuid(),
 import_id uuid not null references pdf_imports(id),
 kind text not null check(kind in ('page','figure')),
 page_number integer not null check(page_number between 1 and 10),
 ordinal integer not null check(ordinal>=0),
 object_key text not null unique,
 sha256 text not null,
 bytes bigint not null check(bytes between 1 and 2097152),
 width integer not null check(width between 1 and 3000),
 height integer not null check(height between 1 and 3000),
 unique(import_id,kind,page_number,ordinal),
 unique(import_id,id)
);
create table pdf_pages (
 import_id uuid not null references pdf_imports(id),
 number integer not null check(number between 1 and 10),
 original_text text not null,
 image_id uuid not null,
 ocr_required boolean not null,
 corrected_text text,
 reviewed_at timestamptz,
 reviewed_by uuid references users(id),
 primary key(import_id,number),
 foreign key(import_id,image_id) references pdf_assets(import_id,id)
);
create table pdf_lesson_sources (
 lesson_id uuid primary key references learning_lessons(id),
 import_id uuid not null references pdf_imports(id),
 reviewed_revision bigint,
 reviewed_at timestamptz,
 reviewed_by uuid references users(id)
);
create function protect_pdf_source() returns trigger language plpgsql as $$ begin
 if (new.organization_id,new.author_id,new.filename,new.source_key,new.sha256,new.source_bytes,new.rights_status,new.license,new.source_label,new.request_id,new.request_hash)
 is distinct from (old.organization_id,old.author_id,old.filename,old.source_key,old.sha256,old.source_bytes,old.rights_status,old.license,old.source_label,old.request_id,old.request_hash)
 then raise exception 'PDF originals and source attribution are immutable' using errcode='23514'; end if;
 return new;
end $$;
create trigger pdf_source_immutable before update on pdf_imports for each row execute function protect_pdf_source();
alter table exercises add column media jsonb not null default '[]'::jsonb check(jsonb_typeof(media)='array' and jsonb_array_length(media)<=4);
create function protect_pdf_page() returns trigger language plpgsql as $$ begin
 if (new.import_id,new.number,new.original_text,new.image_id,new.ocr_required) is distinct from
 (old.import_id,old.number,old.original_text,old.image_id,old.ocr_required) then
 raise exception 'PDF extraction originals are immutable' using errcode='23514'; end if;
 return new;
end $$;
create trigger pdf_page_immutable before update on pdf_pages for each row execute function protect_pdf_page();
create function protect_pdf_asset() returns trigger language plpgsql as $$ begin
 raise exception 'PDF assets are immutable' using errcode='23514';
end $$;
create trigger pdf_asset_immutable before update or delete on pdf_assets for each row execute function protect_pdf_asset();

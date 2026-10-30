do $$ begin
 lock table pdf_imports in access exclusive mode;
 if exists(select 1 from pdf_imports) then raise exception 'Export PDF source objects and metadata before rollback'; end if;
 alter table exercises drop column media;
 drop table pdf_lesson_sources,pdf_pages,pdf_assets,pdf_imports;
 drop function protect_pdf_page();
 drop function protect_pdf_asset();
 drop function protect_pdf_source();
end $$;

do $$ begin
    if exists(select 1 from email_verification_tokens) then
        raise exception 'email verification records exist; refusing destructive rollback';
    end if;
end $$;
drop table email_verification_tokens;

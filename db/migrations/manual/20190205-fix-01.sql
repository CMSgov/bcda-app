alter table tokens
add column aco_id uuid
;

alter table tokens
add constraint tokens_aco_id_fkey foreign key (aco_id)
references acos (uuid)
;

update tokens
set aco_id = (select aco_id from users where uuid = user_id)
;
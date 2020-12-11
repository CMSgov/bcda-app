-- Script migrates our cclf_beneficiaries id column from an int to bigint.
-- It does this by copying all of the ids to a temporary column (new_id),
-- then dropping the id columnn, and finally renaming the new_id to id.
ALTER TABLE cclf_beneficiaries ADD COLUMN new_id bigint;
BEGIN; UPDATE cclf_beneficiaries SET new_id = id WHERE id between 480401 and 1480401; COMMIT;
BEGIN; UPDATE cclf_beneficiaries SET new_id = id WHERE id between 1480402 and 2480402; COMMIT;
BEGIN; UPDATE cclf_beneficiaries SET new_id = id WHERE id between 2480403 and 3480403; COMMIT;
BEGIN; UPDATE cclf_beneficiaries SET new_id = id WHERE id between 3480404 and 4480404; COMMIT;
BEGIN; UPDATE cclf_beneficiaries SET new_id = id WHERE id between 4480405 and 5480405; COMMIT;
BEGIN; UPDATE cclf_beneficiaries SET new_id = id WHERE id between 5480406 and 6480406; COMMIT;
BEGIN; UPDATE cclf_beneficiaries SET new_id = id WHERE id between 6480407 and 7480407; COMMIT;
BEGIN; UPDATE cclf_beneficiaries SET new_id = id WHERE id between 7480408 and 8480408; COMMIT;
BEGIN; UPDATE cclf_beneficiaries SET new_id = id WHERE id between 8480409 and 9480409; COMMIT;
BEGIN; UPDATE cclf_beneficiaries SET new_id = id WHERE id between 9480410 and 10480410; COMMIT;
BEGIN; UPDATE cclf_beneficiaries SET new_id = id WHERE id between 10480411 and 11480411; COMMIT;
BEGIN; UPDATE cclf_beneficiaries SET new_id = id WHERE id between 11480412 and 12480412; COMMIT;
BEGIN; UPDATE cclf_beneficiaries SET new_id = id WHERE id between 12480413 and 13480413; COMMIT;
BEGIN; UPDATE cclf_beneficiaries SET new_id = id WHERE id between 13480414 and 14480414; COMMIT;
BEGIN; UPDATE cclf_beneficiaries SET new_id = id WHERE id between 14480415 and 15480415; COMMIT;
CREATE UNIQUE INDEX new_pk_idx on cclf_beneficiaries(new_id);
BEGIN;
ALTER TABLE cclf_beneficiaries drop constraint cclf_beneficiaries_pkey;
ALTER TABLE cclf_beneficiaries alter column new_id set default nextval('cclf_beneficiaries_id_seq'::regclass);
ALTER TABLE cclf_beneficiaries add constraint cclf_beneficiaries_pkey primary key using index new_pk_idx;
ALTER SEQUENCE cclf_beneficiaries_id_seq OWNED BY cclf_beneficiaries.new_id;
ALTER TABLE cclf_beneficiaries drop column id;
ALTER TABLE cclf_beneficiaries rename column new_id to id;
COMMIT;
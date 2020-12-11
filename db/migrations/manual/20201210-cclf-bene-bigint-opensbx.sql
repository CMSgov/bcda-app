-- Script migrates our cclf_beneficiaries id column from an int to bigint.
-- It does this by copying all of the ids to a temporary column (new_id),
-- then dropping the id columnn, and finally renaming the new_id to id.
ALTER TABLE cclf_beneficiaries ADD COLUMN new_id bigint;
BEGIN; UPDATE cclf_beneficiaries SET new_id = id WHERE id between 1050969 and 2050969; COMMIT;
BEGIN; UPDATE cclf_beneficiaries SET new_id = id WHERE id between 2050970 and 3050970; COMMIT;
BEGIN; UPDATE cclf_beneficiaries SET new_id = id WHERE id between 3050971 and 4050971; COMMIT;
BEGIN; UPDATE cclf_beneficiaries SET new_id = id WHERE id between 4050972 and 5050972; COMMIT;
BEGIN; UPDATE cclf_beneficiaries SET new_id = id WHERE id between 5050973 and 6050973; COMMIT;
BEGIN; UPDATE cclf_beneficiaries SET new_id = id WHERE id between 6050974 and 7050974; COMMIT;
BEGIN; UPDATE cclf_beneficiaries SET new_id = id WHERE id between 7050975 and 8050975; COMMIT;
BEGIN; UPDATE cclf_beneficiaries SET new_id = id WHERE id between 8050976 and 9050976; COMMIT;
BEGIN; UPDATE cclf_beneficiaries SET new_id = id WHERE id between 9050977 and 10050977; COMMIT;
BEGIN; UPDATE cclf_beneficiaries SET new_id = id WHERE id between 10050978 and 11050978; COMMIT;
BEGIN; UPDATE cclf_beneficiaries SET new_id = id WHERE id between 11050979 and 12050979; COMMIT;
CREATE UNIQUE INDEX new_pk_idx on cclf_beneficiaries(new_id);
BEGIN;
ALTER TABLE cclf_beneficiaries drop constraint cclf_beneficiaries_pkey;
ALTER TABLE cclf_beneficiaries alter column new_id set default nextval('cclf_beneficiaries_id_seq'::regclass);
ALTER TABLE cclf_beneficiaries add constraint cclf_beneficiaries_pkey primary key using index new_pk_idx;
ALTER SEQUENCE cclf_beneficiaries_id_seq OWNED BY cclf_beneficiaries.new_id;
ALTER TABLE cclf_beneficiaries drop column id;
ALTER TABLE cclf_beneficiaries rename column new_id to id;
COMMIT;
-- Removes unused indexes found in the cclf_beneficiaries table
BEGIN;

DROP INDEX IF EXISTS idx_cclf_beneficiaries_bb_id;
DROP INDEX IF EXISTS idx_cclf_beneficiaries_mbi;

COMMIT;
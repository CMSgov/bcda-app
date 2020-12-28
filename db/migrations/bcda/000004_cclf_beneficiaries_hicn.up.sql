-- Remove HICN column from cclf_beneficiaries
BEGIN;
ALTER TABLE public.cclf_beneficiaries DROP COLUMN IF EXISTS hicn;
COMMIT;
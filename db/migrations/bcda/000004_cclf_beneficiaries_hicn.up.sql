-- Remove HICN column from cclf_beneficiaries
BEGIN;
ALTER TABLE public.cclf_beneficiaries DROP COLUMN IF EXISTS hicn;
COMMIT;

-- Remove HICN column from suppressions
BEGIN;
ALTER TABLE public.suppressions DROP COLUMN IF EXISTS hicn;
COMMIT;


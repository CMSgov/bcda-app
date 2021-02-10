-- Remove deleted_at column from acos
BEGIN;
ALTER TABLE public.acos DROP COLUMN deleted_at;
COMMIT;

-- Remove deleted_at column from cclf_beneficiaries
BEGIN;
ALTER TABLE public.cclf_beneficiaries DROP COLUMN deleted_at;
COMMIT;

-- Remove deleted_at column from cclf_files
BEGIN;
ALTER TABLE public.cclf_files DROP COLUMN deleted_at;
COMMIT;

-- Remove deleted_at column from job_keys
BEGIN;
ALTER TABLE public.job_keys DROP COLUMN deleted_at;
COMMIT;

-- Remove deleted_at column from jobs
BEGIN;
ALTER TABLE public.jobs DROP COLUMN deleted_at;
COMMIT;

-- Remove deleted_at column from suppressions
BEGIN;
ALTER TABLE public.suppressions DROP COLUMN deleted_at;
COMMIT;

-- Remove deleted_at column from suppression_files
BEGIN;
ALTER TABLE public.suppression_files DROP COLUMN deleted_at;
COMMIT;
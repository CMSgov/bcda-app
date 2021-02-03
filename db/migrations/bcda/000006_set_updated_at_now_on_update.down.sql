-- Remove set_timestamp trigger on acos
BEGIN;
DROP TRIGGER set_timestamp ON public.acos;
COMMIT;

-- Remove set_timestamp trigger on cclf_beneficiaries
BEGIN;
DROP TRIGGER set_timestamp ON public.cclf_beneficiaries;
COMMIT;

-- Remove set_timestamp trigger on cclf_files
BEGIN;
DROP TRIGGER set_timestamp ON public.cclf_files;
COMMIT;

-- Remove set_timestamp trigger on job_keys
BEGIN;
DROP TRIGGER set_timestamp ON public.job_keys;
COMMIT;

-- Remove set_timestamp trigger on suppressions
BEGIN;
DROP TRIGGER set_timestamp ON public.suppressions;
COMMIT;

-- Remove set_timestamp trigger on suppression_files
BEGIN;
DROP TRIGGER set_timestamp ON public.suppression_files;
COMMIT;

-- Remove trigger_set_timestamp function for updating updated_at to now
BEGIN;
DROP FUNCTION trigger_set_timestamp();
COMMIT;

-- Update updated_at column for acos
BEGIN;
ALTER TABLE public.acos ALTER COLUMN updated_at DROP DEFAULT;
COMMIT;

-- Update updated_at column for cclf_beneficiaries
BEGIN;
ALTER TABLE public.cclf_beneficiaries ALTER COLUMN updated_at DROP DEFAULT;
COMMIT;

-- Update updated_at column for cclf_files
BEGIN;
ALTER TABLE public.cclf_files ALTER COLUMN updated_at DROP DEFAULT;
COMMIT;

-- Update updated_at column for job_keys
BEGIN;
ALTER TABLE public.job_keys ALTER COLUMN updated_at DROP DEFAULT;
COMMIT;

-- Update updated_at column for suppressions
BEGIN;
ALTER TABLE public.suppressions ALTER COLUMN updated_at DROP DEFAULT;
COMMIT;

-- Update updated_at column for suppression_files
BEGIN;
ALTER TABLE public.suppression_files ALTER COLUMN updated_at DROP DEFAULT;
COMMIT;
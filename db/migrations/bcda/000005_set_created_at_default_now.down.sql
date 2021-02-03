-- Update created_at column for acos
BEGIN;
ALTER TABLE public.acos ALTER COLUMN created_at DROP DEFAULT;
COMMIT;

-- Update created_at column for cclf_beneficiaries
BEGIN;
ALTER TABLE public.cclf_beneficiaries ALTER COLUMN created_at DROP DEFAULT;
COMMIT;

-- Update created_at column for cclf_files
BEGIN;
ALTER TABLE public.cclf_files ALTER COLUMN created_at DROP DEFAULT;
COMMIT;

-- Update created_at column for job_keys
BEGIN;
ALTER TABLE public.job_keys ALTER COLUMN created_at DROP DEFAULT;
COMMIT;

-- Update created_at column for suppressions
BEGIN;
ALTER TABLE public.suppressions ALTER COLUMN created_at DROP DEFAULT;
COMMIT;

-- Update created_at column for suppression_files
BEGIN;
ALTER TABLE public.suppression_files ALTER COLUMN created_at DROP DEFAULT;
COMMIT;
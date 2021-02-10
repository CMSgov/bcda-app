-- Add deleted_at column to acos
BEGIN;
ALTER TABLE public.acos ADD COLUMN deleted_at timestamp with time zone;
COMMIT;

-- Add deleted_at column to cclf_beneficiaries
BEGIN;
ALTER TABLE public.cclf_beneficiaries ADD COLUMN deleted_at timestamp with time zone;
COMMIT;

-- Add deleted_at column to cclf_files
BEGIN;
ALTER TABLE public.cclf_files ADD COLUMN deleted_at timestamp with time zone;
COMMIT;

-- Add deleted_at column to job_keys
BEGIN;
ALTER TABLE public.job_keys ADD COLUMN deleted_at timestamp with time zone;
COMMIT;

-- Add deleted_at column to jobs
BEGIN;
ALTER TABLE public.jobs ADD COLUMN deleted_at timestamp with time zone;
COMMIT;

-- Add deleted_at column to suppressions
BEGIN;
ALTER TABLE public.suppressions ADD COLUMN deleted_at timestamp with time zone;
COMMIT;

-- Add deleted_at column to suppression_files
BEGIN;
ALTER TABLE public.suppression_files ADD COLUMN deleted_at timestamp with time zone;
COMMIT;
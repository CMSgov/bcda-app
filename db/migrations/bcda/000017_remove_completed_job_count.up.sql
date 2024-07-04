BEGIN;
-- Remove completed_job_count column from jobs table
ALTER TABLE public.jobs DROP COLUMN IF EXISTS completed_job_count;

COMMIT;
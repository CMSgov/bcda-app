BEGIN;
-- Add the completed_job_count column
ALTER TABLE public.jobs ADD COLUMN completed_job_count integer DEFAULT 0;
ALTER TABLE public.jobs ALTER COLUMN completed_job_count DROP DEFAULT;
COMMIT;
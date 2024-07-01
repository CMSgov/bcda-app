BEGIN;

ALTER TABLE public.job_keys DROP COLUMN IF EXISTS que_job_id;
DROP INDEX IF EXISTS idx_job_keys_job_id_que_job_id;

COMMIT;
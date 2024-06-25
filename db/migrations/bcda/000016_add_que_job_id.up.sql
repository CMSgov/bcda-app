-- Add que_job_id to job_keys table

BEGIN;

ALTER TABLE public.job_keys ADD COLUMN que_job_id bigint DEFAULT null;
CREATE INDEX IF NOT EXISTS idx_job_keys_job_id_que_job_id ON public.job_keys USING btree (job_id, que_job_id);

COMMIT;
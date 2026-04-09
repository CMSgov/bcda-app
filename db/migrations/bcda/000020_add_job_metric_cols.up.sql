-- Add useful job metrics to jobs table

BEGIN;

ALTER TABLE public.jobs ADD COLUMN benes_attributed_to_aco int DEFAULT 0;

ALTER TABLE public.job_keys ADD COLUMN benes_with_data int DEFAULT 0;
ALTER TABLE public.job_keys ADD COLUMN benes_retrieved_percent smallint DEFAULT 0;

COMMIT;

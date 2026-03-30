-- Drop job metrics

BEGIN;

ALTER TABLE public.jobs DROP COLUMN benes_attributed_to_aco int DEFAULT 0;

ALTER TABLE public.job_keys DROP COLUMN benes_with_data int DEFAULT 0;
ALTER TABLE public.job_keys DROP COLUMN benes_retrieved_percent smallint DEFAULT 0;

COMMIT;

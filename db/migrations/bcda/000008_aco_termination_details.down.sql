-- Remove column that captures ACO termination details
BEGIN;
ALTER TABLE public.acos DROP COLUMN IF EXISTS termination_details jsonb DEFAULT null;
COMMIT;
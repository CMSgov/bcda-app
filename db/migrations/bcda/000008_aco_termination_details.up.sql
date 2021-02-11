-- Add column that captures ACO termination details
BEGIN;
ALTER TABLE public.acos ADD COLUMN termination_details jsonb DEFAULT null;
COMMIT;
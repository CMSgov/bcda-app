BEGIN;
-- Set termination_details to null
UPDATE public.acos SET termination_details = NULL;

COMMIT;
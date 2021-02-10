BEGIN;
ALTER TABLE public.acos ADD COLUMN termination_details jsonb DEFAULT null;
COMMIT;
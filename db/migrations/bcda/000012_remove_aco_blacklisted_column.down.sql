BEGIN;
-- Add the blacklist column
ALTER TABLE public.acos ADD COLUMN blacklisted boolean DEFAULT false;

-- Set all blacklisted with termination data = true
UPDATE public.acos 
SET blacklisted = true;
WHERE termination_details IS NOT NULL;

COMMIT;
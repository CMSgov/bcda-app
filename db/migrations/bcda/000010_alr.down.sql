-- ALR uses two tables
BEGIN;
-- Remove the two triggers
DROP TRIGGER set_timestamp ON public.alr;
DROP TRIGGER set_timestamp ON public.alr_meta;

-- Drop the tables
DROP TABLE public.alr CASCADE;
DROP TABLE public.alr_meta CASCADE;
COMMIT;
-- Remove column associated with blacklisting a particular ACO
BEGIN;
ALTER TABLE public.acos DROP COLUMN IF EXISTS blacklisted;
COMMIT;
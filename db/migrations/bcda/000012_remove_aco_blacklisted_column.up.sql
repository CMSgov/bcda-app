BEGIN;
-- Remove blacklisted columns from aco table
ALTER TABLE public.acos DROP COLUMN IF EXISTS blacklisted;

COMMIT;
-- Adding column to track blacklisted ACOs
BEGIN;
ALTER TABLE public.acos ADD COLUMN blacklisted bool DEFAULT false;
COMMIT;
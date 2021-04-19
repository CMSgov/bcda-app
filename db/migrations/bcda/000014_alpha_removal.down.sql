BEGIN;

ALTER TABLE public.acos ADD COLUMN if not exists alpha_secret text;

COMMIT;
BEGIN;

ALTER TABLE public.acos
ADD COLUMN if not exists alpha_secret text
ADD COLUMN if not exists public_key text;

COMMIT;
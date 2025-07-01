-- Drop useful constraints

BEGIN;

ALTER TABLE public.acos DROP CONSTRAINT IF EXISTS acos_cms_id_key;

COMMIT;

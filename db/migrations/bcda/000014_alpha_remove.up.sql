-- With removal of alpha and okta, there is no need to have alpha_secret in acos

BEGIN;

ALTER TABLE public.acos
DROP COLUMN if exists alpha_secret
DROP COLUMN if exists public_key;

COMMIT;
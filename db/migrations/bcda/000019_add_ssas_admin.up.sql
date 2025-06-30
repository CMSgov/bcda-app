-- Add useful constraint (defined in the 000001_initial_schema file but seem to have gotten lost in various envs)

BEGIN;

ALTER TABLE public.acos DROP CONSTRAINT IF EXISTS acos_cms_id_key;
ALTER TABLE ONLY public.acos ADD CONSTRAINT acos_cms_id_key UNIQUE (cms_id);

COMMIT;

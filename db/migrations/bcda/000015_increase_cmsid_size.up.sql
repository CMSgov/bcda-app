-- Increasing the size of CMSID columns from 5 to 8

BEGIN;

ALTER table public.acos ALTER COLUMN cms_id type character varying(8);
ALTER table public.cclf_files ALTER COLUMN aco_cms_id type character varying(8);
ALTER table public.suppressions ALTER COLUMN aco_cms_id type character varying(8);

COMMIT;

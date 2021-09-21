BEGIN;

ALTER table public.acos ALTER COLUMN cms_id type character varying(5);
ALTER table public.cclf_files ALTER COLUMN aco_cms_id type character varying(5);
ALTER table public.suppressions ALTER COLUMN aco_cms_id type char(5);

COMMIT;

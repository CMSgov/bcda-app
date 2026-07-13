-- Revert acos and suppressions tables cms_id/aco_cms_id length to character varying(8)
BEGIN;

ALTER TABLE public.acos ALTER COLUMN cms_id TYPE varchar(8);
ALTER TABLE public.suppressions ALTER COLUMN aco_cms_id TYPE varchar(8);

COMMIT;

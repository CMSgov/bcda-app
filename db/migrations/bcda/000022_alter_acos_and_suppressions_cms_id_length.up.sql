-- Alter acos and suppressions tables to increase cms_id/aco_cms_id length
BEGIN;

ALTER TABLE public.acos ALTER COLUMN cms_id TYPE text;
ALTER TABLE public.suppressions ALTER COLUMN aco_cms_id TYPE text;

COMMIT;

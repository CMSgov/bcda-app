-- Alter cclf_files to increase aco_cms_id length

BEGIN;

ALTER TABLE public.cclf_files ALTER COLUMN aco_cms_id TYPE varchar(8);

COMMIT;

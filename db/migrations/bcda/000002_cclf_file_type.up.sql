-- Adding type column to cclf_files table to allow us to distinguish runout CCLF files
BEGIN;
ALTER TABLE public.cclf_files ADD COLUMN type smallint DEFAULT 0;
COMMIT;
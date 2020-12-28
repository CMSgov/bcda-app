-- Add HICN column to cclf_beneficiaries
BEGIN;
ALTER TABLE public.cclf_beneficiaries ADD COLUMN hicn varchar(11) NOT NULL;
COMMIT;

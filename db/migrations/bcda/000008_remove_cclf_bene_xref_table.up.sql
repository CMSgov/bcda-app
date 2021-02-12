BEGIN;
DROP TABLE public.cclf_beneficiary_xrefs CASCADE;
DROP SEQUENCE IF EXISTS public.cclf_beneficiary_xrefs_id_seq;
COMMIT;

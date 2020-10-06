-- This script will update is needed to support NGACOs since they only have 4 characters (V***)
-- Prior to this, we had assumed all CMS_IDs had 5 characters so char(5) and text fields worked as expected.
-- It is intended to be run on all environments (dev, testing, opensbx, and prod)
ALTER TABLE public.cclf_files ALTER COLUMN aco_cms_id SET DATA TYPE varchar(5);
ALTER TABLE public.acos ALTER COLUMN cms_id SET DATA TYPE varchar(5);
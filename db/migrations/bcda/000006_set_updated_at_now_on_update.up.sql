-- Update updated_at column for acos
BEGIN;
ALTER TABLE public.acos ALTER COLUMN updated_at SET DEFAULT now();
COMMIT;

-- Update updated_at column for cclf_beneficiaries
BEGIN;
ALTER TABLE public.cclf_beneficiaries ALTER COLUMN updated_at SET DEFAULT now();
COMMIT;

-- Update updated_at column for cclf_files
BEGIN;
ALTER TABLE public.cclf_files ALTER COLUMN updated_at SET DEFAULT now();
COMMIT;

-- Update updated_at column for job_keys
BEGIN;
ALTER TABLE public.job_keys ALTER COLUMN updated_at SET DEFAULT now();
COMMIT;

-- Update updated_at column for suppressions
BEGIN;
ALTER TABLE public.suppressions ALTER COLUMN updated_at SET DEFAULT now();
COMMIT;

-- Update updated_at column for suppression_files
BEGIN;
ALTER TABLE public.suppression_files ALTER COLUMN updated_at SET DEFAULT now();
COMMIT;

-- Create function which returns updated_at set to NOW in a record
BEGIN;
CREATE OR REPLACE FUNCTION trigger_set_timestamp()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;
COMMIT;

-- Trigger trigger_set_timestamp function on updated_at column for acos
BEGIN;
CREATE TRIGGER set_timestamp
BEFORE UPDATE ON public.acos
FOR EACH ROW
EXECUTE PROCEDURE trigger_set_timestamp();
COMMIT;

-- Trigger trigger_set_timestamp function on updated_at column for cclf_beneficiaries
BEGIN;
CREATE TRIGGER set_timestamp
BEFORE UPDATE ON public.cclf_beneficiaries
FOR EACH ROW
EXECUTE PROCEDURE trigger_set_timestamp();
COMMIT;

-- Trigger trigger_set_timestamp function on updated_at column for cclf_files
BEGIN;
CREATE TRIGGER set_timestamp
BEFORE UPDATE ON public.cclf_files
FOR EACH ROW
EXECUTE PROCEDURE trigger_set_timestamp();
COMMIT;

-- Trigger trigger_set_timestamp function on updated_at column for job_keys
BEGIN;
CREATE TRIGGER set_timestamp
BEFORE UPDATE ON public.job_keys
FOR EACH ROW
EXECUTE PROCEDURE trigger_set_timestamp();
COMMIT;

-- Trigger trigger_set_timestamp function on updated_at column for suppressions
BEGIN;
CREATE TRIGGER set_timestamp
BEFORE UPDATE ON public.suppressions
FOR EACH ROW
EXECUTE PROCEDURE trigger_set_timestamp();
COMMIT;

-- Trigger trigger_set_timestamp function on updated_at column for suppression_files
BEGIN;
CREATE TRIGGER set_timestamp
BEFORE UPDATE ON public.suppression_files
FOR EACH ROW
EXECUTE PROCEDURE trigger_set_timestamp();
COMMIT;
-- This SQL script is for bookkeeping and auditing purposes
-- This script can be used to remove the improved synthea data from opensbx

BEGIN;
DO $$

DECLARE Dev cclf_files.id%TYPE;
DECLARE Small cclf_files.id%TYPE;
DECLARE Large cclf_files.id%TYPE;

BEGIN

SELECT id FROM cclf_files WHERE aco_cms_id = 'A9989' INTO Dev;
SELECT id FROM cclf_files WHERE aco_cms_id = 'A9998' INTO Small;
SELECT id FROM cclf_files WHERE aco_cms_id = 'A9999' INTO Large;

DELETE FROM cclf_beneficiaries WHERE file_id IN (Dev, Small, Large);
DELETE FROM cclf_files WHERE id IN (Dev, Small, Large);

END $$;

COMMIT;

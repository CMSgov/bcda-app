-- This SQL script is for bookkeeping and auditing purposes
-- This script can be used to remove the improved synthea data from sandbox

BEGIN;
DO $$

DECLARE Dev INTEGER[];
DECLARE Small INTEGER[];
DECLARE Large INTEGER[];

BEGIN

SELECT array_agg(id) FROM cclf_files WHERE aco_cms_id = 'A9989' INTO Dev;
SELECT array_agg(id) FROM cclf_files WHERE aco_cms_id = 'A9998' INTO Small;
SELECT array_agg(id) FROM cclf_files WHERE aco_cms_id = 'A9999' INTO Large;

DELETE FROM cclf_beneficiaries WHERE file_id = ANY(Dev || Small || Large);
DELETE FROM cclf_files WHERE id = ANY(Dev || Small || Large);

END $$;

COMMIT;

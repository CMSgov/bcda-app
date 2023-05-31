-- This is a manual DB wrangle to UNDO the conversion of Dec CCLF attribution into runouts
-- This is for ACO REACH (cms_id LIKE 'D%') only, as they ingested the runout files (recd Feb 2023) in the database

BEGIN;
DO $$

DECLARE cclfids integer ARRAY;
BEGIN
cclfids := ARRAY(SELECT id FROM cclf_files WHERE timestamp > '2022-11-30' AND type = 1 AND timestamp < '2023-01-01' AND aco_cms_id LIKE ALL(ARRAY['D%']));

-- Compare this output with what you may have stored in Box
raise notice 'Following IDs affected: %', cclfids;
UPDATE cclf_files SET type = 0 where id = ANY(cclfids);
END;
$$;
COMMIT;

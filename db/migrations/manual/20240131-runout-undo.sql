-- This is a manual DB wrangle to convert Dec CCLF attribution into runouts
-- We will undo this and convert back to Dec CCLF once the runout for 2024 is received

BEGIN;
DO $$

DECLARE cclfids integer ARRAY;
BEGIN
cclfids := ARRAY(SELECT id FROM cclf_files WHERE timestamp > '2023-11-30' AND type = 1 AND timestamp < '2024-01-01' AND aco_cms_id NOT LIKE ALL(ARRAY['A999%', 'V99%', 'E999%']);

-- Compare this output with what you may have stored in keybase
raise notice 'Following IDs affected: %', cclfids;
UPDATE cclf_files SET type = 0 where id = ANY(cclfids);
END;

$$;

COMMIT;

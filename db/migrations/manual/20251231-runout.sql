-- This is a manual DB wrangle to convert Dec CCLF attribution into runouts
-- We will undo this and convert back to Dec CCLF once the runout for 2026 is received

BEGIN;
DO $$

DECLARE cclfids integer ARRAY;
BEGIN
cclfids := ARRAY(SELECT id FROM cclf_files WHERE timestamp > '2025-11-30' AND type = 0 AND aco_cms_id NOT LIKE ALL(ARRAY['V99%', 'E999%', 'DA999%']));
-- This array should ONLY have type == 0

-- You can store the array in Box and ensure the right IDs are affected when undoing this
raise notice 'Following IDs affected: %', cclfids;
-- value of 1 is runout, and 0 is not
UPDATE cclf_files SET type = 1 where id = ANY(cclfids);
END;

$$;

COMMIT;
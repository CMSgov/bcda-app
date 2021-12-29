-- This is a manual DB wrangle to convert Dec CCLF attribution into runouts
-- We will undo this and convert back to Dec CCLF once the runout for 2022 is received

BEGIN;
DO $$

DECLARE cclfids integer ARRAY;
BEGIN
cclfids := ARRAY(SELECT id FROM cclf_files WHERE timestamp > '2021-11-30' AND aco_cms_id NOT LIKE ALL(ARRAY['A999%', 'V99%', 'E999%']);
-- This array should ONLY have type == 0

-- You can check the array matches with the array you may have stored in keybase when you ran
-- 20211229-runout.sql
raise notice 'Following IDs affected: %', cclfids;
-- value of 1 is runout, and 0 is not
UPDATE cclf_files SET type = 1 where id = ANY(cclfids);
END;

$$;

COMMIT;

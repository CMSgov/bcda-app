-- This is a manual migration to update one of the two CCLF files in cclf_files for PACA ACO in Prod.
-- Both files currently have the 'ZC8Y23' format and the CCLF8 file should have the 'ZC8R23' format for successful smoke test pipeline runs.


-- update the CCLF8 to use R instead of Y in ZC8(R|Y)23
BEGIN;
UPDATE cclf_files SET name = 'T.BCD.TEST993.ZC8R23.D230427.T1057310' WHERE id = 33721;
COMMIT;
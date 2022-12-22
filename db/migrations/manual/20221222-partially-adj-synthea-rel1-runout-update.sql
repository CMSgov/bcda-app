-- Updating the sandbox runout files for partially adjudicated TEST99x accounts
-- to allow runout testing.

BEGIN;
UPDATE cclf_files
    set type=1,
        updated_at=now(),
        timestamp='2022-09-29',
        performance_year=22
    where
        cclf_num=8
        and name in ('T.BCD.TEST990.ZC8R21.D210929.T1834180','T.BCD.TEST991.ZC8R21.D210929.T1834180','T.BCD.TEST992.ZC8R21.D210929.T1834180')
        and type=0;
COMMIT;

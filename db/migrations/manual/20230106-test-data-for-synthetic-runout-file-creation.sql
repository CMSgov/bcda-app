--Fake data to test out the synthetic-runout-file-creation script in a lower env!
--model types for test: 'X%' or 'Y%' or 'Z%'

BEGIN;
DO $$
DECLARE x_aco_1_november cclf_files.id%TYPE;

DECLARE x_aco_1 cclf_files.id%TYPE;
DECLARE x_aco_2 cclf_files.id%TYPE;
DECLARE x_aco_3 cclf_files.id%TYPE;

DECLARE y_aco_1 cclf_files.id%TYPE;
DECLARE y_aco_2 cclf_files.id%TYPE;
DECLARE y_aco_3 cclf_files.id%TYPE;

DECLARE y_aco_3_january cclf_files.id%TYPE;

DECLARE z_aco_1_first_dec cclf_files.id%TYPE;
DECLARE z_aco_1_second_dec cclf_files.id%TYPE;

DECLARE testcclfids INTEGER[];

BEGIN

INSERT INTO acos (uuid, name, cms_id)
VALUES ('6dfd646f-8f83-480d-a15f-8ca28146d74a', 'ACO X0001 for Synthetic Runout Script Test', 'X0001'),
('6dfd646f-8f83-480d-a15f-8ca28146d74b', 'ACO X0002 for Synthetic Runout Script Test', 'X0002'),
('6dfd646f-8f83-480d-a15f-8ca28146d74c', 'ACO X0003 for Synthetic Runout Script Test', 'X0003'),
('6dfd646f-8f83-480d-a15f-8ca28146d74d', 'ACO Y0001 for Synthetic Runout Script Test', 'Y0001'),
('6dfd646f-8f83-480d-a15f-8ca28146d74e', 'ACO Y0002 for Synthetic Runout Script Test', 'Y0002'),
('6dfd646f-8f83-480d-a15f-8ca28146d74f', 'ACO Y0003 for Synthetic Runout Script Test', 'Y0003'),
('6dfd646f-8f83-480d-a15f-8ca28146d75a', 'ACO Z0001 for Synthetic Runout Script Test', 'Z0001');

--November file for X0001, should not be targeted 
INSERT INTO public.cclf_files (created_at, updated_at, cclf_num, name, aco_cms_id, timestamp, performance_year, import_status, "type")
 VALUES ('2022-11-01',NOW(),8,'P.X0001.ACO.ZC8R22.D221101.FAKE-DATA-FOR-SYN-RUNOUT-SCRIPT-TEST','X0001', NOW(),22,'Completed',0) 
 RETURNING id INTO x_aco_1_november;

INSERT INTO public.cclf_files (created_at, updated_at, cclf_num, name, aco_cms_id, timestamp, performance_year, import_status, "type")
 VALUES ('2022-12-02',NOW(),8,'P.X0001.ACO.ZC8R22.D221231.FAKE-DATA-FOR-SYN-RUNOUT-SCRIPT-TEST','X0001', NOW(),22,'Completed',0) 
 RETURNING id INTO x_aco_1;

INSERT INTO public.cclf_files (created_at, updated_at, cclf_num, name, aco_cms_id, timestamp, performance_year, import_status, "type")
 VALUES ('2022-12-06',NOW(),8,'P.X0002.ACO.ZC8R22.D221231.FAKE-DATA-FOR-SYN-RUNOUT-SCRIPT-TEST','X0002', NOW(),22,'Completed',0) 
 RETURNING id INTO x_aco_2;

INSERT INTO public.cclf_files (created_at, updated_at, cclf_num, name, aco_cms_id, timestamp, performance_year, import_status, "type")
 VALUES ('2022-12-10',NOW(),8,'P.X0003.ACO.ZC8R22.D221231.FAKE-DATA-FOR-SYN-RUNOUT-SCRIPT-TEST','X0003', NOW(),22,'Completed',0) 
 RETURNING id INTO x_aco_3;

INSERT INTO public.cclf_files (created_at, updated_at, cclf_num, name, aco_cms_id, timestamp, performance_year, import_status, "type")
 VALUES ('2022-12-03',NOW(),8,'P.Y0001.ACO.ZC8R22.D221231.FAKE-DATA-FOR-SYN-RUNOUT-SCRIPT-TEST','Y0001', NOW(),22,'Completed',0) 
 RETURNING id INTO y_aco_1;

INSERT INTO public.cclf_files (created_at, updated_at, cclf_num, name, aco_cms_id, timestamp, performance_year, import_status, "type")
 VALUES ('2022-12-07',NOW(),8,'P.Y0002.ACO.ZC8R22.D221231.FAKE-DATA-FOR-SYN-RUNOUT-SCRIPT-TEST','Y0002', NOW(),22,'Completed',0)
 RETURNING id INTO y_aco_2;

INSERT INTO public.cclf_files (created_at, updated_at, cclf_num, name, aco_cms_id, timestamp, performance_year, import_status, "type")
 VALUES ('2022-12-11',NOW(),8,'P.Y0003.ACO.ZC8R22.D221231.FAKE-DATA-FOR-SYN-RUNOUT-SCRIPT-TEST','Y0003', NOW(),22,'Completed',0) 
 RETURNING id INTO y_aco_3;

--January file for Y0003, should not be targeted
INSERT INTO public.cclf_files (created_at, updated_at, cclf_num, name, aco_cms_id, timestamp, performance_year, import_status, "type")
 VALUES ('2023-01-01',NOW(),8,'P.Y0003.ACO.ZC8R23.D230101.FAKE-DATA-FOR-SYN-RUNOUT-SCRIPT-TEST','Y0003', NOW(),23,'Completed',0) 
 RETURNING id INTO y_aco_3_january;

 --December file #1 for Z0001, script should abort for more than 1 Dec file
INSERT INTO public.cclf_files (created_at, updated_at, cclf_num, name, aco_cms_id, timestamp, performance_year, import_status, "type")
 VALUES ('2022-12-04',NOW(),8,'P.Z0001.ACO.ZC8R22.D221204.FAKE-DATA-FOR-SYN-RUNOUT-SCRIPT-TEST','Z0001', NOW(),22,'Completed',0) 
 RETURNING id INTO z_aco_1_first_dec;

 --December file #2 for Z0001, script should abort for more than 1 Dec file
INSERT INTO public.cclf_files (created_at, updated_at, cclf_num, name, aco_cms_id, timestamp, performance_year, import_status, "type")
 VALUES ('2022-12-05',NOW(),8,'P.Z0001.ACO.ZC8R22.D221105.FAKE-DATA-FOR-SYN-RUNOUT-SCRIPT-TEST','Z0001', NOW(),22,'Completed',0) 
 RETURNING id INTO z_aco_1_second_dec;

INSERT INTO cclf_beneficiaries (file_id, mbi)
 VALUES
    (x_aco_1_november, 'XBADIDNOV00'), --should not be picked up - from november file for cms id X0001
    (x_aco_1, 'XVALID11111'),
    (x_aco_2, 'XVALID12222'),
    (x_aco_2, 'XYVALID2222'), --same mbi appears in y_aco_2
    (x_aco_3, 'XVALID13333'),
    (x_aco_3, 'XVALID23333'),
    (x_aco_3, 'XVALID33333'),
    (y_aco_1, 'YVALID11111'),
    (y_aco_2, 'YVALID12222'),
    (y_aco_2, 'XYVALID2222'), --same mbi appears in x_aco_2
    (y_aco_3, 'YVALID13333'),
    (y_aco_3, 'YVALID23333'),
    (y_aco_3, 'YVALID33333'),
    (y_aco_3_january, 'YBADIDNOV00'), --should not be picked up - from january file for cms id Y0003
    (z_aco_1_first_dec, 'ZBADIDDEC01'), --should not be picked up - script should abort for more than 1 Dec file when running 'Z%'
    (z_aco_1_second_dec, 'ZBADIDDEC02'); --should not be picked up - script should abort for more than 1 Dec file when running 'Z%'

testcclfids := ARRAY[x_aco_1_november,x_aco_1,x_aco_2,x_aco_3,y_aco_1,y_aco_2,y_aco_3,y_aco_3_january, z_aco_1_first_dec, z_aco_1_second_dec];

RAISE NOTICE 'Following CCLF IDs created for synthetic runout file testing: %', testcclfids;

END;
$$;
COMMIT;

-----------------------------------------------------------------------------------------
-- to confirm or remove the test records:

-- select * from acos WHERE name like '%Synthetic Runout%';
-- DELETE from acos WHERE name like '%Synthetic Runout%';
-- select * from cclf_files where name LIKE '%TEST'
-- DELETE from cclf_files where name LIKE '%TEST'
-- select * from cclf_beneficiaries where file_id in (select id from cclf_files where name LIKE '%TEST')
-- DELETE from cclf_beneficiaries where file_id in (select id from cclf_files where name LIKE '%TEST')


-- to confirm or remove the generated synthetic runout files and benes:

-- select * from cclf_files where name LIKE '%221231.SYNTHETICRUNOUTFILE'
-- DELETE from cclf_files where name LIKE '%221231.SYNTHETICRUNOUTFILE'

-- select * from cclf_beneficiaries where file_id in (select id from cclf_files where name LIKE '%221231.SYNTHETICRUNOUTFILE')
-- DELETE from cclf_beneficiaries where file_id in (select id from cclf_files where name LIKE '%221231.SYNTHETICRUNOUTFILE')

-----------------------------------------------------------------------------------------

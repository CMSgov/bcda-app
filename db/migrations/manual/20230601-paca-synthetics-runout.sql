-- This is a manual migration to attribute existing synthetic MBIs to a new ACO to be used for PACA smoke tests (BCDA-6873)
-- We will be copying the MBIs from ACO A9994 (opensbx) runout CCLF file to ACO TEST993 since those are also used for smoke tests 

BEGIN;
DO $$
DECLARE PACA cclf_files.id%TYPE;


-- Adding one row into cclf_files for one new CCLF file (runtout) associated to synthea 
BEGIN
INSERT INTO cclf_files (created_at, updated_at, cclf_num, name, aco_cms_id, timestamp, performance_year, import_status, type) VALUES (now(), now(), 8, 'T.BCD.TEST993.ZC8Y23.D230601.T1322310', 'TEST993', CURRENT_DATE, 23, 'Completed', 1) RETURNING id INTO PACA;


-- Adding rows into cclf_beneficiaries for TEST993
INSERT INTO cclf_beneficiaries (file_id, mbi)
VALUES

        (PACA, '1SG0A00AA00'),
        (PACA, '1SH0A00AA00'),
        (PACA, '1SJ0A00AA00'),
        (PACA, '1SK0A00AA00'),
        (PACA, '1SM0A00AA00'),
        (PACA, '2SG0A00AA00'),
        (PACA, '2SH0A00AA00'),
        (PACA, '2SJ0A00AA00'),
        (PACA, '2SK0A00AA00'),
        (PACA, '2SM0A00AA00'),
        (PACA, '3SG0A00AA00'),
        (PACA, '3SH0A00AA00'),
        (PACA, '3SJ0A00AA00'),
        (PACA, '3SK0A00AA00'),
        (PACA, '3SM0A00AA00'),
        (PACA, '4SG0A00AA00'),
        (PACA, '4SH0A00AA00'),
        (PACA, '4SJ0A00AA00'),
        (PACA, '4SK0A00AA00'),
        (PACA, '4SM0A00AA00'),
        (PACA, '5SG0A00AA00'),
        (PACA, '5SH0A00AA00'),
        (PACA, '5SJ0A00AA00'),
        (PACA, '5SK0A00AA00'),
        (PACA, '6SG0A00AA00'),
        (PACA, '6SH0A00AA00'),
        (PACA, '6SJ0A00AA00'),
        (PACA, '6SK0A00AA00'),
        (PACA, '7SG0A00AA00'),
        (PACA, '7SH0A00AA00'),
        (PACA, '7SJ0A00AA00'),
        (PACA, '7SK0A00AA00'),
        (PACA, '8SG0A00AA00'),
        (PACA, '8SH0A00AA00'),
        (PACA, '8SJ0A00AA00'),
        (PACA, '8SK0A00AA00'),
        (PACA, '9SG0A00AA00'),
        (PACA, '9SH0A00AA00'),
        (PACA, '9SJ0A00AA00'),
        (PACA, '9SK0A00AA00');
END $$;
COMMIT;

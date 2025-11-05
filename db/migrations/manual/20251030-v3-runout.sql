BEGIN;
DO $$
DECLARE FID cclf_files.id%TYPE;


BEGIN
INSERT INTO cclf_files (created_at, updated_at, cclf_num, name, aco_cms_id, timestamp, performance_year, import_status, type) VALUES (now(), now(), 8, 'T.BCD.TEST994.ZC8R25.D251030.T1322310', 'TEST994', CURRENT_DATE, 24, 'Completed', 1) RETURNING id INTO FID;


-- Adding rows into cclf_beneficiaries for TEST994 (V3 Test ACO)
INSERT INTO cclf_beneficiaries (file_id, mbi)
VALUES
        (FID, '1S00E00AA17'),
        (FID, '1S00E00AA34'),
        (FID, '1S00E00AA05'),
        (FID, '1S00E00AA45');

END $$;
COMMIT;

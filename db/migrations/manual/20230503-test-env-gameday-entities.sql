
-- This SQL script is for bookkeeping and auditing purposes
-- This script adds four entities associated with the Dev MBIs from the 20221019-partially-adj-synthea-rel2-new.sql file.

BEGIN;
DO $$
DECLARE DCE1 cclf_files.id%TYPE;
DECLARE DCE2 cclf_files.id%TYPE;
DECLARE DCE3 cclf_files.id%TYPE;
DECLARE ACO1 cclf_files.id%TYPE;

-- Adding three rows into cclf_files for three new CCLF files associated to synthea (improved data)
BEGIN
INSERT INTO cclf_files (created_at, updated_at, cclf_num, name, aco_cms_id, timestamp, performance_year, import_status, type) VALUES (now(), now(), 8, 'T.BCD.D9991.ZC8Y23.D230503.T0000000', 'D9990', '2023-05-03', 23, 'Completed', 0) RETURNING id INTO DCE1;
INSERT INTO cclf_files (created_at, updated_at, cclf_num, name, aco_cms_id, timestamp, performance_year, import_status, type) VALUES (now(), now(), 8, 'T.BCD.D9991.ZC8Y23.D230503.T0000000', 'D9991', '2023-05-03', 23, 'Completed', 0) RETURNING id INTO DCE2;
INSERT INTO cclf_files (created_at, updated_at, cclf_num, name, aco_cms_id, timestamp, performance_year, import_status, type) VALUES (now(), now(), 8, 'T.BCD.D9992.ZC8Y23.D230503.T0000000', 'D9992', '2023-05-03', 23, 'Completed', 0) RETURNING id INTO DCE3;
INSERT INTO cclf_files (created_at, updated_at, cclf_num, name, aco_cms_id, timestamp, performance_year, import_status, type) VALUES (now(), now(), 8, 'T.BCD.A9996.ZC8Y23.D230503.T0000000', 'A9996', '2023-05-03', 23, 'Completed', 0) RETURNING id INTO ACO1;

-- Adding rows into cclf_beneficiaries for ####
INSERT INTO cclf_beneficiaries (file_id, mbi)
VALUES
    (DCE1, '1S00E00GJ60'),
    (DCE1, '1S00E00HF10'),
    (DCE1, '1S00E00HK93'),
    (DCE1, '1S00E00HG53'),
    (DCE1, '1S00E00GH62'),
    (DCE1, '1S00E00HD78'),
    (DCE1, '1S00E00HF98'),
    (DCE1, '1S00E00HF54'),
    (DCE1, '1S00E00MM82'),
    (DCE1, '1S00E00MF89'),
    (DCE1, '1S00E00ME68'),
    (DCE1, '1S00E00MD47'),
    (DCE1, '1S00E00MG66'),
    (DCE1, '1S00E00MG22'),
    (DCE1, '1S00E00MF45'),
    (DCE1, '1S00E00ME24'),
    (DCE1, '1S00E00GK81'),
    (DCE1, '1S00E00HA37'),
    (DCE1, '1S00E00HC13'),
    (DCE1, '1S00E00HD56'),
    (DCE1, '1S00E00HJ72'),
    (DCE1, '1S00E00KJ84'),
    (DCE1, '1S00E00MJ63'),
    (DCE1, '1S00E00MK40'),
    (DCE1, '1S00E00MK84'),
    (DCE1, '1S00E00MK62'),
    (DCE1, '1S00E00HC79'),
    (DCE1, '1S00E00HD34'),
    (DCE1, '1S00E00HD12'),
    (DCE1, '1S00E00HC35'),
    (DCE1, '1S00E00HC57'),
    (DCE1, '1S00E00HA15'),
    (DCE1, '1S00E00HA59'),
    (DCE1, '1S00E00MF23'),
    (DCE1, '1S00E00MF67'),
    (DCE1, '1S00E00MG00'),
    (DCE1, '1S00E00MG44'),
    (DCE1, '1S00E00MG88'),
    (DCE1, '1S00E00MM60'),
    (DCE1, '1S00E00MN81');

-- Adding rows into cclf_beneficiaries for ####
INSERT INTO cclf_beneficiaries (file_id, mbi)
VALUES
    (DCE2, '1S00E00GJ60'),
    (DCE2, '1S00E00HF10'),
    (DCE2, '1S00E00HK93'),
    (DCE2, '1S00E00HG53'),
    (DCE2, '1S00E00GH62'),
    (DCE2, '1S00E00HD78'),
    (DCE2, '1S00E00HF98'),
    (DCE2, '1S00E00HF54'),
    (DCE2, '1S00E00MM82'),
    (DCE2, '1S00E00MF89'),
    (DCE2, '1S00E00ME68'),
    (DCE2, '1S00E00MD47'),
    (DCE2, '1S00E00MG66'),
    (DCE2, '1S00E00MG22'),
    (DCE2, '1S00E00MF45'),
    (DCE2, '1S00E00ME24'),
    (DCE2, '1S00E00GK81'),
    (DCE2, '1S00E00HA37'),
    (DCE2, '1S00E00HC13'),
    (DCE2, '1S00E00HD56'),
    (DCE2, '1S00E00HJ72'),
    (DCE2, '1S00E00KJ84'),
    (DCE2, '1S00E00MJ63'),
    (DCE2, '1S00E00MK40'),
    (DCE2, '1S00E00MK84'),
    (DCE2, '1S00E00MK62'),
    (DCE2, '1S00E00HC79'),
    (DCE2, '1S00E00HD34'),
    (DCE2, '1S00E00HD12'),
    (DCE2, '1S00E00HC35'),
    (DCE2, '1S00E00HC57'),
    (DCE2, '1S00E00HA15'),
    (DCE2, '1S00E00HA59'),
    (DCE2, '1S00E00MF23'),
    (DCE2, '1S00E00MF67'),
    (DCE2, '1S00E00MG00'),
    (DCE2, '1S00E00MG44'),
    (DCE2, '1S00E00MG88'),
    (DCE2, '1S00E00MM60'),
    (DCE2, '1S00E00MN81');

-- Adding rows into cclf_beneficiaries for ####
INSERT INTO cclf_beneficiaries (file_id, mbi)
VALUES
    (DCE3, '1S00E00GJ60'),
    (DCE3, '1S00E00HF10'),
    (DCE3, '1S00E00HK93'),
    (DCE3, '1S00E00HG53'),
    (DCE3, '1S00E00GH62'),
    (DCE3, '1S00E00HD78'),
    (DCE3, '1S00E00HF98'),
    (DCE3, '1S00E00HF54'),
    (DCE3, '1S00E00MM82'),
    (DCE3, '1S00E00MF89'),
    (DCE3, '1S00E00ME68'),
    (DCE3, '1S00E00MD47'),
    (DCE3, '1S00E00MG66'),
    (DCE3, '1S00E00MG22'),
    (DCE3, '1S00E00MF45'),
    (DCE3, '1S00E00ME24'),
    (DCE3, '1S00E00GK81'),
    (DCE3, '1S00E00HA37'),
    (DCE3, '1S00E00HC13'),
    (DCE3, '1S00E00HD56'),
    (DCE3, '1S00E00HJ72'),
    (DCE3, '1S00E00KJ84'),
    (DCE3, '1S00E00MJ63'),
    (DCE3, '1S00E00MK40'),
    (DCE3, '1S00E00MK84'),
    (DCE3, '1S00E00MK62'),
    (DCE3, '1S00E00HC79'),
    (DCE3, '1S00E00HD34'),
    (DCE3, '1S00E00HD12'),
    (DCE3, '1S00E00HC35'),
    (DCE3, '1S00E00HC57'),
    (DCE3, '1S00E00HA15'),
    (DCE3, '1S00E00HA59'),
    (DCE3, '1S00E00MF23'),
    (DCE3, '1S00E00MF67'),
    (DCE3, '1S00E00MG00'),
    (DCE3, '1S00E00MG44'),
    (DCE3, '1S00E00MG88'),
    (DCE3, '1S00E00MM60'),
    (DCE3, '1S00E00MN81');

-- Adding rows into cclf_beneficiaries for ####
INSERT INTO cclf_beneficiaries (file_id, mbi)
VALUES
    (ACO1, '1S00E00GJ60'),
    (ACO1, '1S00E00HF10'),
    (ACO1, '1S00E00HK93'),
    (ACO1, '1S00E00HG53'),
    (ACO1, '1S00E00GH62'),
    (ACO1, '1S00E00HD78'),
    (ACO1, '1S00E00HF98'),
    (ACO1, '1S00E00HF54'),
    (ACO1, '1S00E00MM82'),
    (ACO1, '1S00E00MF89'),
    (ACO1, '1S00E00ME68'),
    (ACO1, '1S00E00MD47'),
    (ACO1, '1S00E00MG66'),
    (ACO1, '1S00E00MG22'),
    (ACO1, '1S00E00MF45'),
    (ACO1, '1S00E00ME24'),
    (ACO1, '1S00E00GK81'),
    (ACO1, '1S00E00HA37'),
    (ACO1, '1S00E00HC13'),
    (ACO1, '1S00E00HD56'),
    (ACO1, '1S00E00HJ72'),
    (ACO1, '1S00E00KJ84'),
    (ACO1, '1S00E00MJ63'),
    (ACO1, '1S00E00MK40'),
    (ACO1, '1S00E00MK84'),
    (ACO1, '1S00E00MK62'),
    (ACO1, '1S00E00HC79'),
    (ACO1, '1S00E00HD34'),
    (ACO1, '1S00E00HD12'),
    (ACO1, '1S00E00HC35'),
    (ACO1, '1S00E00HC57'),
    (ACO1, '1S00E00HA15'),
    (ACO1, '1S00E00HA59'),
    (ACO1, '1S00E00MF23'),
    (ACO1, '1S00E00MF67'),
    (ACO1, '1S00E00MG00'),
    (ACO1, '1S00E00MG44'),
    (ACO1, '1S00E00MG88'),
    (ACO1, '1S00E00MM60'),
    (ACO1, '1S00E00MN81');

END $$;
COMMIT;
-- This is a manual migration to attribute existing synthetic MBIs to a new ACO to be used for Previous PY smoke tests (BCDA-9765)
-- We will be copying the MBIs from ACO A9994 (sandbox) to ACO TEST995 since those are also used for smoke tests
-- Adding one row into cclf_files for one new CCLF file and one runout file associated with Previous PY ACO
BEGIN;

WITH NEW_FILES AS (
    INSERT INTO
        cclf_files (
            created_at,
            updated_at,
            cclf_num,
            name,
            aco_cms_id,
            timestamp,
            performance_year,
            import_status,
            type
        )
    VALUES
        (
            NOW(),
            NOW(),
            8,
            'T.BCD.TEST995.ZC8Y12.D251106.T1701110',
            'TEST995',
            CURRENT_DATE,
            25,
            'Completed',
            0
        ),
        (
            NOW(),
            NOW(),
            8,
            'T.BCD.TEST995.ZC8R12.D251106.T1701140',
            'TEST995',
            CURRENT_DATE,
            25,
            'Completed',
            1
        ) RETURNING id
) -- Adding rows into cclf_beneficiaries for TEST993
INSERT INTO
    cclf_beneficiaries (file_id, mbi)
SELECT
    id,
    '4SJ0A00AA00'
FROM
    NEW_FILES;

COMMIT;
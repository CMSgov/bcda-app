-- This script will generate synthetic runout file data (cclf_files and cclf_beneficiaries records)
-- based on the December file of the prior performance year for identified model entity types that
-- will not have a generated runout file delivered for ingestion by BCDA.
--
-- Required: 
--      (1) Dec 2022 CCLF files have already been delivered & ingested for the model entity type
--      (2) model entity cannot be termed (is active)
--
-- The performance year can be switched in the DECLARE section for reusability, just update the variables.

BEGIN;

DO $$ 
DECLARE
model_type VARCHAR(8) := 'Y%'; -- model type to target. KCC model entity type (KCE and KCF) need to do 'K%' and 'C%'
december_range_start_date cclf_files.created_at%TYPE := '2022-11-30';
december_range_end_date cclf_files.created_at%TYPE := '2022-12-31';
file_performance_year cclf_files.performance_year%TYPE := 22;
synthetic_file_name_convention cclf_files.name%TYPE := CONCAT('.ACO.ZC8R','22','.D','221231','.SYNTHETICRUNOUTFILE');

items RECORD;
affected_aco_count INTEGER;
total_synthetic_file_count INTEGER;
total_mbi_count INTEGER;

BEGIN

--Step 1: Retrieve & store affected ACO IDs
CREATE TEMPORARY TABLE temp_affected_acos(
	aco_cms_id VARCHAR(8),
    dec_file_count integer
);

INSERT INTO temp_affected_acos
SELECT ac.cms_id, count(ac.cms_id)
FROM acos ac
INNER JOIN cclf_files f ON ac.cms_id = f.aco_cms_id
WHERE ac.termination_details IS NULL
	AND (ac.cms_id LIKE model_type)
	AND f.created_at BETWEEN december_range_start_date AND december_range_end_date
GROUP BY ac.cms_id;

SELECT count(*) INTO affected_aco_count FROM temp_affected_acos;

RAISE NOTICE '# of cms ids identified for synthetic runout generation: %', affected_aco_count;
FOR items IN 
    SELECT * FROM temp_affected_acos 
LOOP
    RAISE INFO 'aco_cms_id: %, dec_file_count: %', items.aco_cms_id, items.dec_file_count;
END LOOP;

--verify for each entity that they only have 1 December CCLF file that was ingested for it.
IF (SELECT count(dec_file_count) FROM temp_affected_acos WHERE dec_file_count > 1) > 0 THEN
RAISE EXCEPTION 'Aborting: Found more than 1 cclf file record per aco_cms_id in December!';
END IF;

--Step 2: create synthetic runout file records in cclf_files & obtain the new generated file id for each entity
CREATE TEMPORARY TABLE temp_runout_files(
	aco_cms_id VARCHAR(8),
    synthetic_runout_file_id integer
);

WITH result AS (
    INSERT INTO public.cclf_files (created_at, updated_at, cclf_num, name, aco_cms_id, timestamp, performance_year, import_status, "type") 
        SELECT NOW(),NOW(),8, CONCAT('P.',aco_cms_id,synthetic_file_name_convention), aco_cms_id, NOW(),file_performance_year,'Completed',1
        FROM temp_affected_acos
    RETURNING aco_cms_id, id
    )
INSERT INTO temp_runout_files(aco_cms_id, synthetic_runout_file_id)
    SELECT aco_cms_id, id FROM result;
	
SELECT count(*) INTO total_synthetic_file_count FROM temp_runout_files;

RAISE NOTICE 'total synthetic file ids generated: %', total_synthetic_file_count;

--verify count of affected entities matches count of synthetic records generated
IF affected_aco_count != total_synthetic_file_count THEN
RAISE EXCEPTION 'Aborting: affected_aco_count does not equal total_synthetic_file_count!';
END IF;

RAISE INFO ' ----- new synthetic file ids per entity ----- ';
FOR items IN 
    SELECT * FROM temp_runout_files
LOOP
    RAISE INFO 'aco_cms_id: %, synthetic_runout_file_id: %', items.aco_cms_id, items.synthetic_runout_file_id;
END LOOP;

--Step 3: collect MBI's from prior December 2022 file that we need to associate to the new synthetic runout file, by entity
CREATE TEMPORARY TABLE temp_runout_benes(
	aco_cms_id VARCHAR(8),
   december_file_id integer,
	december_mbi character(11),
    runout_file_id integer
);

INSERT INTO temp_runout_benes
    SELECT f.aco_cms_id,
            b.file_id AS december_file_id,
            b.mbi AS december_mbi,
            rf.synthetic_runout_file_id AS runout_file_id
    FROM temp_affected_acos aa
    INNER JOIN cclf_files f ON aa.aco_cms_id = f.aco_cms_id
    INNER JOIN cclf_beneficiaries b ON b.file_id = f.id
    INNER JOIN temp_runout_files rf ON rf.aco_cms_id = f.aco_cms_id
	WHERE f.created_at BETWEEN december_range_start_date AND december_range_end_date
    ORDER BY f.aco_cms_id ASC;

SELECT count(*) INTO total_mbi_count FROM temp_runout_benes;

RAISE NOTICE 'Total affected MBI count: %', total_mbi_count;
RAISE INFO ' ----- MBI count by entity ----- ';

FOR items IN 
    SELECT aco_cms_id, runout_file_id, count(december_mbi) AS mbi_count
    FROM temp_runout_benes 
    GROUP BY aco_cms_id, runout_file_id
LOOP
    RAISE INFO 'aco_cms_id: %, synthetic_runout_file_id: %, mbi count: %', items.aco_cms_id, items.runout_file_id, items.mbi_count;
END LOOP;

--Step 4: Insert copied bene MBIs in cclf_beneficiaries table that will relate to the synthetic runout file id
INSERT INTO cclf_beneficiaries (file_id, mbi)
    SELECT runout_file_id, december_mbi 
    FROM temp_runout_benes;

--Step 5: Cleanup - drop temp tables used in process (should be gone by end of transaction/session)
 DROP TABLE temp_runout_benes;
 DROP TABLE temp_runout_files;
 DROP TABLE temp_affected_acos;

END;
$$;
COMMIT;
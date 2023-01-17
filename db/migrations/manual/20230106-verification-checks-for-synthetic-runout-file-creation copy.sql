--Verification script #1: confirm no differences in mbis from the generated synthetic runout and dec file.
--If result rows are returned, this means there is a discrepency between the two files
--and a correction would need to be made so the benes on the synthetic file are a 1:1
--of the benes that are on the Dec 2022 CCLF file. 
--This script does NOT account for duplicates (a bene appearing twice on a file), see 2nd script below.

WITH DecCCLFBenesResult AS (
	SELECT files.aco_cms_id,
		benes.mbi
	FROM acos ac
	INNER JOIN cclf_files files ON ac.cms_id = files.aco_cms_id
	INNER JOIN cclf_beneficiaries benes ON benes.file_id = files.id
	WHERE ac.termination_details IS NULL
	AND (ac.cms_id LIKE 'X%' OR ac.cms_id LIKE 'Y%' OR ac.cms_id LIKE 'Z%')  --test data 
	--AND (ac.cms_id LIKE 'K%' OR ac.cms_id LIKE 'C%') -- KCC model entity type (KCE and KCF)
	AND files.created_at BETWEEN '2022-11-30' AND '2022-12-31' --targeting Dec 2022 file
)
, RunoutCCLFBenesResult AS (	
	SELECT files.aco_cms_id,
		benes.mbi
	FROM acos ac
	INNER JOIN cclf_files files ON ac.cms_id = files.aco_cms_id
	INNER JOIN cclf_beneficiaries benes ON benes.file_id = files.id
	WHERE ac.termination_details IS NULL
	AND (ac.cms_id LIKE 'X%' OR ac.cms_id LIKE 'Y%' OR ac.cms_id LIKE 'Z%')  --test data 
	--AND (ac.cms_id LIKE 'K%' OR ac.cms_id LIKE 'C%') -- KCC model entity type (KCE and KCF)
	AND files.name LIKE '%221231.SYNTHETICRUNOUTFILE' --targeting synthetic runout file created for 2022
)
(   SELECT *, TRUE AS dec_file_only, FALSE AS runout_file_only FROM DecCCLFBenesResult
    EXCEPT
    SELECT *, TRUE AS dec_file_only, FALSE AS runout_file_only FROM RunoutCCLFBenesResult)  
UNION ALL
(   SELECT *, FALSE AS dec_file_only, TRUE AS runout_file_only FROM RunoutCCLFBenesResult
    EXCEPT
    SELECT *, FALSE AS dec_file_only, TRUE AS runout_file_only FROM DecCCLFBenesResult) 

--Verification script #2: confirm the count of mbis is the same per entity between generated synthetic runout and dec file (based off dec file).
--Total counts per entity for Dec should be confirmed against total counts per entity on synthetic runout file.

WITH DecCCLFMBICountResult AS (
SELECT files.aco_cms_id,
		count(benes.mbi) AS mbi_count
FROM acos ac
INNER JOIN cclf_files files ON ac.cms_id = files.aco_cms_id
INNER JOIN cclf_beneficiaries benes ON benes.file_id = files.id
WHERE ac.termination_details IS NULL
	AND (ac.cms_id LIKE 'X%' OR ac.cms_id LIKE 'Y%' OR ac.cms_id LIKE 'Z%')  --targeting test data 
	--AND (ac.cms_id LIKE 'K%' OR ac.cms_id LIKE 'C%') -- KCC model entity type (KCE and KCF)
	AND files.created_at BETWEEN '2022-11-30' AND '2022-12-31' --targeting Dec 2022 file
	GROUP BY files.aco_cms_id
) 
, SynRunoutMBICountResult AS (
SELECT files.aco_cms_id,
		count(benes.mbi) AS mbi_count
FROM acos ac
INNER JOIN cclf_files files ON ac.cms_id = files.aco_cms_id
INNER JOIN cclf_beneficiaries benes ON benes.file_id = files.id
WHERE ac.termination_details IS NULL
	AND (ac.cms_id LIKE 'X%' OR ac.cms_id LIKE 'Y%' OR ac.cms_id LIKE 'Z%')  --targeting test data 
	--AND (ac.cms_id LIKE 'K%' OR ac.cms_id LIKE 'C%') -- KCC model entity type (KCE and KCF)
	AND files.name LIKE '%221231.SYNTHETICRUNOUTFILE' --targeting synthetic runout file created for 2022
	GROUP BY files.aco_cms_id
)
Select DecCCLFMBICountResult.aco_cms_id,
DecCCLFMBICountResult.mbi_count AS dec_mbi_count,
SynRunoutMBICountResult.mbi_count AS runout_mbi_count,
CASE WHEN DecCCLFMBICountResult.mbi_count = SynRunoutMBICountResult.mbi_count THEN NULL ELSE 'ERR: Difference in count!' END AS Err
FROM DecCCLFMBICountResult
LEFT JOIN SynRunoutMBICountResult on DecCCLFMBICountResult.aco_cms_id = SynRunoutMBICountResult.aco_cms_id

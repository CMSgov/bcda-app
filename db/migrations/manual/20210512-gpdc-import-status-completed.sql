-- SQL script to re-enable DC models' ability to receive data. 
BEGIN;
UPDATE cclf_files SET import_status = 'Completed' WHERE aco_cms_id LIKE 'D%';
COMMIT;
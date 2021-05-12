-- SQL script to manually prevent DC models from receiving export data. 
BEGIN;
-- only files with an import status set to 'Completed' are able to be retrieved
UPDATE cclf_files SET import_status = 'Ignore_Manual' WHERE aco_cms_id LIKE 'D%';
COMMIT;
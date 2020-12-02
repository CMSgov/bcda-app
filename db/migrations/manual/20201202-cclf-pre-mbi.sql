-- Script will remove CCLF data that existed before MBIs were reliable
BEGIN;

DELETE FROM cclf_beneficiaries WHERE file_id IN (SELECT id FROM cclf_files WHERE created_at < '2020-01-01');
DELETE FROM cclf_files WHERE created_at < '2020-01-01';

ROLLBACK;
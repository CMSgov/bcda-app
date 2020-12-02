-- Script will remove suppression entries that contained a HICN value
BEGIN;

-- Since we only can detect the suppression files to delete by looking at the specific suppresion entry.
-- We need to store what files we need to remove before we clean the suppresion entries
CREATE TEMPORARY TABLE suppression_files_to_remove (file_id integer);
INSERT INTO suppression_files_to_remove SELECT DISTINCT(file_id) from suppressions WHERE LENGTH(hicn) > 0;
DELETE FROM suppressions WHERE LENGTH(hicn) > 0;
DELETE FROM suppression_files WHERE id IN (SELECT file_id FROM suppression_files_to_remove);

ROLLBACK;
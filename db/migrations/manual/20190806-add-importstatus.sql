UPDATE cclf_files set import_status = 'Completed' where created_at < now();
UPDATE suppression_files set import_status = 'Completed' where created_at < now();
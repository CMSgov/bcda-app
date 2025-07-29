SELECT pg_terminate_backend(pid)
FROM pg_stat_activity
WHERE datname = 'bcda_queue' AND pid <> pg_backend_pid();
DROP DATABASE IF EXISTS bcda_queue;

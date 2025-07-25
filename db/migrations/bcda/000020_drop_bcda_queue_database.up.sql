-- Drop bcda_queue database if it exists
-- This migration removes the separate bcda_queue database as the application
-- has been consolidated to use a single database (bcda)

DO $$
BEGIN
    -- Drop the database if it exists
    -- Note: We need to terminate all connections first
    IF EXISTS (SELECT 1 FROM pg_database WHERE datname = 'bcda_queue') THEN
        -- Terminate all connections to bcda_queue database
        PERFORM pg_terminate_backend(pid)
        FROM pg_stat_activity
        WHERE datname = 'bcda_queue' AND pid <> pg_backend_pid();

        -- Drop the database
        DROP DATABASE IF EXISTS bcda_queue;
    END IF;
END $$;

-- Recreate bcda_queue database (down migration)
-- This recreates the bcda_queue database with the original schema

CREATE DATABASE bcda_queue;

-- Connect to the new database and create the original schema
\c bcda_queue;

-- Create the que_jobs table and related objects
CREATE TABLE public.que_jobs (
    priority smallint DEFAULT 100 NOT NULL,
    run_at timestamp with time zone DEFAULT now() NOT NULL,
    job_id bigint NOT NULL,
    job_class text NOT NULL,
    args json DEFAULT '[]'::json NOT NULL,
    error_count integer DEFAULT 0 NOT NULL,
    last_error text,
    queue text DEFAULT ''::text NOT NULL
);

COMMENT ON TABLE public.que_jobs IS '3';

CREATE SEQUENCE public.que_jobs_job_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE public.que_jobs_job_id_seq OWNED BY public.que_jobs.job_id;

ALTER TABLE ONLY public.que_jobs ALTER COLUMN job_id SET DEFAULT nextval('public.que_jobs_job_id_seq'::regclass);

ALTER TABLE ONLY public.que_jobs
    ADD CONSTRAINT que_jobs_pkey PRIMARY KEY (queue, priority, run_at, job_id);

-- Create the schema_migrations_bcda_queue table
CREATE TABLE public.schema_migrations_bcda_queue (
    version bigint NOT NULL,
    dirty boolean NOT NULL
);

ALTER TABLE ONLY public.schema_migrations_bcda_queue
    ADD CONSTRAINT schema_migrations_bcda_queue_pkey PRIMARY KEY (version);

-- Insert the migration version
INSERT INTO public.schema_migrations_bcda_queue (version, dirty) VALUES (1, false);

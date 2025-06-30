-- public.schema_migrations_ssas definition

-- Drop table

-- DROP TABLE public.schema_migrations_ssas;

-- This sets up SSAS schema_migrations.  This table is responsible for keeping track of bcda-ssas-app 
-- migrations.  See: https://github.com/golang-migrate/migrate
-- Overall both of these scripts should only need to run once in each env.

CREATE TABLE public.schema_migrations_ssas (
	"version" int8 NOT NULL,
	dirty bool NOT NULL,
	CONSTRAINT schema_migrations_ssas_pkey PRIMARY KEY (version)
);

-- 9 is the current migration level at the time of this PR, see:
-- https://github.com/CMSgov/bcda-ssas-app/tree/main/db/migrations (000009_add_uuid_to_encryption_key.up.sql)

INSERT INTO public.schema_migrations_ssas (version) VALUES (9);
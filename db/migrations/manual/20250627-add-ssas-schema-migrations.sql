-- public.schema_migrations_ssas definition

-- Drop table

-- DROP TABLE public.schema_migrations_ssas;

CREATE TABLE public.schema_migrations_ssas (
	"version" int8 NOT NULL,
	dirty bool NOT NULL,
	CONSTRAINT schema_migrations_ssas_pkey PRIMARY KEY (version)
);

INSERT INTO public.schema_migrations_ssas (version) VALUES (9);
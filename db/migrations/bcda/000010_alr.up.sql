BEGIN;

CREATE TABLE IF NOT EXISTS public.alr (
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    id bigint NOT NULL,
    -- metakey is the foreign key referencing id in alr_meta
    metakey integer NOT NULL,
    mbi character(10) NOT NULL,
    hic character varying(10),
    firstname character(50),
    lastname character(50),
    sex character varying(10),
    dob timestamp,
    dod timestamp,
    timestp timestamp with time zone,
    keyvalue bytea
);

CREATE TABLE IF NOT EXISTS public.alr_meta (
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    id bigint NOT NULL,
    aco character varying(5),
    timestp timestamp with time zone,
    UNIQUE (id, aco, timestp)
);

CREATE INDEX IF NOT EXISTS idx_metaid_timestamp ON public.alr_meta (aco, timestp);

ALTER TABLE ONLY public.alr_meta
    ADD CONSTRAINT primary_key_alr_meta PRIMARY KEY (id);
ALTER TABLE ONLY public.alr
    ADD CONSTRAINT foreign_key_alr FOREIGN KEY (metakey) REFERENCES public.alr_meta(id) ON UPDATE RESTRICT ON DELETE RESTRICT;

CREATE SEQUENCE IF NOT EXISTS public.metaid_seq START WITH 1 INCREMENT BY 1 CACHE 1 OWNED BY public.alr_meta.id;
ALTER TABLE ONLY public.alr_meta ALTER COLUMN id SET DEFAULT nextval('public.metaid_seq');

COMMIT;
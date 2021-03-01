BEGIN;

-- Create ALR table
CREATE TABLE IF NOT EXISTS public.alr (
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    id bigint NOT NULL,
    -- metakey is the foreign key referencing id in alr_meta
    metakey bigint NOT NULL,
    mbi character(11) NOT NULL,
    hic character(12),
    firstname character varying(30),
    lastname character varying(40),
    sex character(1),
    dob timestamp,
    dod timestamp,
    keyvalue bytea
);

-- Create ALR Metadata table
CREATE TABLE IF NOT EXISTS public.alr_meta (
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    id bigint NOT NULL,
    aco character varying(5),
    timestp timestamp with time zone,
    UNIQUE (id, aco, timestp)
);

-- Index the ALR Metadata table by two fields
CREATE INDEX IF NOT EXISTS idx_metaid_timestamp ON public.alr_meta USING btree (aco, timestp);

-- Set PK and FK
ALTER TABLE ONLY public.alr_meta
    ADD CONSTRAINT primary_key_alr_meta PRIMARY KEY (id);
ALTER TABLE ONLY public.alr
    ADD CONSTRAINT foreign_key_alr FOREIGN KEY (metakey) REFERENCES public.alr_meta(id) ON UPDATE RESTRICT ON DELETE RESTRICT;

-- Set Incrementing to both IDs
CREATE SEQUENCE IF NOT EXISTS public.alr_meta_id_seq START WITH 1 INCREMENT BY 1 CACHE 1 OWNED BY public.alr_meta.id;
ALTER TABLE ONLY public.alr_meta ALTER COLUMN id SET DEFAULT nextval('public.alr_meta_id_seq');
CREATE SEQUENCE IF NOT EXISTS public.alr_id_seq START WITH 1 INCREMENT BY 1 CACHE 1 OWNED BY public.alr.id;
ALTER TABLE ONLY public.alr ALTER COLUMN id SET DEFAULT nextval('public.alr_id_seq');

-- Function to call when a row is updated
CREATE OR REPLACE FUNCTION trigger_set_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger trigger_set_timestamp function on updated_at column for alr & alr_meta
CREATE TRIGGER set_timestamp
BEFORE UPDATE ON public.alr
FOR EACH ROW
EXECUTE PROCEDURE trigger_set_timestamp();

CREATE TRIGGER set_timestamp
BEFORE UPDATE ON public.alr_meta
FOR EACH ROW
EXECUTE PROCEDURE trigger_set_timestamp();

COMMIT;
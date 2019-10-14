BEGIN;
CREATE TABLE public.blacklist_entries (
                                          id integer NOT NULL,
                                          created_at timestamp with time zone,
                                          updated_at timestamp with time zone,
                                          deleted_at timestamp with time zone,
                                          key text NOT NULL,
                                          entry_date bigint NOT NULL,
                                          cache_expiration bigint NOT NULL
);
ALTER TABLE public.blacklist_entries OWNER TO postgres;
CREATE SEQUENCE public.blacklist_entries_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER TABLE public.blacklist_entries_id_seq OWNER TO postgres;
ALTER SEQUENCE public.blacklist_entries_id_seq OWNED BY public.blacklist_entries.id;


CREATE TABLE public.encryption_keys (
                                        id integer NOT NULL,
                                        created_at timestamp with time zone,
                                        updated_at timestamp with time zone,
                                        deleted_at timestamp with time zone,
                                        body text,
                                        system_id integer
);
ALTER TABLE public.encryption_keys OWNER TO postgres;
CREATE SEQUENCE public.encryption_keys_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER TABLE public.encryption_keys_id_seq OWNER TO postgres;
ALTER SEQUENCE public.encryption_keys_id_seq OWNED BY public.encryption_keys.id;

CREATE TABLE public.groups (
                               id integer NOT NULL,
                               created_at timestamp with time zone,
                               updated_at timestamp with time zone,
                               deleted_at timestamp with time zone,
                               group_id text NOT NULL,
                               x_data text,
                               data jsonb
);
ALTER TABLE public.groups OWNER TO postgres;
CREATE SEQUENCE public.groups_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER TABLE public.groups_id_seq OWNER TO postgres;
ALTER SEQUENCE public.groups_id_seq OWNED BY public.groups.id;

CREATE TABLE public.secrets (
                                id integer NOT NULL,
                                created_at timestamp with time zone,
                                updated_at timestamp with time zone,
                                deleted_at timestamp with time zone,
                                hash text,
                                system_id integer
);
ALTER TABLE public.secrets OWNER TO postgres;
CREATE SEQUENCE public.secrets_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER TABLE public.secrets_id_seq OWNER TO postgres;
ALTER SEQUENCE public.secrets_id_seq OWNED BY public.secrets.id;

CREATE TABLE public.systems (
                                id integer NOT NULL,
                                created_at timestamp with time zone,
                                updated_at timestamp with time zone,
                                deleted_at timestamp with time zone,
                                group_id text,
                                client_id text,
                                software_id text,
                                client_name text,
                                api_scope text
);
ALTER TABLE public.systems OWNER TO postgres;
CREATE SEQUENCE public.systems_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER TABLE public.systems_id_seq OWNER TO postgres;
ALTER SEQUENCE public.systems_id_seq OWNED BY public.systems.id;

ALTER TABLE ONLY public.blacklist_entries ALTER COLUMN id SET DEFAULT nextval('public.blacklist_entries_id_seq'::regclass);
ALTER TABLE ONLY public.encryption_keys ALTER COLUMN id SET DEFAULT nextval('public.encryption_keys_id_seq'::regclass);
ALTER TABLE ONLY public.groups ALTER COLUMN id SET DEFAULT nextval('public.groups_id_seq'::regclass);
ALTER TABLE ONLY public.secrets ALTER COLUMN id SET DEFAULT nextval('public.secrets_id_seq'::regclass);
ALTER TABLE ONLY public.systems ALTER COLUMN id SET DEFAULT nextval('public.systems_id_seq'::regclass);

ALTER TABLE ONLY public.blacklist_entries
    ADD CONSTRAINT blacklist_entries_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.encryption_keys
    ADD CONSTRAINT encryption_keys_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.groups
    ADD CONSTRAINT groups_group_id_key UNIQUE (group_id);
ALTER TABLE ONLY public.groups
    ADD CONSTRAINT groups_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.secrets
    ADD CONSTRAINT secrets_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.systems
    ADD CONSTRAINT systems_pkey PRIMARY KEY (id);

CREATE INDEX idx_blacklist_entries_deleted_at ON public.blacklist_entries USING btree (deleted_at);
CREATE INDEX idx_encryption_keys_deleted_at ON public.encryption_keys USING btree (deleted_at);
CREATE INDEX idx_groups_deleted_at ON public.groups USING btree (deleted_at);
CREATE INDEX idx_secrets_deleted_at ON public.secrets USING btree (deleted_at);
CREATE INDEX idx_systems_deleted_at ON public.systems USING btree (deleted_at);

ALTER TABLE ONLY public.encryption_keys
    ADD CONSTRAINT encryption_keys_system_id_systems_id_foreign FOREIGN KEY (system_id) REFERENCES public.systems(id) ON UPDATE RESTRICT ON DELETE RESTRICT;
ALTER TABLE ONLY public.secrets
    ADD CONSTRAINT secrets_system_id_systems_id_foreign FOREIGN KEY (system_id) REFERENCES public.systems(id) ON UPDATE RESTRICT ON DELETE RESTRICT;
ALTER TABLE ONLY public.systems
    ADD CONSTRAINT systems_group_id_groups_group_id_foreign FOREIGN KEY (group_id) REFERENCES public.groups(group_id) ON UPDATE RESTRICT ON DELETE RESTRICT;
COMMIT;
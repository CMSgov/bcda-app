--
-- bcda schema as of 2020-10-01
--

BEGIN;

--
-- PostgreSQL database dump
--

--
-- Name: acos; Type: TABLE; Schema: public; 
--

CREATE TABLE IF NOT EXISTS public.acos (
    uuid uuid NOT NULL,
    name text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    id integer NOT NULL,
    deleted_at timestamp with time zone,
    client_id text,
    cms_id character varying(5),
    alpha_secret text,
    public_key text,
    group_id text,
    system_id text
);


--
-- Name: acos_id_seq; Type: SEQUENCE; Schema: public; 
--

CREATE SEQUENCE IF NOT EXISTS public.acos_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: acos_id_seq; Type: SEQUENCE OWNED BY; Schema: public; 
--

ALTER SEQUENCE public.acos_id_seq OWNED BY public.acos.id;

--
-- Name: cclf_beneficiaries; Type: TABLE; Schema: public; 
--

CREATE TABLE IF NOT EXISTS public.cclf_beneficiaries (
    id integer NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    file_id integer NOT NULL,
    hicn character varying(11) NOT NULL,
    mbi character(11) NOT NULL,
    beneficiary_id integer,
    blue_button_id text
);

--
-- Name: cclf_beneficiaries_id_seq; Type: SEQUENCE; Schema: public; 
--

CREATE SEQUENCE IF NOT EXISTS public.cclf_beneficiaries_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: cclf_beneficiaries_id_seq; Type: SEQUENCE OWNED BY; Schema: public; 
--

ALTER SEQUENCE public.cclf_beneficiaries_id_seq OWNED BY public.cclf_beneficiaries.id;


--
-- Name: cclf_beneficiary_xrefs; Type: TABLE; Schema: public; 
--

CREATE TABLE IF NOT EXISTS public.cclf_beneficiary_xrefs (
    id integer NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    file_id integer NOT NULL,
    xref_indicator text,
    current_num text,
    prev_num text,
    prevs_efct_dt text,
    prevs_obslt_dt text
);


--
-- Name: cclf_beneficiary_xrefs_id_seq; Type: SEQUENCE; Schema: public; 
--

CREATE SEQUENCE IF NOT EXISTS public.cclf_beneficiary_xrefs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: cclf_beneficiary_xrefs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; 
--

ALTER SEQUENCE public.cclf_beneficiary_xrefs_id_seq OWNED BY public.cclf_beneficiary_xrefs.id;


--
-- Name: cclf_files; Type: TABLE; Schema: public; 
--

CREATE TABLE IF NOT EXISTS public.cclf_files (
    id integer NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    cclf_num integer NOT NULL,
    name text NOT NULL,
    aco_cms_id character varying(5),
    "timestamp" timestamp with time zone NOT NULL,
    performance_year integer NOT NULL,
    import_status text
);


--
-- Name: cclf_files_id_seq; Type: SEQUENCE; Schema: public; 
--

CREATE SEQUENCE IF NOT EXISTS public.cclf_files_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: cclf_files_id_seq; Type: SEQUENCE OWNED BY; Schema: public; 
--

ALTER SEQUENCE public.cclf_files_id_seq OWNED BY public.cclf_files.id;

--
-- Name: job_keys; Type: TABLE; Schema: public; 
--

CREATE TABLE IF NOT EXISTS public.job_keys (
    id integer NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    job_id integer NOT NULL,
    file_name character(127),
    resource_type text
);


--
-- Name: job_keys_id_seq; Type: SEQUENCE; Schema: public; 
--

CREATE SEQUENCE IF NOT EXISTS public.job_keys_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: job_keys_id_seq; Type: SEQUENCE OWNED BY; Schema: public; 
--

ALTER SEQUENCE public.job_keys_id_seq OWNED BY public.job_keys.id;


--
-- Name: job_keys_job_id_seq; Type: SEQUENCE; Schema: public; 
--

CREATE SEQUENCE IF NOT EXISTS public.job_keys_job_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: job_keys_job_id_seq; Type: SEQUENCE OWNED BY; Schema: public; 
--

ALTER SEQUENCE public.job_keys_job_id_seq OWNED BY public.job_keys.job_id;


--
-- Name: jobs; Type: TABLE; Schema: public; 
--

CREATE TABLE IF NOT EXISTS public.jobs (
    id integer NOT NULL,
    aco_id uuid NOT NULL,
    request_url text NOT NULL,
    status text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    job_count integer,
    completed_job_count integer,
    transaction_time timestamp with time zone,
    priority integer
);


--
-- Name: jobs_id_seq; Type: SEQUENCE; Schema: public; 
--

CREATE SEQUENCE IF NOT EXISTS public.jobs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: jobs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; 
--

ALTER SEQUENCE public.jobs_id_seq OWNED BY public.jobs.id;

--
-- Name: suppression_files; Type: TABLE; Schema: public; 
--

CREATE TABLE IF NOT EXISTS public.suppression_files (
    id integer NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    name text NOT NULL,
    "timestamp" timestamp with time zone NOT NULL,
    import_status text
);


--
-- Name: suppression_files_id_seq; Type: SEQUENCE; Schema: public; 
--

CREATE SEQUENCE IF NOT EXISTS public.suppression_files_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: suppression_files_id_seq; Type: SEQUENCE OWNED BY; Schema: public; 
--

ALTER SEQUENCE public.suppression_files_id_seq OWNED BY public.suppression_files.id;


--
-- Name: suppressions; Type: TABLE; Schema: public; 
--

CREATE TABLE IF NOT EXISTS public.suppressions (
    id integer NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    hicn character varying(11) NOT NULL,
    source_code character varying(5),
    effective_dt timestamp with time zone,
    pref_indicator character(1),
    samhsa_source_code character varying(5),
    samhsa_effective_dt timestamp with time zone,
    samhsa_pref_indicator character(1),
    aco_cms_id character(5),
    beneficiary_link_key integer,
    effective_date timestamp with time zone,
    preference_indicator character(1),
    samhsa_effective_date timestamp with time zone,
    samhsa_preference_indicator character(1),
    file_id integer NOT NULL,
    blue_button_id text,
    mbi character varying(11)
);


--
-- Name: suppressions_id_seq; Type: SEQUENCE; Schema: public; 
--

CREATE SEQUENCE IF NOT EXISTS public.suppressions_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: suppressions_id_seq; Type: SEQUENCE OWNED BY; Schema: public; 
--

ALTER SEQUENCE public.suppressions_id_seq OWNED BY public.suppressions.id;

--
-- Name: acos id; Type: DEFAULT; Schema: public; 
--

ALTER TABLE ONLY public.acos ALTER COLUMN id SET DEFAULT nextval('public.acos_id_seq'::regclass);

--
-- Name: cclf_beneficiaries id; Type: DEFAULT; Schema: public; 
--

ALTER TABLE ONLY public.cclf_beneficiaries ALTER COLUMN id SET DEFAULT nextval('public.cclf_beneficiaries_id_seq'::regclass);


--
-- Name: cclf_beneficiary_xrefs id; Type: DEFAULT; Schema: public; 
--

ALTER TABLE ONLY public.cclf_beneficiary_xrefs ALTER COLUMN id SET DEFAULT nextval('public.cclf_beneficiary_xrefs_id_seq'::regclass);


--
-- Name: cclf_files id; Type: DEFAULT; Schema: public; 
--

ALTER TABLE ONLY public.cclf_files ALTER COLUMN id SET DEFAULT nextval('public.cclf_files_id_seq'::regclass);

--
-- Name: job_keys id; Type: DEFAULT; Schema: public; 
--

ALTER TABLE ONLY public.job_keys ALTER COLUMN id SET DEFAULT nextval('public.job_keys_id_seq'::regclass);


--
-- Name: job_keys job_id; Type: DEFAULT; Schema: public; 
--

ALTER TABLE ONLY public.job_keys ALTER COLUMN job_id SET DEFAULT nextval('public.job_keys_job_id_seq'::regclass);


--
-- Name: jobs id; Type: DEFAULT; Schema: public; 
--

ALTER TABLE ONLY public.jobs ALTER COLUMN id SET DEFAULT nextval('public.jobs_id_seq'::regclass);


--
-- Name: suppression_files id; Type: DEFAULT; Schema: public; 
--

ALTER TABLE ONLY public.suppression_files ALTER COLUMN id SET DEFAULT nextval('public.suppression_files_id_seq'::regclass);


--
-- Name: suppressions id; Type: DEFAULT; Schema: public; 
--

ALTER TABLE ONLY public.suppressions ALTER COLUMN id SET DEFAULT nextval('public.suppressions_id_seq'::regclass);

--
-- Name: acos acos_cms_id_key; Type: CONSTRAINT; Schema: public; 
--

ALTER TABLE ONLY public.acos
    ADD CONSTRAINT acos_cms_id_key UNIQUE (cms_id);


--
-- Name: acos acos_pkey; Type: CONSTRAINT; Schema: public; 
--

ALTER TABLE ONLY public.acos
    ADD CONSTRAINT acos_pkey PRIMARY KEY (uuid);

--
-- Name: cclf_beneficiaries cclf_beneficiaries_pkey; Type: CONSTRAINT; Schema: public; 
--

ALTER TABLE ONLY public.cclf_beneficiaries
    ADD CONSTRAINT cclf_beneficiaries_pkey PRIMARY KEY (id);


--
-- Name: cclf_beneficiary_xrefs cclf_beneficiary_xrefs_pkey; Type: CONSTRAINT; Schema: public; 
--

ALTER TABLE ONLY public.cclf_beneficiary_xrefs
    ADD CONSTRAINT cclf_beneficiary_xrefs_pkey PRIMARY KEY (id);


--
-- Name: cclf_files cclf_files_pkey; Type: CONSTRAINT; Schema: public; 
--

ALTER TABLE ONLY public.cclf_files
    ADD CONSTRAINT cclf_files_pkey PRIMARY KEY (id);

--
-- Name: job_keys job_keys_pkey; Type: CONSTRAINT; Schema: public; 
--

ALTER TABLE ONLY public.job_keys
    ADD CONSTRAINT job_keys_pkey PRIMARY KEY (id, job_id);


--
-- Name: jobs jobs_pkey; Type: CONSTRAINT; Schema: public; 
--

ALTER TABLE ONLY public.jobs
    ADD CONSTRAINT jobs_pkey PRIMARY KEY (id);

--
-- Name: suppression_files suppression_files_name_key; Type: CONSTRAINT; Schema: public; 
--

ALTER TABLE ONLY public.suppression_files
    ADD CONSTRAINT suppression_files_name_key UNIQUE (name);


--
-- Name: suppression_files suppression_files_pkey; Type: CONSTRAINT; Schema: public; 
--

ALTER TABLE ONLY public.suppression_files
    ADD CONSTRAINT suppression_files_pkey PRIMARY KEY (id);


--
-- Name: suppressions suppressions_pkey; Type: CONSTRAINT; Schema: public; 
--

ALTER TABLE ONLY public.suppressions
    ADD CONSTRAINT suppressions_pkey PRIMARY KEY (id);

--
-- Name: idx_acos_deleted_at; Type: INDEX; Schema: public; 
--

CREATE INDEX IF NOT EXISTS idx_acos_deleted_at ON public.acos USING btree (deleted_at);

--
-- Name: idx_cclf_beneficiaries_bb_id; Type: INDEX; Schema: public; 
--

CREATE INDEX IF NOT EXISTS idx_cclf_beneficiaries_bb_id ON public.cclf_beneficiaries USING btree (blue_button_id);


--
-- Name: idx_cclf_beneficiaries_deleted_at; Type: INDEX; Schema: public; 
--

CREATE INDEX IF NOT EXISTS idx_cclf_beneficiaries_deleted_at ON public.cclf_beneficiaries USING btree (deleted_at);


--
-- Name: idx_cclf_beneficiaries_file_id; Type: INDEX; Schema: public; 
--

CREATE INDEX IF NOT EXISTS idx_cclf_beneficiaries_file_id ON public.cclf_beneficiaries USING btree (file_id);


--
-- Name: idx_cclf_beneficiaries_hicn; Type: INDEX; Schema: public; 
--

CREATE INDEX IF NOT EXISTS idx_cclf_beneficiaries_hicn ON public.cclf_beneficiaries USING btree (hicn);


--
-- Name: idx_cclf_beneficiaries_mbi; Type: INDEX; Schema: public; 
--

CREATE INDEX IF NOT EXISTS idx_cclf_beneficiaries_mbi ON public.cclf_beneficiaries USING btree (mbi);


--
-- Name: idx_cclf_beneficiary_xrefs_deleted_at; Type: INDEX; Schema: public; 
--

CREATE INDEX IF NOT EXISTS idx_cclf_beneficiary_xrefs_deleted_at ON public.cclf_beneficiary_xrefs USING btree (deleted_at);


--
-- Name: idx_cclf_files_deleted_at; Type: INDEX; Schema: public; 
--

CREATE INDEX IF NOT EXISTS idx_cclf_files_deleted_at ON public.cclf_files USING btree (deleted_at);

--
-- Name: idx_cclf_files_name_aco_cms_id_key; Type: INDEX; Schema: public; 
--

CREATE UNIQUE INDEX idx_cclf_files_name_aco_cms_id_key ON public.cclf_files USING btree (name, aco_cms_id);


--
-- Name: idx_job_keys_deleted_at; Type: INDEX; Schema: public; 
--

CREATE INDEX IF NOT EXISTS idx_job_keys_deleted_at ON public.job_keys USING btree (deleted_at);


--
-- Name: idx_jobs_deleted_at; Type: INDEX; Schema: public; 
--

CREATE INDEX IF NOT EXISTS idx_jobs_deleted_at ON public.jobs USING btree (deleted_at);

--
-- Name: idx_suppression_bb_id; Type: INDEX; Schema: public; 
--

CREATE INDEX IF NOT EXISTS idx_suppression_bb_id ON public.suppressions USING btree (blue_button_id);


--
-- Name: idx_suppression_files_deleted_at; Type: INDEX; Schema: public; 
--

CREATE INDEX IF NOT EXISTS idx_suppression_files_deleted_at ON public.suppression_files USING btree (deleted_at);


--
-- Name: idx_suppression_mbi; Type: INDEX; Schema: public; 
--

CREATE INDEX IF NOT EXISTS idx_suppression_mbi ON public.suppressions USING btree (mbi);


--
-- Name: idx_suppressions_deleted_at; Type: INDEX; Schema: public; 
--

CREATE INDEX IF NOT EXISTS idx_suppressions_deleted_at ON public.suppressions USING btree (deleted_at);

--
-- Name: cclf_beneficiaries cclf_beneficiaries_file_id_cclf_files_id_foreign; Type: FK CONSTRAINT; Schema: public; 
--

ALTER TABLE ONLY public.cclf_beneficiaries
    ADD CONSTRAINT cclf_beneficiaries_file_id_cclf_files_id_foreign FOREIGN KEY (file_id) REFERENCES public.cclf_files(id) ON UPDATE RESTRICT ON DELETE RESTRICT;


--
-- Name: jobs jobs_aco_id_fkey; Type: FK CONSTRAINT; Schema: public; 
--

ALTER TABLE ONLY public.jobs
    ADD CONSTRAINT jobs_aco_id_fkey FOREIGN KEY (aco_id) REFERENCES public.acos(uuid);

--
-- PostgreSQL database dump complete
--

COMMIT;
-- Name: cclf_beneficiary_xrefs; Type: TABLE; Schema: public; 
BEGIN;
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
COMMIT;

-- Name: cclf_beneficiary_xrefs_id_seq; Type: SEQUENCE; Schema: public; 
BEGIN;
CREATE SEQUENCE IF NOT EXISTS public.cclf_beneficiary_xrefs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
COMMIT;

-- Name: cclf_beneficiary_xrefs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; 
BEGIN;
ALTER SEQUENCE public.cclf_beneficiary_xrefs_id_seq OWNED BY public.cclf_beneficiary_xrefs.id;
COMMIT;

-- Name: cclf_beneficiary_xrefs id; Type: DEFAULT; Schema: public; 
BEGIN;
ALTER TABLE ONLY public.cclf_beneficiary_xrefs ALTER COLUMN id SET DEFAULT nextval('public.cclf_beneficiary_xrefs_id_seq'::regclass);
COMMIT;
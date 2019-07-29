create table acos (
  uuid uuid not null primary key,
  cms_id char(5) unique null,
  name text not null,
  created_at timestamp with time zone not null default now(),
  updated_at timestamp with time zone not null default now(),
  client_id text null,
  alpha_secret text null,
  public_key text null
);

create table users (
  uuid uuid not null primary key,
  name text not null,
  email text not null unique,
  aco_id uuid not null,
  created_at timestamp with time zone not null default now(),
  updated_at timestamp with time zone not null default now(),
  foreign key (aco_id) references acos (uuid)
);

create table jobs (
  id serial not null primary key,
  aco_id uuid not null references acos,
  user_id uuid not null references users,
  request_url text not null,
  status text not null,
  created_at timestamp with time zone not null default now()
);

create table tokens (
  uuid uuid not null primary key,
  value text not null,
  active boolean not null default false
);

create table cclf_files (
    id serial primary key,
    cclf_num integer not null,
    name varchar not null,
    aco_cms_id char(5),
    "timestamp" timestamp with time zone not null,
    performance_year integer not null,
    created_at timestamp with time zone not null default now(),
    updated_at timestamp with time zone not null default now(),
    deleted_at timestamp with time zone
);

create table cclf_beneficiaries (
    id serial primary key,
    file_id integer not null,
    hicn varchar(11) not null,
    mbi char(11) not null,
    blue_button_id text,
    created_at timestamp with time zone not null default now(),
    updated_at timestamp with time zone not null default now(),
    deleted_at timestamp with time zone
);

create table cclf_beneficiary_xrefs (
    id serial primary key,
    file_id integer not null,
    xref_indicator varchar,
    current_num varchar,
    prev_num varchar,
    prevs_efct_dt varchar,
    prevs_obslt_dt varchar,
    created_at timestamp with time zone not null default now(),
    updated_at timestamp with time zone not null default now(),
    deleted_at timestamp with time zone
);

create table suppressions (
    id serial primary key,
    hicn varchar(11) not null,
    source_code varchar,
    effective_date timestamp with time zone,
    preference_indicator char(1),
    samhsa_source_code varchar,
    samhsa_effective_date timestamp with time zone,
    samhsa_preference_indicator char(1),
    aco_cms_id char(5),
    beneficiary_link_key integer,
    created_at timestamp with time zone not null default now(),
    updated_at timestamp with time zone not null default now(),
    deleted_at timestamp with time zone
);

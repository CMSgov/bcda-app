create table acos (
      id serial not null primary key,
      name text not null,
      encrypted_password text not null,
      created_at timestamp with time zone not null default now(),
      updated_at timestamp with time zone not null default now()
);

create table jobs (
      id serial not null primary key,
      aco_id integer not null references acos,
      location text not null,
      status text not null,
      created_at timestamp with time zone not null default now()
);

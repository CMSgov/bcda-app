create table acos (
  uuid uuid not null primary key,
  name text not null,
  created_at timestamp with time zone not null default now(),
  updated_at timestamp with time zone not null default now()
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
  location text not null,
  status text not null,
  created_at timestamp with time zone not null default now()
);

create table tokens (
  uuid uuid not null primary key,
  user_id uuid not null references users,
  value text not null,
  active boolean not null default false
);

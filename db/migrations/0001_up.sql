-- +goose Up
create table tenants (
id text primary key,
name text not null,
api_key_hash text not null,
created_at timestamptz default now()
);


create table job_types (
id bigserial primary key,
tenant_id text not null references tenants(id) on delete cascade,
type text not null,
max_attempts int not null default 10,
backoff_policy text not null default 'exponential',
visibility_timeout_sec int not null default 60,
rate_limit_key text,
rate_limit_qps int,
schema jsonb,
unique (tenant_id, type)
);


create type job_status as enum ('queued','leased','succeeded','failed_temp','failed_perm','dead_lettered');


create table jobs (
id uuid primary key,
tenant_id text not null references tenants(id) on delete cascade,
type text not null,
payload jsonb not null,
priority int not null default 100,
run_at timestamptz not null default now(),
dedupe_key text,
dedupe_ttl_sec int,
attempt int not null default 0,
max_attempts int not null,
backoff_policy text not null,
visibility_timeout_sec int not null,
status job_status not null default 'queued',
leased_by text,
lease_expires_at timestamptz,
error text,
created_at timestamptz default now(),
updated_at timestamptz default now()
);
create index jobs_tenant_status_runat on jobs(tenant_id, status, run_at);
create index jobs_dedupe on jobs(dedupe_key);


create table job_events (
id bigserial primary key,
job_id uuid not null references jobs(id) on delete cascade,
tenant_id text not null references tenants(id) on delete cascade,
event text not null,
metadata jsonb,
created_at timestamptz default now()
);


create table schedules (
id uuid primary key,
tenant_id text not null references tenants(id) on delete cascade,
type text not null,
cron_expr text,
interval_sec int,
next_run_at timestamptz not null,
enabled bool not null default true,
payload jsonb,
config jsonb
);


create table dead_letters (
job_id uuid primary key references jobs(id) on delete cascade,
tenant_id text not null references tenants(id) on delete cascade,
reason text,
parked_at timestamptz default now()
);
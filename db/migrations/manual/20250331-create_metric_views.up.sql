-- These are views designed to give metrics and insights into BCDA usage without exposing actual DB structures/info.
-- The goal is to be very explicit with what is being shown to make sure NO PHI/PII is exposed.
-- These views will be used for various metrics and analytics so will be slightly more accessible than other tables.

-- Active ACOs View table
-- Select query from https://github.com/CMSgov/bcda-ops/blob/main/terraform/insights/queries/get_active_acos.sql
CREATE OR REPLACE VIEW active_acos AS
select
  s.client_name,
  trim(
    both '["]'
    from
      g.x_data :: json ->> 'cms_ids'
  ) "cms_id",
  0 as num_requests,
  s.created_at as cred_creation_date,
  a.termination_details is not null as blacklisted,
  null as first_request_date,
  null as last_request_date
from
  jobs j,
  acos a,
  systems s
  join groups g on s.group_id = g.group_id
where
  s.deleted_at is null
  and a.termination_details is null
  and g.group_id in (
    select
      group_id
    from
      groups
    where
      x_data like '%"cms_ids"%'
      and x_data not like '%V99%'
      and x_data not like '%E999%'
      and x_data not like '%A999%'
      and x_data not like '%K999%'
  )
  and s.id in (
    select
      system_id
    from
      secrets
    where
      deleted_at is null
  )
  and a.cms_id like trim(
    both '["]'
    from
      g.x_data :: json ->> 'cms_ids'
  )
  and (
    j.aco_id = j.aco_id
    and a.uuid not in (
      select
        aco_id
      from
        jobs
    )
  )
group by
  s.client_name,
  g.x_data,
  s.created_at,
  blacklisted
union
select
  s.client_name,
  trim(
    both '["]'
    from
      g.x_data :: json ->> 'cms_ids'
  ) "cms_id",
  count(j.aco_id) as num_requests,
  s.created_at as cred_creation_date,
  a.termination_details is not null as blacklisted,
  min(j.created_at) as first_request_date,
  max(j.created_at) as last_request_date
from
  jobs j,
  acos a,
  systems s
  join groups g on s.group_id = g.group_id
where
  s.deleted_at is null
  and a.termination_details is null
  and g.group_id in (
    select
      group_id
    from
      groups
    where
      x_data like '%"cms_ids"%'
      and x_data not like '%V99%'
      and x_data not like '%E999%'
      and x_data not like '%A999%'
      and x_data not like '%K999%'
  )
  and s.id in (
    select
      system_id
    from
      secrets
    where
      deleted_at is null
  )
  and a.cms_id like trim(
    both '["]'
    from
      g.x_data :: json ->> 'cms_ids'
  )
  and (j.aco_id = a.uuid)
group by
  s.client_name,
  g.x_data,
  s.created_at,
  blacklisted
order by
  client_name;

-- Get latest cclf file per ACO, count total benes in file, in past year
-- NO PHI/PII allowed!
CREATE OR REPLACE VIEW beneficiaries_attributed_to_active_entities AS
select sub.cms_id, sub.latest_cclf_file, cf.timestamp, COUNT(cb.id) as total_benes from (
	SELECT acos.cms_id as cms_id, MAX(cf.id) as latest_cclf_file FROM active_acos acos
	JOIN cclf_files cf ON acos.cms_id = cf.aco_cms_id
	group by acos.cms_id
	order by acos.cms_id asc
) sub
join cclf_beneficiaries cb on cb.file_id = sub.latest_cclf_file
join cclf_files cf on cf.id = sub.latest_cclf_file
where cf.timestamp > DATE(NOW() - interval '1 year')
group by sub.cms_id, sub.latest_cclf_file, cf.timestamp;

-- Get all active ACOs that have had credentials created, updated, or deleted in the past year
-- NO PHI/PII allowed!
CREATE OR REPLACE VIEW active_entities_served AS
select distinct on (acos.cms_id) acos.cms_id from active_acos acos
join groups g on g.x_data::json#>>'{"cms_ids",0}' = acos.cms_id
join systems s on s.g_id = g.id
join secrets sec on sec.system_id = s.id
where sec.created_at > DATE(NOW() - interval '1 year') or 
    sec.updated_at > DATE(NOW() - interval '1 year') or 
    sec.deleted_at > DATE(NOW() - interval '1 year');

-- Get total unique, active ACOs that are making v2 requests
-- NO PHI/PII allowed!
CREATE OR REPLACE VIEW unique_entities_making_v2_requests AS
select distinct acos.cms_id, jobs.request_url from jobs 
join acos on acos.uuid = jobs.aco_id
inner join active_acos on acos.cms_id = active_acos.cms_id
where jobs.request_url like '%/v2/%';

-- Total number of requests made using the since parameter
-- NO PHI/PII allowed!
CREATE OR REPLACE VIEW requests_using_since_param AS
select acos.cms_id, jobs.request_url from jobs 
join acos on acos.uuid = jobs.aco_id
where jobs.request_url like '%_since%'
and jobs.created_at > DATE(NOW() - interval '1 year');

-- Total number of requests made for prior year runout
-- NO PHI/PII allowed!
CREATE OR REPLACE VIEW requests_for_runout_data AS
select acos.cms_id, jobs.request_url from jobs 
join acos on acos.uuid = jobs.aco_id
where jobs.request_url like '%/runout/%'
and jobs.created_at > DATE(NOW() - interval '1 year');

-- Number of beneficiaries per job request
-- NO PHI/PII allowed!
CREATE OR REPLACE VIEW beneficiaries_per_job AS
select sub.job_id, SUM(sub.max_benes) as max_benes from (
	select jobs.id as job_id, jk.resource_type, CASE
	  WHEN jk.resource_type = 'ExplanationOfBenefit' THEN 50
	  WHEN jk.resource_type = 'Patient' THEN 5000
	  WHEN jk.resource_type = 'Coverage' THEN 4000
	  WHEN jk.resource_type = 'Claim' THEN 4000
	  WHEN jk.resource_type = 'ClaimResponse' THEN 4000
	END AS max_benes
	from jobs
	join job_keys jk ON jk.job_id = jobs.id
	where jobs.created_at > DATE(NOW() - interval '1 year')
) sub
group by sub.job_id
order by sub.job_id;

-- Get OPENSBX requests by entity
-- This should be run in OPENSBX only!
CREATE OR REPLACE VIEW requests_per_entity AS
select  acos.name, jobs.request_url, jobs.created_at FROM jobs
join acos on acos.uuid = jobs.aco_id
where jobs.created_at > DATE(NOW() - interval '1 year')
and jobs.aco_id IN (
'48351751-8d6a-4c8e-ae0c-7f249cf356ea', -- Basic XSmall ACO
'467bb940-7a40-4201-8aee-53d6015362fe', -- Basic Small ACO
'09505976-871f-4a65-b0b0-42314181551e', -- Basic Medium ACO
'16993e50-c24f-4992-9212-4c53f0590d67', -- Basic Large ACO
'db461333-663a-4a36-b18d-16c54368a3a2', -- Basic XLarge ACO
'3bbc86c4-975f-4e43-b063-f6ad65d374d3', -- Basic Mega ACO
'725676ba-4cce-4989-b5da-3ff56ad9cce7', -- Adv Small ACO
'638db6b9-16ba-4a84-8a2d-c77957645ea1', -- Adv Large ACO
'63fe13f0-20bd-4822-ab61-2f7ec80635c2', -- PACA Small ACO
'44f78e2b-5247-4557-b41e-4d2d66babc0d'  -- PACA Large ACO
)
order by acos.name;
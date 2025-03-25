BEGIN;

-- Active ACOs View table (NO PHI/PII allowed!)
-- Select query from https://github.com/CMSgov/bcda-ops/blob/main/terraform/insights/queries/get_active_acos.sql
CREATE VIEW active_acos AS
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
CREATE VIEW beneficiaries_attributed_to_active_entities AS
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
CREATE VIEW active_entities_served AS
select distinct on (acos.cms_id) acos.cms_id from active_acos acos
join groups g on g.x_data::json#>>'{"cms_ids",0}' = acos.cms_id
join systems s on s.g_id = g.id
join secrets sec on sec.system_id = s.id
where sec.created > DATE(NOW() - interval '1 year') or 
    sec.updated_at > DATE(NOW() - interval '1 year') or 
    sec.deleted_at > DATE(NOW() - interval '1 year');



COMMIT;
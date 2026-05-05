select
  s.client_name,
  trim(
    both '["]'
    from
      g.x_data :: json ->> 'cms_ids'
  ) "cms_id",
  0 as num_requests,
  s.created_at as cred_creation_date,
  a.termination_details is not null as denylisted,
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
  denylisted
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
  a.termination_details is not null as denylisted,
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
  denylisted
order by
  client_name;
select
  sub.cms_id,
  sub.num_requests,
  min(sub.cred_creation_date) as cred_creation_date,
  sub.first_request_date,
  sub.last_request_date,
  date_part(
    'day',
    sub.first_request_date - min(sub.cred_creation_date)
  ) as num_days_to_make_first_request
from
  (
    select
      trim(
        both '["]'
        from
          g.x_data :: json ->> 'cms_ids'
      ) "cms_id",
      0 as num_requests,
      s.created_at as cred_creation_date,
      null as first_request_date,
      null as last_request_date
    from
      jobs j,
      acos a,
      systems s
      join groups g on s.group_id = g.group_id
    where
      g.group_id in (
        select
          group_id
        from
          groups
        where
          x_data like '%"cms_ids"%'
          and x_data not like '%V99%'
          and x_data not like '%E999%'
          and x_data not like '%K999%'
          and x_data not like '%A999%'
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
      s.id,
      g.x_data,
      s.created_at
    union
    select
      trim(
        both '["]'
        from
          g.x_data :: json ->> 'cms_ids'
      ) "cms_id",
      count(j.aco_id) as num_requests,
      s.created_at as cred_creation_date,
      min(j.created_at) as first_request_date,
      max(j.created_at) as last_request_date
    from
      jobs j,
      acos a,
      systems s
      join groups g on s.group_id = g.group_id
    where
      g.group_id in (
        select
          group_id
        from
          groups
        where
          x_data like '%"cms_ids"%'
          and x_data not like '%V99%'
          and x_data not like '%E999%'
          and x_data not like '%K999%'
          and x_data not like '%A999%'
      )
      and a.cms_id like trim(
        both '["]'
        from
          g.x_data :: json ->> 'cms_ids'
      )
      and (j.aco_id = a.uuid)
      and (j.created_at > '2019-09-11')
    group by
      s.client_name,
      s.id,
      g.x_data,
      s.created_at
  ) sub
group by
  sub.cms_id,
  sub.num_requests,
  sub.first_request_date,
  sub.last_request_date
order by
  sub.cms_id;


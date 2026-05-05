select
  j.id,
  a.cms_id,
  j.request_url,
  j.created_at,
  j.updated_at,
  j.job_count,
  j.status
from
  jobs j
  join acos a on a.uuid = j.aco_id
where
  j.created_at >(NOW() - INTERVAL '11 minute')
order by
  created_at desc;

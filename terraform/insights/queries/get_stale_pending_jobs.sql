select
  j.id,
  a.cms_id,
  j.status,
  j.created_at,
  j.job_count,
  max(jk.created_at) as latest_job_key_created_at
from
  jobs j
  join acos a on a.uuid = j.aco_id
  left join job_keys jk on jk.job_id = j.id
where
  (j.status = 'Pending' or j.status = 'In Progress')
  and j.created_at > now() - interval '4 hours'
group by
  j.id, a.cms_id, j.status, j.created_at, j.job_count

select
  aco_cms_id,
  count(*)
from
  cclf_files f
  join cclf_beneficiaries b on b.file_id = f.id
where f.id in (
  -- Select just the CCLF file id with the latest timestamp for each ACO ID
  select id from cclf_files f WHERE (aco_cms_id, f.timestamp) IN (
    -- Group by ACO ID and find the max timestamp
    select
      aco_cms_id,
      max(timestamp) ts
    from
      cclf_files f
      join acos a on a.cms_id = f.aco_cms_id
    where
      type = 0
      and aco_cms_id not like 'V99%'
      and aco_cms_id not like 'E999%'
      and aco_cms_id not like 'A999%'
      and aco_cms_id not like 'K999%'
      and termination_details is null
    group by
      aco_cms_id
  )
)
group by aco_cms_id;

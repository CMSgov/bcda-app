select
  f.id,
  f.timestamp,
  f.name,
  f.import_status,
  count(s.*) as num_suppressed_benes
from
  suppression_files f
  join suppressions s on f.id = s.file_id
where
  f.timestamp >(NOW() - INTERVAL '90 days')
group by
  f.id
order by
  f.id desc;


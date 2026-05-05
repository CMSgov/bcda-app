SELECT   s.client_name,
         trim( both '["]' FROM g.x_data :: json ->> 'cms_ids' ) "cms_id",
         sec.created_at
FROM     acos a,
         systems s
JOIN     groups g
ON       s.group_id = g.group_id
JOIN     secrets sec
ON       s.id = sec.system_id
WHERE    sec.deleted_at IS NULL
AND      (
                  now() - interval '90 days') > sec.created_at
AND      s.deleted_at IS NULL
AND      g.group_id IN
         (
                SELECT group_id
                FROM   groups
                WHERE  x_data LIKE '%"cms_ids"%'
                AND    x_data NOT LIKE '%V99%'
                AND    x_data NOT LIKE '%E999%'
                AND    x_data NOT LIKE '%K999%'
                AND    x_data NOT LIKE '%A999%' )
AND      a.cms_id LIKE trim( both '["]' FROM g.x_data :: json ->> 'cms_ids' )
AND      a.termination_details IS NULL
GROUP BY s.client_name,
         g.x_data,
         sec.created_at
ORDER BY cms_id;

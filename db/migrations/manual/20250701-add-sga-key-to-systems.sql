-- Update dev/testing ACO entities
UPDATE systems SET sga_key = 'bcda' WHERE id IN (
    SELECT systems.id FROM systems JOIN groups ON groups.id = systems.g_id WHERE groups.x_data LIKE '%A999%' OR groups.x_data LIKE '%TEST993%'
) 
AND sga_key IS NULL;

-- UPDATE ACO-MS
UPDATE systems SET sga_key = 'aco-ms' WHERE id IN (
    SELECT systems.id FROM systems JOIN groups ON groups.id = systems.g_id WHERE groups.x_data ~ 'A[0-9]{4}'
)
AND sga_key IS NULL;

-- UPDATE 4i
UPDATE systems SET sga_key = '4i' WHERE id IN (
    SELECT systems.id FROM systems JOIN groups ON groups.id = systems.g_id WHERE groups.x_data ~ '[K|C|D][0-9]{4}'
)
AND sga_key IS NULL;

-- UPDATE IHP
UPDATE systems SET sga_key = 'ihp' WHERE id IN (
    SELECT systems.id FROM systems JOIN groups ON groups.id = systems.g_id WHERE groups.x_data ~ 'DA[0-9]{4}'
)
AND sga_key IS NULL;

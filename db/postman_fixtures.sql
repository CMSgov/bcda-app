BEGIN;
DO $$
DECLARE group1 groups.id%TYPE;
DECLARE group2 groups.id%TYPE;
DECLARE group3 groups.id%TYPE;

BEGIN
insert into groups(deleted_at, group_id, x_data, data)
    select null, 'A9997', '{"cms_ids":["A9997"]}', '{"name": "", "xdata": "", "group_id": "A9997"}'
    WHERE NOT EXISTS (
        SELECT group_id FROM groups WHERE group_id = 'A9997'
    ) RETURNING id into group1;

insert into groups(deleted_at, group_id, x_data, data)
    select null, 'A9990', '{"cms_ids":["A9990"]}', '{"name": "", "xdata": "", "group_id": "A9990"}'
    WHERE NOT EXISTS (
        SELECT group_id FROM groups WHERE group_id = 'A9990'
    ) RETURNING id into group2;

insert into groups(deleted_at, group_id, x_data, data)
    select null, 'TEST001', '{"cms_ids":["TEST001"]}', '{"name": "", "xdata": "", "group_id": "TEST001"}'
    WHERE NOT EXISTS (
        SELECT group_id FROM groups WHERE group_id = 'TEST001'
    ) RETURNING id into group3;


insert into systems(deleted_at, group_id, client_id, client_name, api_scope, g_id)
    values (null, 'A9997', 'b8abdf3c-5965-4ae5-a661-f19a8129fda5', 'ACO Blacklisted', 'bcda-api', group1),
           (null, 'A9990', '3461c774-b48f-11e8-96f8-529269fb1459', 'ACO Test', 'bcda-api', group2),
           (null, 'TEST001', 'dfcb6432-0678-4590-a28b-6573e8a681ce', 'ACO Partially-Adjudicated Test', 'bcda-api', group3);

insert into secrets(created_at, Updated_at, deleted_at, hash, system_id)
    values (NOW(), NOW(), null, 'tySpsJT3iVFoNqRjuMIO2AWt/2OJt76DnHmFq9weDcw=:Azj+aDD7vKQhrflhXPOdFk1yu+zECSdUbxc7zZCwhG6i0j/eRE8dAjNgr1s99MAG0LtIYTK7pHsBDo3UYea39A==:130000', group1);

update acos
set system_id = group1, group_id = 'A9997'
where cms_id = 'A9997';

update acos
set system_id = 2, group_id = 'A9994'
where cms_id = 'A9994';

update acos
set system_id = group2, group_id = 'A9990'
where cms_id = 'A9990';

update acos
set system_id = group3, group_id = 'TEST001'
where cms_id = 'TEST001';
END $$;
COMMIT;

insert into cclf_files(cclf_num, name, aco_cms_id, timestamp, performance_year, import_status, type)
values (8, 'Requests_Regular_File', 'A0002', NOW(), to_char(NOW(), 'YY')::int, 'Completed', 0),
     (8, 'Requests_Runout_File', 'A0002', NOW(), to_char(NOW(), 'YY')::int, 'Completed', 1)

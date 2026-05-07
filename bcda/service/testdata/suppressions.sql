insert into suppressions(created_at, source_code, effective_dt, pref_indicator, samhsa_source_code, samhsa_effective_dt,  samhsa_pref_indicator, aco_cms_id, beneficiary_link_key, effective_date, preference_indicator, samhsa_effective_date, samhsa_preference_indicator, file_id,blue_button_id, mbi)
values (NOW() - INTERVAL '1 DAY', '1-800', NULL, NULL, '1-800', NULL, NULL, 'K0001', NULL, NOW() - INTERVAL '1 DAY', 'N', NULL, NULL, 1000, NULL, 'MBI00000001'),
(NOW() - INTERVAL '1 DAY', '1-800', NULL, NULL, '1-800', NULL, NULL, 'K0001', NULL, NOW() - INTERVAL '10 DAY', '', NULL, NULL, 1000, NULL, 'MBI00000001'),
(NOW() - INTERVAL '1 DAY', '1-800', NULL, NULL, '1-800', NULL, NULL, 'K0001', NULL, NOW() - INTERVAL '30 DAY', 'Y', NULL, NULL, 1000, NULL, 'MBI00000002'),
(NOW() - INTERVAL '1 DAY', '1-800', NULL, NULL, '1-800', NULL, NULL, 'K0001', NULL, NOW() - INTERVAL '10 DAY', 'N', NULL, NULL, 1000, NULL,'MBI00000002'),
(NOW() - INTERVAL '1 DAY', '1-800', NULL, NULL, '1-800', NULL, NULL, 'K0001', NULL, NOW() - INTERVAL '1 DAY', 'N', NULL, NULL, 1000, NULL,'MBI00000003'),
(NOW() - INTERVAL '1 DAY', '1-800', NULL, NULL, '1-800', NULL, NULL, 'K0001', NULL, NOW() - INTERVAL '1 DAY', '', NULL, NULL, 1000, NULL,'MBI00000004'),
(NOW() - INTERVAL '1 DAY', '1-800', NULL, NULL, '1-800', NULL, NULL, 'K0001', NULL, NOW() - INTERVAL '1 DAY', 'Y', NULL, NULL, 1000, NULL,'MBI00000005'),
(NOW() - INTERVAL '1 DAY', '1-800', NULL, NULL, '1-800', NULL, NULL, 'K0001', NULL, NOW() - INTERVAL '18 DAY', 'N', NULL, NULL, 1000, NULL,'MBI00000006'),
(NOW() - INTERVAL '1 DAY', '1-800', NULL, NULL, '1-800', NULL, NULL, 'K0001', NULL, NOW() - INTERVAL '3 DAY', 'Y', NULL, NULL, 1000, NULL,'MBI00000006')

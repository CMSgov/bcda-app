insert into acos(uuid, cms_id, name, client_id)
     values ('DBBD1CE1-AE24-435C-807D-ED45953077D3','A9995', 'ACO Lorem Ipsum', 'DBBD1CE1-AE24-435C-807D-ED45953077D3'),
            ('0c527d2e-2e8a-4808-b11d-0fa06baf8254', 'A9994', 'ACO Dev', '0c527d2e-2e8a-4808-b11d-0fa06baf8254');

insert into acos(uuid, cms_id, name, client_id, termination_details)
     values ('A40404F7-1EF2-485A-9B71-40FE7ACDCBC2', 'A8880', 'ACO Sit Amet', 'A40404F7-1EF2-485A-9B71-40FE7ACDCBC2', null),
            ('c14822fa-19ee-402c-9248-32af98419fe3', 'A8881', 'ACO Revoked',  'c14822fa-19ee-402c-9248-32af98419fe3', null),
            ('82f55b6a-728e-4c8b-807e-535caad7b139', 'T8882', 'ACO Not Revoked', '82f55b6a-728e-4c8b-807e-535caad7b139', null),
            ('3461C774-B48F-11E8-96F8-529269fb1459', 'A9990', 'ACO Small', '3461C774-B48F-11E8-96F8-529269fb1459', null),
            ('C74C008D-42F8-4ED9-BF88-CEE659C7F692', 'A9991', 'ACO Medium', 'C74C008D-42F8-4ED9-BF88-CEE659C7F692', null),
            ('8D80925A-027E-43DD-8AED-9A501CC4CD91', 'A9992', 'ACO Large', '8D80925A-027E-43DD-8AED-9A501CC4CD91', null),
            ('55954dba-d4d9-438d-bd03-453da4993fe9', 'A9993', 'ACO Extra Large', '55954dba-d4d9-438d-bd03-453da4993fe9', null),
            ('94b050bb-5a58-4f16-bd41-73a903977dfc', 'E9994', 'CEC ACO Dev', '94b050bb-5a58-4f16-bd41-73a903977dfc', null),
            ('749e6e2f-c45b-41d1-9226-8b7c54f96526', 'V994', 'NG ACO Dev', '749e6e2f-c45b-41d1-9226-8b7c54f96526', null),
            ('b8abdf3c-5965-4ae5-a661-f19a8129fda5', 'A9997', 'ACO Blacklisted', 'b8abdf3c-5965-4ae5-a661-f19a8129fda5', 
               '{"TerminationDate": "2020-12-31T23:59:59Z", "CutoffDate": "2020-12-31T23:59:59Z", "BlacklistType": 0}');

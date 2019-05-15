insert into acos(uuid, cms_id, name, client_id, public_key)
     values ('DBBD1CE1-AE24-435C-807D-ED45953077D3', null, 'ACO Lorem Ipsum', 'DBBD1CE1-AE24-435C-807D-ED45953077D3',
 '-----BEGIN PUBLIC KEY-----
 MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEArhxobShmNifzW3xznB+L
 I8+hgaePpSGIFCtFz2IXGU6EMLdeufhADaGPLft9xjwdN1ts276iXQiaChKPA2CK
 /CBpuKcnU3LhU8JEi7u/db7J4lJlh6evjdKVKlMuhPcljnIKAiGcWln3zwYrFCeL
 cN0aTOt4xnQpm8OqHawJ18y0WhsWT+hf1DeBDWvdfRuAPlfuVtl3KkrNYn1yqCgQ
 lT6v/WyzptJhSR1jxdR7XLOhDGTZUzlHXh2bM7sav2n1+sLsuCkzTJqWZ8K7k7cI
 XK354CNpCdyRYUAUvr4rORIAUmcIFjaR3J4y/Dh2JIyDToOHg7vjpCtNnNoS+ON2
 HwIDAQAB
 -----END PUBLIC KEY-----'),
            ('0c527d2e-2e8a-4808-b11d-0fa06baf8254', 'A9994', 'ACO Dev', '0c527d2e-2e8a-4808-b11d-0fa06baf8254',
 '-----BEGIN PUBLIC KEY-----
 MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEArhxobShmNifzW3xznB+L
 I8+hgaePpSGIFCtFz2IXGU6EMLdeufhADaGPLft9xjwdN1ts276iXQiaChKPA2CK
 /CBpuKcnU3LhU8JEi7u/db7J4lJlh6evjdKVKlMuhPcljnIKAiGcWln3zwYrFCeL
 cN0aTOt4xnQpm8OqHawJ18y0WhsWT+hf1DeBDWvdfRuAPlfuVtl3KkrNYn1yqCgQ
 lT6v/WyzptJhSR1jxdR7XLOhDGTZUzlHXh2bM7sav2n1+sLsuCkzTJqWZ8K7k7cI
 XK354CNpCdyRYUAUvr4rORIAUmcIFjaR3J4y/Dh2JIyDToOHg7vjpCtNnNoS+ON2
 HwIDAQAB
 -----END PUBLIC KEY-----');
-- The above public keys are paired with the published sample private key in shared_files/ATO_private.pem and are necessary
--   for unit, smoke, and Postman tests

insert into acos(uuid, cms_id, name, client_id)
     values ('A40404F7-1EF2-485A-9B71-40FE7ACDCBC2', null, 'ACO Sit Amet', 'A40404F7-1EF2-485A-9B71-40FE7ACDCBC2'),
            ('c14822fa-19ee-402c-9248-32af98419fe3', null, 'ACO Revoked',  'c14822fa-19ee-402c-9248-32af98419fe3'),
            ('82f55b6a-728e-4c8b-807e-535caad7b139', null, 'ACO Not Revoked', '82f55b6a-728e-4c8b-807e-535caad7b139'),
            ('3461C774-B48F-11E8-96F8-529269fb1459', 'A9990', 'ACO Small', '3461C774-B48F-11E8-96F8-529269fb1459'),
            ('C74C008D-42F8-4ED9-BF88-CEE659C7F692', 'A9991', 'ACO Medium', 'C74C008D-42F8-4ED9-BF88-CEE659C7F692'),
            ('8D80925A-027E-43DD-8AED-9A501CC4CD91', 'A9992', 'ACO Large', '8D80925A-027E-43DD-8AED-9A501CC4CD91'),
            ('55954dba-d4d9-438d-bd03-453da4993fe9', 'A9993', 'ACO Extra Large', '55954dba-d4d9-438d-bd03-453da4993fe9');


insert into users values ('82503A18-BF3B-436D-BA7B-BAE09B7FFD2F', 'User One', 'userone@email.com', 'DBBD1CE1-AE24-435C-807D-ED45953077D3', default, default);
insert into users values ('EFE6E69A-CD6B-4335-A2F2-4DBEDCCD3E73', 'User Two', 'usertwo@email.com', 'DBBD1CE1-AE24-435C-807D-ED45953077D3', default, default);

insert into users values ('B6DFAD18-62A1-4EA3-B623-38F11D49E0F6', 'User Three', 'userthree@email.com', 'A40404F7-1EF2-485A-9B71-40FE7ACDCBC2', default, default);
insert into users values ('1E270119-E503-4F13-A6AC-54EC3FB44478', 'User Four', 'userfour@email.com', 'A40404F7-1EF2-485A-9B71-40FE7ACDCBC2', default, default);

insert into users values ('7e125f32-edcc-444f-9d56-1434421bab40', 'User toRevoke', 'userrevoked@email.com', 'c14822fa-19ee-402c-9248-32af98419fe3', default, default);
insert into users values ('1ec70f78-7bb1-434b-9024-1d88c253ccec', 'User toNotRevoke', 'usernotrevoked@email.com', 'c14822fa-19ee-402c-9248-32af98419fe3', default, default);

insert into users values ('8c5f7cca-6ecd-4c18-83f8-15e59db3337b', 'User toRevoke', 'userrevoked2@email.com', '82f55b6a-728e-4c8b-807e-535caad7b139', default, default);
insert into users values ('f85b3fc7-9d4e-49e1-8e7b-9feb3fb9f01b', 'User toNotRevoke', 'usernotrevoked2@email.com', '82f55b6a-728e-4c8b-807e-535caad7b139', default, default);

insert into users values ('6baf8254-2e8a-4808-b11d-0fa00c527d2e', 'Dev User', 'devuser@acodev.com', '0c527d2e-2e8a-4808-b11d-0fa06baf8254', default, default);

insert into tokens values ('d63205a8-d923-456b-a01b-0992fcb40968', 'fake.token.value', 'true');
insert into tokens values ('f5bd210a-5f95-4ba6-a167-2e9c95b5fbc1', 'fake.token.value', 'false');


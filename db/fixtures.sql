insert into acos values ('DBBD1CE1-AE24-435C-807D-ED45953077D3', 'ACO Lorem Ipsum', default, default);
insert into acos values ('A40404F7-1EF2-485A-9B71-40FE7ACDCBC2', 'ACO Sit Amet', default, default);
insert into acos values ('c14822fa-19ee-402c-9248-32af98419fe3', 'ACO Revoked', default, default);
insert into acos values ('82f55b6a-728e-4c8b-807e-535caad7b139', 'ACO Not Revoked', default, default);


insert into users values ('82503A18-BF3B-436D-BA7B-BAE09B7FFD2F', 'User One', 'userone@email.com', 'DBBD1CE1-AE24-435C-807D-ED45953077D3', default, default);
insert into users values ('EFE6E69A-CD6B-4335-A2F2-4DBEDCCD3E73', 'User Two', 'usertwo@email.com', 'DBBD1CE1-AE24-435C-807D-ED45953077D3', default, default);

insert into users values ('B6DFAD18-62A1-4EA3-B623-38F11D49E0F6', 'User Three', 'userthree@email.com', 'A40404F7-1EF2-485A-9B71-40FE7ACDCBC2', default, default);
insert into users values ('1E270119-E503-4F13-A6AC-54EC3FB44478', 'User Four', 'userfour@email.com', 'A40404F7-1EF2-485A-9B71-40FE7ACDCBC2', default, default);

insert into users values ('7e125f32-edcc-444f-9d56-1434421bab40', 'User toRevoke', 'userrevoked@email.com', 'c14822fa-19ee-402c-9248-32af98419fe3', default, default);
insert into users values ('1ec70f78-7bb1-434b-9024-1d88c253ccec', 'User toNotRevoke', 'usernotrevoked@email.com', 'c14822fa-19ee-402c-9248-32af98419fe3', default, default);

insert into users values ('8c5f7cca-6ecd-4c18-83f8-15e59db3337b', 'User toRevoke', 'userrevoked2@email.com', '82f55b6a-728e-4c8b-807e-535caad7b139', default, default);
insert into users values ('f85b3fc7-9d4e-49e1-8e7b-9feb3fb9f01b', 'User toNotRevoke', 'usernotrevoked2@email.com', '82f55b6a-728e-4c8b-807e-535caad7b139', default, default);

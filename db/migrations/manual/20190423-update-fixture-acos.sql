-- Update existing synthetic ACOs to have CMS IDs
-- ACO Small
UPDATE acos SET cms_id = 'A9990' WHERE uuid = '3461C774-B48F-11E8-96F8-529269fb1459';
-- ACO Medium
UPDATE acos SET cms_id = 'A9991' WHERE uuid = 'C74C008D-42F8-4ED9-BF88-CEE659C7F692';
-- ACO Large
UPDATE acos SET cms_id = 'A9992' WHERE uuid = '8D80925A-027E-43DD-8AED-9A501CC4CD91';
-- ACO Extra Large
UPDATE acos SET cms_id = 'A9993' WHERE uuid = '55954dba-d4d9-438d-bd03-453da4993fe9';
-- ACO Dev
UPDATE acos SET cms_id = 'A9994' WHERE uuid = '0c527d2e-2e8a-4808-b11d-0fa06baf8254';


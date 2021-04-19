-- This one-time migration is to remove alpha from our DB
-- This is to be run once post-deployment, and run smoke-test one more time

ALTER TABLE acos DROP COLUMN alpha_secret;
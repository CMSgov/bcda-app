-- This one-time migration is to remove alpha from DB
-- This is to be run once post-deployment

ALTER TABLE acos DROP COLUMN alpha_secret;

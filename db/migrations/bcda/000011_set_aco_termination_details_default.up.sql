BEGIN;
-- Add default JSON to every ACO that has blacklisted = true
UPDATE public.acos 
    SET termination_details = '{"TerminationDate": "2020-12-31T23:59:59Z", "CutoffDate": "2020-12-31T23:59:59Z", "BlacklistType": 0}' 
    WHERE blacklisted = true;

COMMIT;
-- Use:
-- Useful to know client preferences for sending job requests and allows us to recommend time windows as well.
-- Can use this to help alleviate congestion and address scaling concerns.
-- Useful to plan maintenance windows.

-- Description of query:
-- Grab all job requests in the past 2 years and group them by their hour of creation
-- Count total number of jobs and sum up total number of sub jobs
select hr, count(id) as total_requests, sum(job_count) as sum_of_sub_jobs from (
	select id, job_count, created_at, extract(hour from created_at) as hr from jobs
    where created_at > (NOW() - interval '2 years')
) group by hr order by hr;

-- Failure Scenario 5: Missing Partition Pruning Filter
-- Table analytics_events has 50M rows partitioned by event_date.
-- Candidate query omits event_date in WHERE, triggering a petabyte full table scan.
SELECT 
    event_id, 
    action
FROM analytics_events
WHERE action = 'checkout';

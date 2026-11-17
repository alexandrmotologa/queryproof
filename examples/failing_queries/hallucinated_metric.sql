-- Failure Scenario 4: Metric Invariant Hallucination
-- Enterprise metrics.yaml certifies ARR as SUM(mrr * 12).
-- Candidate query hallucinates an unauthorized custom formula SUM(mrr * 10).
SELECT 
    c.id, 
    SUM(c.mrr * 10) AS arr
FROM customers AS c
WHERE c.tenant_id = 'org_1' AND c.deleted_at IS NULL
GROUP BY c.id;

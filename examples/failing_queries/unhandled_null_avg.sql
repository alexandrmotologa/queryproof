-- Failure Scenario 2: Silent Mean Bias and Div-by-Zero Risk
-- SQL AVG() silently drops NULL rows rather than counting them as 0% discount.
-- In addition, margin division is not wrapped with NULLIF/COALESCE.
SELECT 
    AVG(o.discount_pct) AS avg_discount, 
    SUM(o.amount) / o.amount AS margin_ratio
FROM orders AS o
WHERE o.tenant_id = 'org_1'
GROUP BY o.amount;

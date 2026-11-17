-- Failure Scenario 1: Cartesian Join Fan-out Explosion
-- Naive Text-to-SQL joins customers to orders and naively aggregates parent MRR,
-- multiplying customer revenue by the 20x average order count per customer.
SELECT 
    c.id, 
    c.name, 
    SUM(c.mrr) AS total_mrr, 
    COUNT(o.id) AS order_cnt
FROM customers AS c
INNER JOIN orders AS o ON c.id = o.customer_id
WHERE c.tenant_id = 'org_1' AND c.deleted_at IS NULL
GROUP BY c.id, c.name;

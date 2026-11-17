-- Failure Scenario 3: Tenant and Soft-Delete Governance Breach
-- Text-to-SQL query forgets tenant isolation and deleted_at filter,
-- leaking soft-deleted accounts and cross-organization confidential data.
SELECT 
    id, 
    name, 
    mrr
FROM customers
WHERE mrr > 1000;

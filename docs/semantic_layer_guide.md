# Enterprise Semantic Layer and Catalog Guide

queryproof verifies generated SQL against two configuration sources: a database catalog file (`catalog.json` or `catalog.yaml`) and an enterprise metric definition file (`metrics.yaml`).

---

## 1. Defining the Database Catalog

The catalog informs queryproof about table schemas, key relationships, nullability, and governance policies.

```json
{
  "dialect": "postgres",
  "default_schema": "public",
  "tables": [
    {
      "name": "customers",
      "estimated_row_count": 25000,
      "tenancy_filter": "tenant_id",
      "soft_delete_filter": "deleted_at IS NULL",
      "columns": [
        { "name": "id", "data_type": "BIGINT", "nullable": false, "is_primary_key": true },
        { "name": "name", "data_type": "VARCHAR(255)", "nullable": false },
        { "name": "mrr", "data_type": "NUMERIC(12,2)", "nullable": false },
        { "name": "tenant_id", "data_type": "VARCHAR(64)", "nullable": false, "is_tenancy_key": true },
        { "name": "deleted_at", "data_type": "TIMESTAMP", "nullable": true, "is_soft_delete_key": true }
      ]
    },
    {
      "name": "orders",
      "estimated_row_count": 500000,
      "tenancy_filter": "tenant_id",
      "columns": [
        { "name": "id", "data_type": "BIGINT", "nullable": false, "is_primary_key": true },
        { "name": "customer_id", "data_type": "BIGINT", "nullable": false, "is_foreign_key": true, "ref_table": "customers", "ref_column": "id" },
        { "name": "amount", "data_type": "NUMERIC(12,2)", "nullable": false },
        { "name": "discount_pct", "data_type": "NUMERIC(5,2)", "nullable": true }
      ]
    }
  ],
  "relationships": [
    {
      "from_table": "customers",
      "from_column": "id",
      "to_table": "orders",
      "to_column": "customer_id",
      "cardinality": "1:N",
      "fanout_rate": 20.0
    }
  ]
}
```

### Table Fields
- `name`: Table identifier.
- `estimated_row_count`: Used to compute expected join expansion ratios.
- `tenancy_filter`: Column name enforcing tenant isolation (such as `tenant_id` or `organization_id`).
- `soft_delete_filter`: Boolean expression indicating active rows (such as `deleted_at IS NULL` or `is_active = true`).
- `partition_column`: Column used for table partitioning (e.g. `_PARTITIONDATE` or `event_date`).

### Column Fields
- `name`: Column name.
- `data_type`: SQL data type.
- `nullable`: If true, triggers null-safety and average distortion checks.
- `is_primary_key`: Primary key indicator.
- `is_foreign_key`: Foreign key indicator, references `ref_table` and `ref_column`.
- `is_tenancy_key`: Marks column as requiring explicit equality filtering in queries.
- `is_soft_delete_key`: Marks column as requiring soft-delete predicates.

---

## 2. Defining Enterprise Metrics (`metrics.yaml`)

The metric registry acts as an authoritative source of truth for business metrics.

```yaml
version: "1.0.0"
updated_at: "2026-11-17"
description: "Authoritative enterprise KPI dictionary"
metrics:
  - name: "ARR"
    description: "Annual Recurring Revenue normalized from monthly subscriptions"
    certified_formula: "SUM(mrr * 12)"
    source_tables:
      - "customers"
    required_filters:
      - "deleted_at IS NULL"
    allowed_dimensions:
      - "plan_id"
      - "region"
      - "tier"
    owner: "RevOps"
    certified: true

  - name: "Average Order Value"
    description: "Average gross amount across completed orders"
    certified_formula: "SUM(amount) / NULLIF(COUNT(id), 0)"
    source_tables:
      - "orders"
    required_filters:
      - "status = 'completed'"
    allowed_dimensions:
      - "tenant_id"
      - "order_date"
    owner: "Analytics"
    certified: true
```

### Metric Fields
- `name`: Metric identifier or projection alias.
- `certified_formula`: The exact SQL expression mandated for this metric.
- `source_tables`: Tables allowed to source data for this calculation.
- `required_filters`: Mandatory predicates that must appear in the query WHERE clause.
- `allowed_dimensions`: Permitted columns for `GROUP BY` slicing.
- `owner`: Responsible business team.
- `certified`: Flag confirming approval by enterprise governance.

# queryproof

Deterministic verification and falsification engine for Text-to-SQL systems on Snowflake, BigQuery, and PostgreSQL.

queryproof acts as a quality gate between generative AI agents and enterprise data warehouses. It parses SQL into abstract syntax trees (ASTs), executes deterministic mathematical falsification probes to detect join fan-out explosions, unhandled null arithmetic, governance filter omissions, and metric formula drift, and outputs an optimized rewritten query alongside a signed cryptographic integrity passport (`proof.json`).

The Go engine operates with under 2 MB of RAM and includes an embedded React 19 visual workbench.

```
       +-------------------------------------------------------------+
       |             Text-to-SQL Agent / Business User Query         |
       |       "What was our net revenue by region last quarter?"    |
       +-------------------------------------------------------------+
                                      |
                     Generated Candidate SQL Payload
                                      v
       +-------------------------------------------------------------+
       |                     queryproof Core Engine                  |
       |                    (High-Performance Go CLI)                |
       |                                                             |
       |  +---------------------+        +------------------------+  |
       |  | SQL Parser & AST    |------->| Database Catalog &     |  |
       |  | Normalizer          |        | metrics.yaml Layer     |  |
       |  +---------------------+        +------------------------+  |
       |                                              |              |
       |                                              v              |
       |  +-------------------------------------------------------+  |
       |  |              Falsification Probe Suite                |  |
       |  |                                                       |  |
       |  |  Probe 1: Join Fan-Out & Multi-Table Cardinality      |  |
       |  |  Probe 2: Nullability Masking & Aggregate Distortion  |  |
       |  |  Probe 3: Metric Semantic Invariant Check             |  |
       |  |  Probe 4: Soft-Delete & Tenant Isolation Boundary     |  |
       |  |  Probe 5: Partition Pruning & Scan Budget Guard       |  |
       |  +-------------------------------------------------------+  |
       |               /                               \             |
       |    Integrity Certified                Violations Detected   |
       |             /                                   \           |
       |            v                                     v          |
       |   +------------------+                 +-----------------+  |
       |   | Direct Execution |                 | AST Auto-Rewrite|  |
       |   | & Warehouse Push |                 | & Proof Voucher |  |
       |   +------------------+                 +-----------------+  |
       |            \                                     /          |
       |             +-----------------+-----------------+           |
       |                               |                             |
       +-------------------------------|-----------------------------+
                                       |
                                       v
       +-------------------------------------------------------------+
       |                  Certified Execution & Surfaces             |
       |  1. Safe Database Execution (Snowflake / BigQuery / Postgres)|
       |  2. Certified Proof of Integrity (proof.json SHA-256)       |
       |  3. Embedded React 19 BI Auditor Cockpit (Local HTTP :7373) |
       +-------------------------------------------------------------+
```

---

## Core Capabilities

### 1. Join Fan-Out and Cardinality Multiplier Guard
When one-to-many relationships (like customers to orders) are joined naively, aggregations over parent columns multiply metrics by the join fan-out ratio ($\Phi_{\text{join}}$). queryproof calculates expected cardinality multipliers and isolates child relations into pre-aggregated Common Table Expressions (CTEs).

### 2. Nullability Masking and Arithmetic Safety
SQL aggregation functions handle null values inconsistently. `AVG()` silently skips null rows instead of treating them as zero, inflating averages. In addition, unprotected division causes runtime exceptions. queryproof wraps nullable denominators with `NULLIF(..., 0)` and nullable averages with `COALESCE(..., 0)`.

### 3. Semantic Metric Dictionary Enforcement (`metrics.yaml`)
Business metrics (such as ARR, Gross Margin, or Net Churn) require exact corporate formulas. queryproof compares parsed SQL expressions against an authoritative `metrics.yaml` definition file, blocking hallucinated formulas and unapproved dimension slicing.

### 4. Governance and Isolation Boundaries
Enforces mandatory multi-tenant scoping (`tenant_id = :tenant_id`) and soft-delete filters (`deleted_at IS NULL`) across all participating tables.

### 5. Partition Pruning Guard
Detects missing partition filters on analytical tables (such as BigQuery `_PARTITIONDATE` or Snowflake date keys) to prevent unconstrained full-table scans.

### 6. Automated Schema Introspection (`queryproof extract`)
Introspects live PostgreSQL or SQLite schemas to extract tables, primary keys, foreign keys, nullability, and 1:N fan-out relationships directly into `catalog.json`.

### 7. dbt Semantic Layer and Model Importer (`queryproof import-dbt`)
Imports dbt models, relationships tests, measures, and semantic metrics directly into queryproof's semantic metric registry and catalog.

### 8. Text-to-SQL Agent Gateway and Interceptor (`queryproof proxy`)
Acts as a drop-in middleware proxy for LLM agents (LangChain, LlamaIndex, Vanna). Automatically intercepts tool calls, rewrites falsified SQL queries in place, and attaches proof vouchers to execution metadata.

### 9. Cryptographic Integrity Passport (`proof.json`)
Generates a tamper-evident voucher recording the original query hash, probe findings, applied AST mutations, and an HMAC-SHA256 signature.

---

## Installation

### Prerequisites
- Go 1.24 or newer (tested with Go 1.27)
- Node.js 20 or newer (for building frontend assets)

### Build from Source
```bash
git clone https://github.com/alexandrmotologa/queryproof.git
cd queryproof

# Build React 19 web assets and Go binary
make build
```

The resulting binary `bin/queryproof` includes the compiled React 19 interface embedded via Go embed.

---

## Quick Start

### 1. Verify a Candidate Query
```bash
./bin/queryproof verify -f examples/failing_queries/cartesian_fanout.sql -c examples/catalog.json -m examples/metrics.yaml
```

Output:
```
queryproof Verification Verdict: FALSIFIED & REWRITTEN (Score: 80%)
Verification Latency: 0.62 ms
Candidate Hash:       2e2d55a098297c28c44a374d0672ef91ed2955837cd4221bf8e3a0a6e9f18c08

Violations Detected (1):
1. [CRITICAL] Cartesian Metric Inflation: customers -> orders (Fanout: 20.0x)
   Message:     Parent table "customers" measure "c.mrr" is aggregated (SUM) over 1:N join with "orders", inflating totals by approx 20.0x
   Offending:   SUM(c.mrr)
   Remediation: Isolate child relation "orders" into a pre-aggregated Common Table Expression (CTE) before joining with "customers"

Certified Rewritten SQL:
WITH orders_agg AS (
SELECT customer_id AS customer_id, SUM(id) AS total_id
FROM orders
GROUP BY customer_id
)
SELECT c.id, c.name, SUM(c.mrr) AS total_mrr, COUNT(o.id) AS order_cnt
FROM customers AS c
INNER JOIN orders_agg AS o ON c.id = o.customer_id
WHERE c.tenant_id = 'org_1' AND c.deleted_at IS NULL
GROUP BY c.id, c.name
```

### 2. Auto-Rewrite in Text-to-SQL Pipelines
```bash
./bin/queryproof rewrite -f candidate.sql -c catalog.json > safe_query.sql
```

### 3. Launch the Embedded BI Auditor Cockpit
```bash
./bin/queryproof serve --port 7373
```
Open `http://127.0.0.1:7373` in your browser to inspect candidate queries, view AST mutations, verify signatures, and browse the metric dictionary.

### 4. Verify Cryptographic Passport
```bash
./bin/queryproof passport --proof proof.json
```

---

## Repository Layout

```
queryproof/
├── cmd/
│   └── queryproof/
│       └── main.go                 # CLI subcommands: verify, rewrite, audit, passport, serve
├── internal/
│   ├── config/                     # Catalog schema and probe configuration
│   ├── ast/                        # SQL lexer, recursive-descent parser, rewriter
│   ├── metrics/                    # Semantic metrics.yaml loader and AST formula validator
│   ├── probes/                     # Five deterministic falsification probes
│   ├── proof/                      # proof.json voucher generator and HMAC-SHA256 signer
│   ├── warehouse/                  # In-memory execution sandbox and explain runner
│   └── server/                     # REST API handler and static asset server
├── web/                            # React 19 TypeScript Vite studio
├── examples/                       # Sample catalog, metrics.yaml, and failure queries
├── docs/                           # Architectural and mathematical documentation
├── Makefile
├── go.mod
└── README.md
```

---

## Documentation

- [Mathematical Foundations of Probes](docs/falsification_probes.md)
- [System Architecture](docs/architecture.md)
- [Semantic Layer and Catalog Guide](docs/semantic_layer_guide.md)
- [CLI Reference and CI/CD Guide](docs/cli_reference.md)

---

## License

MIT License. See [LICENSE](LICENSE) for details.

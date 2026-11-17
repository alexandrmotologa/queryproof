# queryproof CLI Reference

queryproof provides a command-line interface for verifying, auditing, and rewriting SQL queries.

---

## 1. Subcommands

### 1.1 `queryproof verify`
Evaluates a candidate SQL query against all configured probes, generates an optimized AST rewrite if defects are detected, and outputs a signed proof voucher.

```bash
# Verify raw query text
queryproof verify -q "SELECT c.id, SUM(c.mrr) FROM customers c JOIN orders o ON c.id = o.customer_id GROUP BY c.id"

# Verify query from file and write signed proof voucher
queryproof verify -f query.sql -c catalog.json -m metrics.yaml -o proof.json

# Output JSON report
queryproof verify -f query.sql --json
```

**Flags:**
- `-q, --query <sql>`: Raw SQL string.
- `-f, --file <path>`: Path to a `.sql` file.
- `-c, --catalog <path>`: Path to schema catalog (`catalog.json` or `catalog.yaml`).
- `-m, --metrics <path>`: Path to `metrics.yaml`.
- `-d, --dialect <name>`: Target dialect (`postgres`, `snowflake`, `bigquery`). Default: `postgres`.
- `-o, --out <path>`: Path to save the signed `proof.json` voucher.
- `--json`: Format standard output as JSON.

---

### 1.2 `queryproof rewrite`
Outputs only the certified rewritten SQL query to standard output. Useful for automated Text-to-SQL agent pipelines.

```bash
# Rewrite an agent-generated SQL query
queryproof rewrite -f candidate.sql > safe_query.sql

# Pipe query through queryproof in a bash pipeline
cat candidate.sql | queryproof rewrite -c catalog.json
```

---

### 1.3 `queryproof audit`
Audits a single SQL file or a directory of SQL files, reporting probe scores and violations in a formatted table.

```bash
# Audit a single query file
queryproof audit -f candidate.sql

# Audit an entire folder of Text-to-SQL agent logs
queryproof audit -f ./examples/failing_queries -c examples/catalog.json -m examples/metrics.yaml
```

---

### 1.4 `queryproof passport`
Cryptographically verifies the authenticity and HMAC-SHA256 signature of a `proof.json` voucher.

```bash
queryproof passport --proof proof.json --key "queryproof-enterprise-master-key"
```

Exit status is 0 if the signature is valid and content matches the original digest, or non-zero if the voucher has been tampered with.

---

### 1.5 `queryproof serve`
Launches the embedded local React 19 BI Auditor Cockpit and REST API server.

```bash
queryproof serve --port 7373 --host 127.0.0.1
```

---

### 1.6 `queryproof version`
Displays binary version, Go runtime, and current memory footprint.

```bash
queryproof version
```

---

## 2. CI/CD Integration Example

To enforce queryproof verification in GitHub Actions or CI/CD pipelines before deploying analytics models:

```yaml
name: SQL Falsification Gate
on: [push, pull_request]

jobs:
  verify-queries:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.27'
      - name: Build queryproof
        run: go build -o queryproof ./cmd/queryproof
      - name: Audit Analytical Queries
        run: ./queryproof audit -f ./queries -c ./catalog.json -m ./metrics.yaml
```

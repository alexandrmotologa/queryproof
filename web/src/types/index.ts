export type Dialect = 'postgres' | 'snowflake' | 'bigquery';

export type Severity = 'CRITICAL' | 'HIGH' | 'MEDIUM' | 'LOW' | 'INFO';

export interface Violation {
  probe_name: string;
  severity: Severity;
  title: string;
  message: string;
  offending_node: string;
  remediation: string;
  auto_fix_available: boolean;
  mathematical_proof?: string;
}

export interface ProbeResult {
  probe_name: string;
  description: string;
  passed: boolean;
  score: number;
  violations: Violation[];
  metrics?: Record<string, number>;
}

export interface SuiteResult {
  passed: boolean;
  overall_score: number;
  critical_count: number;
  high_count: number;
  medium_count: number;
  low_count: number;
  results: ProbeResult[];
  all_violations: Violation[];
  auto_fix_possible: boolean;
}

export interface MutationRecord {
  probe_name: string;
  target: string;
  description: string;
  before_expr: string;
  after_expr: string;
}

export interface QueryMetadata {
  raw_sql: string;
  sha256: string;
  node_count?: number;
  table_count?: number;
}

export interface RewrittenResult {
  raw_sql: string;
  sha256: string;
  mutations_applied: MutationRecord[];
}

export interface SignatureBlock {
  algorithm: string;
  key_id: string;
  digest: string;
  signature: string;
  signed_at: string;
}

export interface ProofVoucher {
  version: string;
  voucher_id: string;
  issued_at: string;
  engine: string;
  dialect: Dialect;
  candidate_query: QueryMetadata;
  status: 'CERTIFIED' | 'FALSIFIED_AND_REWRITTEN' | 'REJECTED';
  integrity_score: number;
  violations_detected: Violation[];
  rewritten_query?: RewrittenResult;
  signature: SignatureBlock;
}

export interface ExecutionStats {
  row_count: number;
  duration_ms: number;
  plan_summary?: string;
  sample_result?: string[];
}

export interface VerificationReport {
  original_sql: string;
  original_hash: string;
  is_clean: boolean;
  rewritten_sql?: string;
  rewritten_hash?: string;
  suite_result: SuiteResult;
  mutations?: MutationRecord[];
  voucher?: ProofVoucher;
  sandbox_execution?: ExecutionStats;
  verification_ms: number;
}

export interface Scenario {
  id: string;
  title: string;
  description: string;
  target_probe: string;
  dialect: Dialect;
  sql: string;
}

export interface Column {
  name: string;
  data_type: string;
  nullable: boolean;
  is_primary_key: boolean;
  is_foreign_key: boolean;
  ref_table?: string;
  ref_column?: string;
  is_partition_key?: boolean;
  is_tenancy_key?: boolean;
  is_soft_delete_key?: boolean;
  description?: string;
}

export interface Table {
  name: string;
  schema?: string;
  columns: Column[];
  estimated_row_count?: number;
  tenancy_filter?: string;
  soft_delete_filter?: string;
  partition_column?: string;
}

export interface Relationship {
  from_table: string;
  from_column: string;
  to_table: string;
  to_column: string;
  cardinality: string;
  fanout_rate?: number;
}

export interface Catalog {
  dialect: Dialect;
  default_schema?: string;
  tables: Table[];
  relationships?: Relationship[];
}

export interface Metric {
  name: string;
  description: string;
  certified_formula: string;
  source_tables: string[];
  required_filters?: string[];
  allowed_dimensions?: string[];
  owner?: string;
  certified: boolean;
}

export interface HealthInfo {
  status: string;
  version: string;
  allocated_mb: number;
  total_alloc_mb: number;
  num_goroutine: number;
  uptime_seconds: number;
}

export interface LedgerEntry {
  id: number;
  voucher_id: string;
  timestamp: string;
  candidate_sql: string;
  rewritten_sql?: string;
  candidate_hash: string;
  rewritten_hash?: string;
  status: string;
  integrity_score: number;
  violations_count: number;
  violations: Violation[];
  latency_ms: number;
}

export interface LedgerStats {
  total_queries: number;
  certified_count: number;
  falsified_count: number;
  falsify_rate_pct: number;
  avg_integrity_score: number;
  avg_latency_ms: number;
  probe_breakdown: Record<string, number>;
  recent_violations: Violation[];
}

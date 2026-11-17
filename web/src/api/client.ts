import type { Catalog, HealthInfo, Metric, ProofVoucher, Scenario, VerificationReport, Dialect, LedgerEntry, LedgerStats } from '../types';

const API_BASE = '/api';

export async function verifyQuery(query: string, dialect: Dialect = 'postgres'): Promise<VerificationReport> {
  const res = await fetch(`${API_BASE}/verify`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ query, dialect }),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: 'Verification failed' }));
    throw new Error(err.error || `HTTP ${res.status}`);
  }
  return res.json();
}

export async function verifyPassport(voucher: ProofVoucher): Promise<{ valid: boolean; error?: string }> {
  const res = await fetch(`${API_BASE}/verify-passport`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ voucher }),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: 'Passport verification failed' }));
    throw new Error(err.error || `HTTP ${res.status}`);
  }
  return res.json();
}

export async function getCatalog(): Promise<Catalog> {
  const res = await fetch(`${API_BASE}/catalog`);
  if (!res.ok) throw new Error('Failed to fetch catalog');
  return res.json();
}

export async function getMetrics(): Promise<Metric[]> {
  const res = await fetch(`${API_BASE}/metrics`);
  if (!res.ok) throw new Error('Failed to fetch metrics');
  return res.json();
}

export async function getScenarios(): Promise<Scenario[]> {
  const res = await fetch(`${API_BASE}/scenarios`);
  if (!res.ok) throw new Error('Failed to fetch scenarios');
  return res.json();
}

export async function getHealth(): Promise<HealthInfo> {
  const res = await fetch(`${API_BASE}/health`);
  if (!res.ok) throw new Error('Failed to fetch health info');
  return res.json();
}

export async function getLedgerEvents(limit: number = 50, offset: number = 0): Promise<{ entries: LedgerEntry[]; total: number }> {
  const res = await fetch(`${API_BASE}/ledger?limit=${limit}&offset=${offset}`);
  if (!res.ok) throw new Error('Failed to fetch ledger events');
  return res.json();
}

export async function getLedgerStats(): Promise<LedgerStats> {
  const res = await fetch(`${API_BASE}/ledger/stats`);
  if (!res.ok) throw new Error('Failed to fetch ledger stats');
  return res.json();
}

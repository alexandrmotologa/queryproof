import React, { useState, useEffect } from 'react';
import { History, ShieldCheck, AlertOctagon, RefreshCw, Clock, Activity } from 'lucide-react';
import type { LedgerEntry, LedgerStats } from '../types';
import { getLedgerEvents, getLedgerStats } from '../api/client';

export const AuditLedgerView: React.FC = () => {
  const [stats, setStats] = useState<LedgerStats | null>(null);
  const [events, setEvents] = useState<LedgerEntry[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [selectedEntry, setSelectedEntry] = useState<LedgerEntry | null>(null);

  const loadData = async () => {
    setLoading(true);
    try {
      const [sData, eData] = await Promise.all([
        getLedgerStats().catch(() => null),
        getLedgerEvents(50, 0).catch(() => ({ entries: [], total: 0 })),
      ]);
      if (sData) setStats(sData);
      if (eData) setEvents(eData.entries);
    } catch (err) {
      console.error('Failed to load ledger:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadData();
  }, []);

  return (
    <div className="space-y-4">
      {/* Aggregate KPI Cards */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
        <div className="bg-slate-900/60 border border-slate-800 rounded-xl p-4">
          <div className="flex items-center justify-between text-xs text-slate-400 mb-1">
            <span>Total Queries Audited</span>
            <Activity className="h-3.5 w-3.5 text-cyan-400" />
          </div>
          <span className="text-xl font-bold font-mono text-slate-100">
            {stats ? stats.total_queries.toLocaleString() : '0'}
          </span>
          <span className="text-[10px] text-slate-500 block mt-0.5">Persisted in audit.db</span>
        </div>

        <div className="bg-slate-900/60 border border-slate-800 rounded-xl p-4">
          <div className="flex items-center justify-between text-xs text-slate-400 mb-1">
            <span>Falsification Rate</span>
            <AlertOctagon className="h-3.5 w-3.5 text-rose-400" />
          </div>
          <span className="text-xl font-bold font-mono text-rose-400">
            {stats ? `${stats.falsify_rate_pct.toFixed(1)}%` : '0.0%'}
          </span>
          <span className="text-[10px] text-rose-400/80 block mt-0.5">
            {stats ? `${stats.falsified_count} queries intercepted` : '0 intercepted'}
          </span>
        </div>

        <div className="bg-slate-900/60 border border-slate-800 rounded-xl p-4">
          <div className="flex items-center justify-between text-xs text-slate-400 mb-1">
            <span>Average Integrity Score</span>
            <ShieldCheck className="h-3.5 w-3.5 text-emerald-400" />
          </div>
          <span className="text-xl font-bold font-mono text-emerald-400">
            {stats ? `${(stats.avg_integrity_score * 100).toFixed(0)}%` : '100%'}
          </span>
          <span className="text-[10px] text-emerald-400/80 block mt-0.5">
            {stats ? `${stats.certified_count} clean executions` : '0 clean'}
          </span>
        </div>

        <div className="bg-slate-900/60 border border-slate-800 rounded-xl p-4">
          <div className="flex items-center justify-between text-xs text-slate-400 mb-1">
            <span>Average Verification Speed</span>
            <Clock className="h-3.5 w-3.5 text-cyan-400" />
          </div>
          <span className="text-xl font-bold font-mono text-cyan-400">
            {stats ? `${stats.avg_latency_ms.toFixed(2)} ms` : '< 1 ms'}
          </span>
          <span className="text-[10px] text-cyan-400/80 block mt-0.5">Deterministic Go AST runtime</span>
        </div>
      </div>

      {/* Main Ledger Table */}
      <div className="bg-slate-900/60 border border-slate-800 rounded-xl overflow-hidden">
        <div className="px-5 py-4 bg-slate-950/80 border-b border-slate-800 flex items-center justify-between">
          <div className="flex items-center space-x-2">
            <History className="h-4 w-4 text-cyan-400" />
            <h3 className="text-sm font-semibold text-slate-200">
              Historical Verification Ledger & Intercept Trail
            </h3>
          </div>
          <button
            onClick={loadData}
            disabled={loading}
            className="flex items-center space-x-1 text-xs text-slate-400 hover:text-slate-200 transition-colors cursor-pointer"
          >
            <RefreshCw className={`h-3 w-3 ${loading ? 'animate-spin' : ''}`} />
            <span>Refresh</span>
          </button>
        </div>

        {events.length === 0 ? (
          <div className="p-8 text-center text-slate-400 text-xs">
            No verification events recorded yet. Run query verifications to populate the audit ledger.
          </div>
        ) : (
          <div className="divide-y divide-slate-800 overflow-x-auto">
            <table className="w-full text-left text-xs">
              <thead className="bg-slate-950 text-slate-400 border-b border-slate-800">
                <tr>
                  <th className="py-2.5 px-4">Voucher</th>
                  <th className="py-2.5 px-4">Timestamp</th>
                  <th className="py-2.5 px-4">Status</th>
                  <th className="py-2.5 px-4">Integrity</th>
                  <th className="py-2.5 px-4">Issues</th>
                  <th className="py-2.5 px-4">Speed</th>
                  <th className="py-2.5 px-4">Query Preview</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-800/80 font-mono text-[11px]">
                {events.map((ev) => (
                  <tr
                    key={ev.id}
                    onClick={() => setSelectedEntry(ev)}
                    className="hover:bg-slate-800/40 cursor-pointer transition-colors"
                  >
                    <td className="py-2 px-4 text-cyan-300 font-semibold">{ev.voucher_id || `log-${ev.id}`}</td>
                    <td className="py-2 px-4 text-slate-400 font-sans">
                      {new Date(ev.timestamp).toLocaleTimeString()}
                    </td>
                    <td className="py-2 px-4 font-sans">
                      <span
                        className={`px-1.5 py-0.5 rounded text-[10px] font-semibold ${
                          ev.status === 'CERTIFIED'
                            ? 'bg-emerald-950 text-emerald-300 border border-emerald-800'
                            : 'bg-rose-950 text-rose-300 border border-rose-800'
                        }`}
                      >
                        {ev.status}
                      </span>
                    </td>
                    <td className="py-2 px-4 text-slate-200">{(ev.integrity_score * 100).toFixed(0)}%</td>
                    <td className="py-2 px-4">
                      {ev.violations_count > 0 ? (
                        <span className="text-rose-400 font-bold">{ev.violations_count} detected</span>
                      ) : (
                        <span className="text-emerald-400 font-medium">0 clean</span>
                      )}
                    </td>
                    <td className="py-2 px-4 text-slate-400">{ev.latency_ms.toFixed(2)} ms</td>
                    <td className="py-2 px-4 text-slate-400 truncate max-w-xs">{ev.candidate_sql}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Selected Entry Detail Modal / Drawer */}
      {selectedEntry && (
        <div className="bg-slate-950 border border-slate-800 rounded-xl p-5 text-xs">
          <div className="flex items-center justify-between mb-3 border-b border-slate-800 pb-2">
            <span className="font-bold text-slate-200">
              Audit Event Detail: {selectedEntry.voucher_id}
            </span>
            <button
              onClick={() => setSelectedEntry(null)}
              className="text-slate-400 hover:text-slate-200 text-xs cursor-pointer"
            >
              Close
            </button>
          </div>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <span className="text-slate-400 font-semibold block mb-1">Candidate SQL:</span>
              <pre className="p-3 rounded bg-slate-900 border border-slate-800 font-mono text-[11px] text-slate-200 overflow-x-auto leading-relaxed">
                {selectedEntry.candidate_sql}
              </pre>
            </div>
            <div>
              <span className="text-slate-400 font-semibold block mb-1">Certified Rewritten SQL:</span>
              <pre className="p-3 rounded bg-slate-900 border border-slate-800 font-mono text-[11px] text-slate-200 overflow-x-auto leading-relaxed">
                {selectedEntry.rewritten_sql || '// No rewrite needed'}
              </pre>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

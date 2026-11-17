import React, { useState } from 'react';
import { GitFork, ShieldCheck, AlertTriangle, ArrowRight, Layers, Database, Sparkles } from 'lucide-react';
import type { VerificationReport } from '../types';

interface FanoutGraphProps {
  report: VerificationReport | null;
}

export const FanoutGraph: React.FC<FanoutGraphProps> = ({ report }) => {
  const [viewMode, setViewMode] = useState<'compare' | 'naive' | 'cte'>('compare');

  const fanoutViolation = report?.suite_result?.all_violations?.find(
    (v) => v.probe_name === 'JoinFanoutGuard'
  );

  return (
    <div className="bg-slate-900/60 border border-slate-800 rounded-xl overflow-hidden">
      <div className="px-5 py-4 bg-slate-950/80 border-b border-slate-800 flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center space-x-2.5">
          <GitFork className="h-4 w-4 text-cyan-400" />
          <h3 className="text-sm font-semibold text-slate-200">
            Visual Cardinality & Join Fan-Out Architecture Flow
          </h3>
        </div>

        <div className="flex items-center space-x-1.5 bg-slate-900 border border-slate-800 rounded-lg p-1 text-xs">
          <button
            onClick={() => setViewMode('compare')}
            className={`px-2.5 py-1 rounded transition-colors ${
              viewMode === 'compare' ? 'bg-cyan-950 text-cyan-300 font-semibold' : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            Side-by-Side Flow
          </button>
          <button
            onClick={() => setViewMode('naive')}
            className={`px-2.5 py-1 rounded transition-colors ${
              viewMode === 'naive' ? 'bg-rose-950 text-rose-300 font-semibold' : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            Naive Multiplication
          </button>
          <button
            onClick={() => setViewMode('cte')}
            className={`px-2.5 py-1 rounded transition-colors ${
              viewMode === 'cte' ? 'bg-emerald-950 text-emerald-300 font-semibold' : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            CTE Protected
          </button>
        </div>
      </div>

      <div className="p-6">
        {fanoutViolation ? (
          <div className="mb-6 p-3.5 rounded-lg bg-rose-950/30 border border-rose-900/60 flex items-start space-x-3 text-xs text-rose-300">
            <AlertTriangle className="h-4 w-4 shrink-0 text-rose-400 mt-0.5" />
            <div>
              <span className="font-bold">{fanoutViolation.title}</span>
              <p className="text-rose-300/80 mt-0.5 leading-relaxed">{fanoutViolation.message}</p>
            </div>
          </div>
        ) : (
          <div className="mb-6 p-3.5 rounded-lg bg-emerald-950/30 border border-emerald-900/60 flex items-start space-x-3 text-xs text-emerald-300">
            <ShieldCheck className="h-4 w-4 shrink-0 text-emerald-400 mt-0.5" />
            <div>
              <span className="font-bold">No 1:N Cartesian join explosion detected in current query.</span>
              <p className="text-emerald-300/80 mt-0.5">
                The diagram below demonstrates how queryproof guards against fanout multiplication.
              </p>
            </div>
          </div>
        )}

        {/* Visual Flow Canvas */}
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          {/* Panel A: Naive Evaluation */}
          {(viewMode === 'compare' || viewMode === 'naive') && (
            <div className="p-4 rounded-xl bg-slate-950/60 border border-rose-900/50 relative overflow-hidden">
              <div className="flex items-center justify-between mb-4">
                <span className="text-xs font-bold text-rose-400 flex items-center">
                  <AlertTriangle className="h-3.5 w-3.5 mr-1" />
                  Naive Agent Text-to-SQL Join (Unprotected)
                </span>
                <span className="text-[10px] font-mono px-2 py-0.5 rounded bg-rose-950 text-rose-300 border border-rose-800">
                  Cartesian Risk: High
                </span>
              </div>

              {/* Node diagram */}
              <div className="flex flex-col items-center space-y-4 py-2 text-xs font-mono">
                <div className="w-full flex items-center justify-between bg-slate-900 border border-slate-800 rounded-lg p-3">
                  <div className="flex items-center space-x-2">
                    <Database className="h-4 w-4 text-cyan-400" />
                    <span className="text-slate-200 font-bold">customers (Parent)</span>
                  </div>
                  <span className="text-slate-400 text-[11px]">1 customer row (MRR = $5,000)</span>
                </div>

                <div className="flex flex-col items-center text-rose-400 font-sans text-xs">
                  <span className="text-[10px] bg-rose-950/80 px-2 py-0.5 rounded border border-rose-800/80 font-bold">
                    JOIN ON customer_id (1:N Fanout ×20.0)
                  </span>
                  <div className="h-4 w-0.5 bg-rose-500/60 my-1"></div>
                  <ArrowRight className="h-3 w-3 rotate-90" />
                </div>

                <div className="w-full flex items-center justify-between bg-slate-900 border border-slate-800 rounded-lg p-3">
                  <div className="flex items-center space-x-2">
                    <Database className="h-4 w-4 text-indigo-400" />
                    <span className="text-slate-200 font-bold">orders (Child)</span>
                  </div>
                  <span className="text-slate-400 text-[11px]">20 matched order rows</span>
                </div>

                <div className="w-full p-3 rounded-lg bg-rose-950/40 border border-rose-800 text-center font-sans">
                  <span className="text-[11px] text-rose-300 font-semibold block">
                    Calculated SUM(c.mrr) = 20 × $5,000 = $100,000
                  </span>
                  <span className="text-[10px] text-rose-400 font-mono">
                    Silent 20.0x Over-Reporting of Company Revenue!
                  </span>
                </div>
              </div>
            </div>
          )}

          {/* Panel B: Certified CTE Rewrite */}
          {(viewMode === 'compare' || viewMode === 'cte') && (
            <div className="p-4 rounded-xl bg-slate-950/60 border border-emerald-900/50 relative overflow-hidden">
              <div className="flex items-center justify-between mb-4">
                <span className="text-xs font-bold text-emerald-400 flex items-center">
                  <Sparkles className="h-3.5 w-3.5 mr-1" />
                  queryproof Pre-Aggregated CTE (Certified)
                </span>
                <span className="text-[10px] font-mono px-2 py-0.5 rounded bg-emerald-950 text-emerald-300 border border-emerald-800">
                  Integrity: 100%
                </span>
              </div>

              {/* Node diagram */}
              <div className="flex flex-col items-center space-y-3 py-2 text-xs font-mono">
                <div className="w-full p-2.5 rounded-lg bg-cyan-950/40 border border-cyan-800/80">
                  <div className="flex items-center justify-between text-[11px] text-cyan-300">
                    <span className="font-bold flex items-center">
                      <Layers className="h-3.5 w-3.5 mr-1" />
                      WITH orders_agg AS (...)
                    </span>
                    <span>Pre-aggregates orders to 1 row per customer</span>
                  </div>
                </div>

                <div className="w-full flex items-center justify-between bg-slate-900 border border-slate-800 rounded-lg p-3">
                  <div className="flex items-center space-x-2">
                    <Database className="h-4 w-4 text-cyan-400" />
                    <span className="text-slate-200 font-bold">customers (Parent)</span>
                  </div>
                  <span className="text-slate-400 text-[11px]">1 customer row (MRR = $5,000)</span>
                </div>

                <div className="flex flex-col items-center text-emerald-400 font-sans text-xs">
                  <span className="text-[10px] bg-emerald-950/80 px-2 py-0.5 rounded border border-emerald-800/80 font-bold">
                    JOIN ON customer_id (1:1 Ratio Isolated)
                  </span>
                  <div className="h-4 w-0.5 bg-emerald-500/60 my-1"></div>
                  <ArrowRight className="h-3 w-3 rotate-90" />
                </div>

                <div className="w-full flex items-center justify-between bg-slate-900 border border-slate-800 rounded-lg p-3">
                  <div className="flex items-center space-x-2">
                    <Database className="h-4 w-4 text-emerald-400" />
                    <span className="text-slate-200 font-bold">orders_agg (Isolated CTE)</span>
                  </div>
                  <span className="text-slate-400 text-[11px]">1 aggregated row</span>
                </div>

                <div className="w-full p-3 rounded-lg bg-emerald-950/40 border border-emerald-800 text-center font-sans">
                  <span className="text-[11px] text-emerald-300 font-semibold block">
                    Calculated SUM(c.mrr) = 1 × $5,000 = $5,000
                  </span>
                  <span className="text-[10px] text-emerald-400 font-mono">
                    Accurate Ground-Truth Financial Revenue
                  </span>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

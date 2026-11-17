import React, { useState } from 'react';
import type { SuiteResult, Violation } from '../types';
import { CheckCircle2, AlertOctagon, AlertTriangle, Info, ChevronDown, ChevronUp, Wrench, ShieldAlert } from 'lucide-react';

interface ProbeMatrixProps {
  suiteResult: SuiteResult | null;
}

export const ProbeMatrix: React.FC<ProbeMatrixProps> = ({ suiteResult }) => {
  const [expandedProofs, setExpandedProofs] = useState<Record<string, boolean>>({});

  if (!suiteResult) {
    return (
      <div className="bg-slate-900/60 border border-slate-800 rounded-xl p-8 text-center text-slate-400 text-sm">
        Run verification on a candidate query to view falsification probe findings.
      </div>
    );
  }

  const toggleProof = (idx: string) => {
    setExpandedProofs((prev) => ({ ...prev, [idx]: !prev[idx] }));
  };

  const getSeverityBadge = (severity: string) => {
    switch (severity) {
      case 'CRITICAL':
        return (
          <span className="px-2 py-0.5 rounded text-[10px] font-bold bg-rose-950/80 text-rose-300 border border-rose-800 flex items-center space-x-1">
            <AlertOctagon className="h-3 w-3 mr-0.5" />
            CRITICAL
          </span>
        );
      case 'HIGH':
        return (
          <span className="px-2 py-0.5 rounded text-[10px] font-bold bg-amber-950/80 text-amber-300 border border-amber-800 flex items-center space-x-1">
            <AlertTriangle className="h-3 w-3 mr-0.5" />
            HIGH
          </span>
        );
      case 'MEDIUM':
        return (
          <span className="px-2 py-0.5 rounded text-[10px] font-bold bg-yellow-950/80 text-yellow-300 border border-yellow-800 flex items-center space-x-1">
            <Info className="h-3 w-3 mr-0.5" />
            MEDIUM
          </span>
        );
      default:
        return (
          <span className="px-2 py-0.5 rounded text-[10px] font-medium bg-slate-800 text-slate-300">
            {severity}
          </span>
        );
    }
  };

  return (
    <div className="bg-slate-900/60 border border-slate-800 rounded-xl overflow-hidden">
      {/* Integrity Summary Header */}
      <div className="px-5 py-4 bg-slate-950/80 border-b border-slate-800 flex flex-wrap items-center justify-between gap-4">
        <div>
          <div className="flex items-center space-x-2">
            <ShieldAlert className="h-4 w-4 text-cyan-400" />
            <h3 className="text-sm font-semibold text-slate-200">
              Deterministic Falsification Matrix
            </h3>
          </div>
          <p className="text-xs text-slate-400 mt-0.5">
            5 automated probes mathematically evaluating AST join safety, null handling, and enterprise metrics
          </p>
        </div>

        <div className="flex items-center space-x-3">
          <div className="flex items-center space-x-2 px-3 py-1.5 rounded-lg bg-slate-900 border border-slate-800">
            <span className="text-xs text-slate-400">Integrity Score:</span>
            <span
              className={`text-sm font-mono font-bold ${
                suiteResult.passed
                  ? 'text-emerald-400'
                  : suiteResult.critical_count > 0
                  ? 'text-rose-400'
                  : 'text-amber-400'
              }`}
            >
              {(suiteResult.overall_score * 100).toFixed(0)}%
            </span>
          </div>

          <div className="flex items-center space-x-1.5 text-xs">
            {suiteResult.critical_count > 0 && (
              <span className="px-2 py-1 rounded bg-rose-950 text-rose-300 font-semibold border border-rose-800">
                {suiteResult.critical_count} Critical
              </span>
            )}
            {suiteResult.high_count > 0 && (
              <span className="px-2 py-1 rounded bg-amber-950 text-amber-300 font-semibold border border-amber-800">
                {suiteResult.high_count} High
              </span>
            )}
            {suiteResult.passed && (
              <span className="px-2.5 py-1 rounded bg-emerald-950 text-emerald-300 font-semibold border border-emerald-800 flex items-center space-x-1">
                <CheckCircle2 className="h-3.5 w-3.5 mr-1" />
                All Invariants Verified
              </span>
            )}
          </div>
        </div>
      </div>

      {/* Probes List */}
      <div className="divide-y divide-slate-800/80">
        {suiteResult.results.map((pr, pIdx) => {
          return (
            <div key={pIdx} className="p-4 bg-slate-900/40">
              <div className="flex items-start justify-between">
                <div className="flex items-start space-x-3">
                  {pr.passed ? (
                    <div className="mt-0.5 p-1 rounded-full bg-emerald-950 text-emerald-400 border border-emerald-800/60">
                      <CheckCircle2 className="h-4 w-4" />
                    </div>
                  ) : (
                    <div className="mt-0.5 p-1 rounded-full bg-rose-950 text-rose-400 border border-rose-800/60">
                      <AlertOctagon className="h-4 w-4" />
                    </div>
                  )}
                  <div>
                    <div className="flex items-center space-x-2">
                      <h4 className="text-xs font-semibold text-slate-200">{pr.probe_name}</h4>
                      <span
                        className={`text-[10px] font-medium px-1.5 py-0.5 rounded ${
                          pr.passed
                            ? 'bg-emerald-950/70 text-emerald-300 border border-emerald-800/60'
                            : 'bg-rose-950/70 text-rose-300 border border-rose-800/60'
                        }`}
                      >
                        {pr.passed ? 'PASSED' : 'VIOLATION DETECTED'}
                      </span>
                    </div>
                    <p className="text-xs text-slate-400 mt-0.5">{pr.description}</p>
                  </div>
                </div>

                <div className="text-right">
                  <span className="text-xs font-mono text-slate-300">
                    Score: {(pr.score * 100).toFixed(0)}%
                  </span>
                </div>
              </div>

              {/* Violations under this probe */}
              {pr.violations.length > 0 && (
                <div className="mt-3.5 space-y-2.5 pl-8">
                  {pr.violations.map((v: Violation, vIdx: number) => {
                    const proofKey = `${pIdx}-${vIdx}`;
                    const isExpanded = !!expandedProofs[proofKey];

                    return (
                      <div
                        key={vIdx}
                        className="bg-slate-950/70 border border-slate-800/90 rounded-lg p-3 text-xs"
                      >
                        <div className="flex items-center justify-between mb-1.5">
                          <div className="flex items-center space-x-2">
                            {getSeverityBadge(v.severity)}
                            <span className="font-semibold text-slate-200">{v.title}</span>
                          </div>
                          {v.auto_fix_available && (
                            <span className="text-[10px] text-cyan-400 flex items-center bg-cyan-950/50 px-1.5 py-0.5 rounded border border-cyan-800/40">
                              <Wrench className="h-2.5 w-2.5 mr-1" />
                              Auto-Fix Supported
                            </span>
                          )}
                        </div>

                        <p className="text-slate-300 text-[11px] leading-relaxed mb-2">{v.message}</p>

                        <div className="bg-slate-900/90 rounded p-2 border border-slate-800 text-[11px] font-mono text-slate-300 mb-2">
                          <span className="text-slate-400">Offending Expression: </span>
                          <span className="text-rose-400 font-bold">{v.offending_node}</span>
                        </div>

                        <div className="flex items-center justify-between pt-1">
                          <div className="text-[11px] text-cyan-300">
                            <span className="text-slate-400">Remediation: </span>
                            {v.remediation}
                          </div>

                          {v.mathematical_proof && (
                            <button
                              onClick={() => toggleProof(proofKey)}
                              className="text-[11px] text-slate-400 hover:text-slate-200 flex items-center space-x-1 cursor-pointer"
                            >
                              <span>{isExpanded ? 'Hide Invariant Proof' : 'View Proof'}</span>
                              {isExpanded ? <ChevronUp className="h-3 w-3" /> : <ChevronDown className="h-3 w-3" />}
                            </button>
                          )}
                        </div>

                        {/* Collapsible Mathematical Proof */}
                        {isExpanded && v.mathematical_proof && (
                          <div className="mt-2.5 p-2.5 rounded bg-slate-900 border border-cyan-950/60 font-mono text-[10px] text-cyan-300 leading-relaxed">
                            <span className="text-slate-400 block mb-1 font-sans font-semibold uppercase tracking-wider text-[9px]">
                              Mathematical Proof & Formal Verification Invariant:
                            </span>
                            {v.mathematical_proof}
                          </div>
                        )}
                      </div>
                    );
                  })}
                </div>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
};

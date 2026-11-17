import React, { useState } from 'react';
import { Play, Copy, Check, Sparkles, FileText, ArrowRight } from 'lucide-react';
import type { MutationRecord, VerificationReport } from '../types';

interface SqlDiffEditorProps {
  query: string;
  onChangeQuery: (newQuery: string) => void;
  onVerify: () => void;
  report: VerificationReport | null;
  loading: boolean;
}

export const SqlDiffEditor: React.FC<SqlDiffEditorProps> = ({
  query,
  onChangeQuery,
  onVerify,
  report,
  loading,
}) => {
  const [copiedOriginal, setCopiedOriginal] = useState(false);
  const [copiedRewritten, setCopiedRewritten] = useState(false);

  const copyText = (text: string, isOriginal: boolean) => {
    navigator.clipboard.writeText(text);
    if (isOriginal) {
      setCopiedOriginal(true);
      setTimeout(() => setCopiedOriginal(false), 2000);
    } else {
      setCopiedRewritten(true);
      setTimeout(() => setCopiedRewritten(false), 2000);
    }
  };

  return (
    <div className="bg-slate-900/60 border border-slate-800 rounded-xl overflow-hidden">
      {/* Action Header */}
      <div className="flex items-center justify-between px-4 py-2.5 bg-slate-950/60 border-b border-slate-800">
        <div className="flex items-center space-x-2">
          <FileText className="h-4 w-4 text-cyan-400" />
          <span className="text-xs font-semibold text-slate-200">
            AST Verification & Rewriter Workbench
          </span>
          {report && (
            <span className="text-[11px] font-mono px-2 py-0.5 rounded bg-slate-800 text-cyan-400 border border-slate-700/60">
              Verified in {report.verification_ms.toFixed(2)} ms
            </span>
          )}
        </div>

        <button
          onClick={onVerify}
          disabled={loading}
          className="flex items-center space-x-1.5 px-4 py-1.5 rounded-lg bg-gradient-to-r from-cyan-500 to-blue-600 hover:from-cyan-400 hover:to-blue-500 text-white font-medium text-xs shadow-md shadow-cyan-900/30 transition-all disabled:opacity-50 cursor-pointer"
        >
          <Play className="h-3.5 w-3.5 fill-current" />
          <span>{loading ? 'Evaluating Probes...' : 'Verify & Falsify Query'}</span>
        </button>
      </div>

      {/* Side-by-side SQL Columns */}
      <div className="grid grid-cols-1 lg:grid-cols-2 divide-y lg:divide-y-0 lg:divide-x divide-slate-800">
        {/* Candidate SQL (Left) */}
        <div className="flex flex-col bg-slate-950/30">
          <div className="flex items-center justify-between px-3.5 py-2 bg-slate-900/40 border-b border-slate-800/60 text-xs">
            <div className="flex items-center space-x-2">
              <span className="w-2 h-2 rounded-full bg-rose-500"></span>
              <span className="font-semibold text-slate-300">Candidate LLM Query</span>
              <span className="text-[10px] text-slate-400">(Agent Generated)</span>
            </div>
            <button
              onClick={() => copyText(query, true)}
              className="text-slate-400 hover:text-slate-200 flex items-center space-x-1 text-[11px]"
            >
              {copiedOriginal ? <Check className="h-3 w-3 text-emerald-400" /> : <Copy className="h-3 w-3" />}
              <span>{copiedOriginal ? 'Copied' : 'Copy'}</span>
            </button>
          </div>
          <div className="p-3">
            <textarea
              value={query}
              onChange={(e) => onChangeQuery(e.target.value)}
              rows={11}
              className="w-full bg-slate-900/80 border border-slate-800/80 rounded-lg p-3 font-mono text-xs text-slate-200 focus:outline-none focus:ring-1 focus:ring-cyan-500/50 resize-none leading-relaxed"
              placeholder="Paste candidate Text-to-SQL statement here..."
            />
          </div>
        </div>

        {/* Certified Rewritten SQL (Right) */}
        <div className="flex flex-col bg-slate-950/30">
          <div className="flex items-center justify-between px-3.5 py-2 bg-slate-900/40 border-b border-slate-800/60 text-xs">
            <div className="flex items-center space-x-2">
              <span className="w-2 h-2 rounded-full bg-emerald-500"></span>
              <span className="font-semibold text-slate-300">Certified Rewritten Query</span>
              {report?.is_clean ? (
                <span className="text-[10px] text-emerald-400 font-medium bg-emerald-950/60 px-1.5 py-0.5 rounded border border-emerald-800/50">
                  Clean / No Mutation Needed
                </span>
              ) : report?.rewritten_sql ? (
                <span className="text-[10px] text-cyan-400 font-medium bg-cyan-950/60 px-1.5 py-0.5 rounded border border-cyan-800/50 flex items-center">
                  <Sparkles className="h-2.5 w-2.5 mr-1" />
                  Auto-Remediated
                </span>
              ) : null}
            </div>
            {report?.rewritten_sql && (
              <button
                onClick={() => copyText(report.rewritten_sql || '', false)}
                className="text-slate-400 hover:text-slate-200 flex items-center space-x-1 text-[11px]"
              >
                {copiedRewritten ? <Check className="h-3 w-3 text-emerald-400" /> : <Copy className="h-3 w-3" />}
                <span>{copiedRewritten ? 'Copied' : 'Copy'}</span>
              </button>
            )}
          </div>
          <div className="p-3 flex-1 flex flex-col">
            <pre className="w-full flex-1 bg-slate-900/80 border border-slate-800/80 rounded-lg p-3 font-mono text-xs text-slate-200 overflow-x-auto leading-relaxed min-h-[180px]">
              {report?.is_clean
                ? report.original_sql
                : report?.rewritten_sql || '// Run verification to compute the mathematically safe rewritten AST query'}
            </pre>
          </div>
        </div>
      </div>

      {/* Applied Mutations List */}
      {report?.mutations && report.mutations.length > 0 && (
        <div className="px-4 py-3 bg-slate-950/80 border-t border-slate-800">
          <div className="flex items-center space-x-2 mb-2">
            <Sparkles className="h-3.5 w-3.5 text-cyan-400" />
            <span className="text-xs font-semibold text-slate-300">
              Deterministic AST Mutations Applied ({report.mutations.length})
            </span>
          </div>
          <div className="space-y-1.5">
            {report.mutations.map((m: MutationRecord, idx: number) => (
              <div
                key={idx}
                className="text-xs bg-slate-900/90 border border-slate-800 rounded px-2.5 py-1.5 flex items-start space-x-2 text-slate-300"
              >
                <span className="px-1.5 py-0.5 rounded text-[10px] font-mono bg-cyan-950 text-cyan-300 border border-cyan-800/50">
                  {m.probe_name}
                </span>
                <div className="flex-1">
                  <p className="font-medium text-slate-200">{m.description}</p>
                  <div className="flex items-center space-x-2 mt-1 text-[11px] font-mono text-slate-400">
                    <span className="text-rose-400 bg-rose-950/40 px-1 rounded">{m.before_expr}</span>
                    <ArrowRight className="h-3 w-3 text-slate-500" />
                    <span className="text-emerald-400 bg-emerald-950/40 px-1 rounded">{m.after_expr}</span>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};

import React from 'react';
import type { Metric } from '../types';
import { BookOpen, CheckCircle, Tag } from 'lucide-react';

interface MetricDictionaryProps {
  metrics: Metric[];
}

export const MetricDictionary: React.FC<MetricDictionaryProps> = ({ metrics }) => {
  return (
    <div className="bg-slate-900/60 border border-slate-800 rounded-xl overflow-hidden">
      <div className="px-5 py-4 bg-slate-950/80 border-b border-slate-800 flex items-center justify-between">
        <div className="flex items-center space-x-2">
          <BookOpen className="h-4 w-4 text-cyan-400" />
          <h3 className="text-sm font-semibold text-slate-200">
            Enterprise Semantic Metric Registry (metrics.yaml)
          </h3>
        </div>
        <span className="text-xs text-slate-400">
          Authoritative corporate KPI definitions enforced by Probe 3
        </span>
      </div>

      <div className="p-4 grid grid-cols-1 md:grid-cols-2 gap-3.5">
        {metrics.map((m, idx) => (
          <div
            key={idx}
            className="bg-slate-950/70 border border-slate-800/90 rounded-lg p-3.5 flex flex-col justify-between text-xs"
          >
            <div>
              <div className="flex items-center justify-between mb-1.5">
                <span className="font-bold text-sm text-cyan-300">{m.name}</span>
                {m.certified && (
                  <span className="px-2 py-0.5 rounded text-[10px] font-semibold bg-emerald-950 text-emerald-300 border border-emerald-800 flex items-center">
                    <CheckCircle className="h-3 w-3 mr-1" />
                    Boardroom Certified
                  </span>
                )}
              </div>
              <p className="text-slate-400 text-xs mb-3">{m.description}</p>

              {/* Certified Formula */}
              <div className="bg-slate-900 rounded p-2 border border-slate-800 font-mono text-[11px] text-slate-200 mb-2.5">
                <span className="text-slate-400 block text-[9px] uppercase font-sans tracking-wider mb-0.5">
                  Certified Formula:
                </span>
                {m.certified_formula}
              </div>

              {/* Source Tables & Filters */}
              {m.required_filters && m.required_filters.length > 0 && (
                <div className="mb-2">
                  <span className="text-[10px] text-slate-400 uppercase tracking-wider block mb-1">
                    Required Filters:
                  </span>
                  <div className="flex flex-wrap gap-1">
                    {m.required_filters.map((rf, rIdx) => (
                      <span
                        key={rIdx}
                        className="px-1.5 py-0.5 rounded text-[10px] font-mono bg-rose-950/60 text-rose-300 border border-rose-900/60"
                      >
                        {rf}
                      </span>
                    ))}
                  </div>
                </div>
              )}

              {/* Allowed Dimensions */}
              {m.allowed_dimensions && m.allowed_dimensions.length > 0 && (
                <div>
                  <span className="text-[10px] text-slate-400 uppercase tracking-wider block mb-1">
                    Allowed Dimensions:
                  </span>
                  <div className="flex flex-wrap gap-1">
                    {m.allowed_dimensions.map((dim, dIdx) => (
                      <span
                        key={dIdx}
                        className="px-1.5 py-0.5 rounded text-[10px] font-mono bg-slate-900 text-slate-300 border border-slate-800 flex items-center"
                      >
                        <Tag className="h-2.5 w-2.5 mr-1 text-slate-400" />
                        {dim}
                      </span>
                    ))}
                  </div>
                </div>
              )}
            </div>

            {m.owner && (
              <div className="mt-3 pt-2 border-t border-slate-800/80 text-[10px] text-slate-400 flex items-center justify-between">
                <span>Owner: {m.owner}</span>
                <span>Tables: {m.source_tables?.join(', ')}</span>
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  );
};

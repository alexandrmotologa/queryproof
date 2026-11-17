import React from 'react';
import { ShieldCheck, Cpu, HardDrive, Database, ExternalLink } from 'lucide-react';
import type { Dialect, HealthInfo } from '../types';

interface HeaderProps {
  dialect: Dialect;
  onDialectChange: (dialect: Dialect) => void;
  health: HealthInfo | null;
}

export const Header: React.FC<HeaderProps> = ({ dialect, onDialectChange, health }) => {
  return (
    <header className="border-b border-slate-800/80 bg-slate-950/80 backdrop-blur-md sticky top-0 z-50 px-6 py-3.5">
      <div className="max-w-7xl mx-auto flex items-center justify-between">
        <div className="flex items-center space-x-3.5">
          <div className="h-9 w-9 rounded-lg bg-gradient-to-tr from-cyan-500 to-blue-600 flex items-center justify-center shadow-lg shadow-cyan-500/20 ring-1 ring-cyan-400/30">
            <ShieldCheck className="h-5 w-5 text-white stroke-[2.5]" />
          </div>
          <div>
            <div className="flex items-center space-x-2">
              <span className="font-bold text-lg tracking-tight text-white">queryproof</span>
              <span className="px-1.5 py-0.5 text-[10px] font-semibold bg-cyan-950 text-cyan-300 rounded border border-cyan-800/60 uppercase tracking-wider">
                v1.0.0
              </span>
            </div>
            <p className="text-xs text-slate-400 font-medium">
              Deterministic Text-to-SQL Verification and Falsification Engine
            </p>
          </div>
        </div>

        <div className="flex items-center space-x-4">
          {/* Engine Memory & Footprint Meter */}
          <div className="hidden md:flex items-center space-x-3 px-3 py-1.5 rounded-md bg-slate-900/90 border border-slate-800 text-xs text-slate-400 font-mono">
            <div className="flex items-center space-x-1.5 text-slate-300">
              <Cpu className="h-3.5 w-3.5 text-emerald-400" />
              <span>Go Engine</span>
            </div>
            <span className="text-slate-600">|</span>
            <div className="flex items-center space-x-1.5">
              <HardDrive className="h-3.5 w-3.5 text-cyan-400" />
              <span className="text-slate-200 font-semibold">
                {health ? `${health.allocated_mb.toFixed(2)} MB` : '< 2.5 MB'}
              </span>
              <span className="text-[10px] text-emerald-400 font-normal">(&lt; 20MB budget)</span>
            </div>
          </div>

          {/* Dialect Selector */}
          <div className="flex items-center space-x-2 bg-slate-900 rounded-md border border-slate-800 p-1">
            <Database className="h-3.5 w-3.5 text-slate-400 ml-2" />
            <select
              value={dialect}
              onChange={(e) => onDialectChange(e.target.value as Dialect)}
              className="bg-transparent text-xs font-medium text-slate-200 focus:outline-none pr-3 cursor-pointer capitalize"
            >
              <option value="postgres" className="bg-slate-900 text-slate-200">PostgreSQL</option>
              <option value="snowflake" className="bg-slate-900 text-slate-200">Snowflake</option>
              <option value="bigquery" className="bg-slate-900 text-slate-200">BigQuery</option>
            </select>
          </div>

          <a
            href="https://github.com/alexandrmotologa/queryproof"
            target="_blank"
            rel="noreferrer"
            className="flex items-center space-x-1 text-xs text-slate-400 hover:text-slate-200 transition-colors"
          >
            <span>GitHub</span>
            <ExternalLink className="h-3 w-3" />
          </a>
        </div>
      </div>
    </header>
  );
};

import React from 'react';
import type { Scenario } from '../types';
import { Sparkles, AlertTriangle } from 'lucide-react';

interface ScenarioSelectorProps {
  scenarios: Scenario[];
  selectedId: string;
  onSelect: (scenario: Scenario) => void;
}

export const ScenarioSelector: React.FC<ScenarioSelectorProps> = ({
  scenarios,
  selectedId,
  onSelect,
}) => {
  return (
    <div className="bg-slate-900/60 border border-slate-800 rounded-xl p-4">
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center space-x-2">
          <Sparkles className="h-4 w-4 text-cyan-400" />
          <h2 className="text-sm font-semibold text-slate-200">
            Enterprise Falsification Scenarios
          </h2>
        </div>
        <span className="text-xs text-slate-400">
          Select an agent-generated candidate query to stress-test
        </span>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-2.5">
        {scenarios.map((sc) => {
          const isSelected = sc.id === selectedId;
          return (
            <button
              key={sc.id}
              onClick={() => onSelect(sc)}
              className={`text-left p-3 rounded-lg border transition-all text-xs flex flex-col justify-between ${
                isSelected
                  ? 'bg-cyan-950/40 border-cyan-500/70 shadow-md shadow-cyan-950/50 ring-1 ring-cyan-500/30'
                  : 'bg-slate-900 border-slate-800/80 hover:border-slate-700 hover:bg-slate-800/60 text-slate-300'
              }`}
            >
              <div>
                <div className="flex items-center justify-between mb-1.5">
                  <span className={`font-semibold ${isSelected ? 'text-cyan-300' : 'text-slate-200'}`}>
                    {sc.title}
                  </span>
                </div>
                <p className="text-[11px] text-slate-400 line-clamp-2 leading-relaxed">
                  {sc.description}
                </p>
              </div>
              <div className="mt-2.5 flex items-center justify-between text-[10px]">
                <span className="px-1.5 py-0.5 rounded bg-slate-800 text-slate-300 font-mono">
                  {sc.target_probe}
                </span>
                <span className="flex items-center text-rose-400 font-medium">
                  <AlertTriangle className="h-2.5 w-2.5 mr-1" />
                  Trap
                </span>
              </div>
            </button>
          );
        })}
      </div>
    </div>
  );
};

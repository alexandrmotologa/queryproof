import React, { useState } from 'react';
import type { Catalog, Table } from '../types';
import { Database, Key, Shield, Layers, GitFork } from 'lucide-react';

interface CatalogViewerProps {
  catalog: Catalog | null;
}

export const CatalogViewer: React.FC<CatalogViewerProps> = ({ catalog }) => {
  const [selectedTable, setSelectedTable] = useState<string>(
    catalog?.tables[0]?.name || ''
  );

  if (!catalog) {
    return (
      <div className="bg-slate-900/60 border border-slate-800 rounded-xl p-8 text-center text-slate-400 text-xs">
        Loading database catalog...
      </div>
    );
  }

  const currentTable: Table | undefined = catalog.tables.find(
    (t) => t.name === selectedTable
  ) || catalog.tables[0];

  return (
    <div className="bg-slate-900/60 border border-slate-800 rounded-xl overflow-hidden">
      <div className="px-5 py-4 bg-slate-950/80 border-b border-slate-800 flex items-center justify-between">
        <div className="flex items-center space-x-2">
          <Database className="h-4 w-4 text-cyan-400" />
          <h3 className="text-sm font-semibold text-slate-200">
            Database Catalog Schema & Foreign Key Graph
          </h3>
        </div>
        <span className="text-xs text-slate-400 font-mono">
          Dialect: {catalog.dialect} | {catalog.tables.length} Tables Registered
        </span>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-4 divide-y md:divide-y-0 md:divide-x divide-slate-800">
        {/* Table Selector List */}
        <div className="p-3 space-y-1 bg-slate-950/30">
          <span className="text-[10px] font-semibold text-slate-400 uppercase tracking-wider block px-2 mb-2">
            Catalog Tables
          </span>
          {catalog.tables.map((t) => {
            const isSelected = (currentTable?.name === t.name);
            return (
              <button
                key={t.name}
                onClick={() => setSelectedTable(t.name)}
                className={`w-full text-left px-3 py-2 rounded text-xs transition-colors flex items-center justify-between cursor-pointer ${
                  isSelected
                    ? 'bg-cyan-950/50 text-cyan-300 font-semibold border border-cyan-800/60'
                    : 'text-slate-300 hover:bg-slate-900 hover:text-white'
                }`}
              >
                <span>{t.name}</span>
                <span className="text-[10px] font-mono text-slate-400">
                  {t.columns.length} cols
                </span>
              </button>
            );
          })}
        </div>

        {/* Selected Table Details */}
        <div className="md:col-span-3 p-4 bg-slate-900/40">
          {currentTable && (
            <div>
              <div className="flex items-center justify-between mb-3">
                <div className="flex items-center space-x-2">
                  <h4 className="text-sm font-bold text-slate-100">{currentTable.name}</h4>
                  {currentTable.estimated_row_count && (
                    <span className="px-2 py-0.5 rounded text-[10px] bg-slate-800 text-slate-300 font-mono">
                      ~{currentTable.estimated_row_count.toLocaleString()} rows
                    </span>
                  )}
                </div>

                <div className="flex items-center space-x-2 text-[11px]">
                  {currentTable.tenancy_filter && (
                    <span className="px-2 py-0.5 rounded bg-cyan-950 text-cyan-300 border border-cyan-800 flex items-center">
                      <Shield className="h-3 w-3 mr-1" />
                      Tenancy: {currentTable.tenancy_filter}
                    </span>
                  )}
                  {currentTable.soft_delete_filter && (
                    <span className="px-2 py-0.5 rounded bg-indigo-950 text-indigo-300 border border-indigo-800 flex items-center">
                      <Layers className="h-3 w-3 mr-1" />
                      Soft-Delete: {currentTable.soft_delete_filter}
                    </span>
                  )}
                  {currentTable.partition_column && (
                    <span className="px-2 py-0.5 rounded bg-amber-950 text-amber-300 border border-amber-800 flex items-center">
                      Partition: {currentTable.partition_column}
                    </span>
                  )}
                </div>
              </div>

              {/* Columns Table */}
              <div className="border border-slate-800 rounded-lg overflow-hidden">
                <table className="w-full text-left text-xs">
                  <thead className="bg-slate-950 text-slate-400 border-b border-slate-800 font-medium">
                    <tr>
                      <th className="py-2 px-3">Column</th>
                      <th className="py-2 px-3">Type</th>
                      <th className="py-2 px-3">Nullable</th>
                      <th className="py-2 px-3">Constraints</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-800/80 font-mono">
                    {currentTable.columns.map((c) => (
                      <tr key={c.name} className="hover:bg-slate-900/60">
                        <td className="py-2 px-3 text-slate-200 font-semibold">{c.name}</td>
                        <td className="py-2 px-3 text-cyan-400">{c.data_type}</td>
                        <td className="py-2 px-3">
                          {c.nullable ? (
                            <span className="text-amber-400">YES (Risk)</span>
                          ) : (
                            <span className="text-slate-400">NO</span>
                          )}
                        </td>
                        <td className="py-2 px-3">
                          <div className="flex flex-wrap gap-1 font-sans">
                            {c.is_primary_key && (
                              <span className="px-1.5 py-0.2 rounded text-[10px] bg-emerald-950 text-emerald-300 border border-emerald-800 flex items-center">
                                <Key className="h-2.5 w-2.5 mr-0.5" />
                                PK
                              </span>
                            )}
                            {c.is_foreign_key && (
                              <span className="px-1.5 py-0.2 rounded text-[10px] bg-blue-950 text-blue-300 border border-blue-800 flex items-center">
                                <GitFork className="h-2.5 w-2.5 mr-0.5" />
                                FK → {c.ref_table}.{c.ref_column}
                              </span>
                            )}
                            {c.is_tenancy_key && (
                              <span className="px-1.5 py-0.2 rounded text-[10px] bg-cyan-950 text-cyan-300 border border-cyan-800">
                                TenantKey
                              </span>
                            )}
                            {c.is_partition_key && (
                              <span className="px-1.5 py-0.2 rounded text-[10px] bg-amber-950 text-amber-300 border border-amber-800">
                                PartitionKey
                              </span>
                            )}
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

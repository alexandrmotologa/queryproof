import React, { useState, useEffect } from 'react';
import { Header } from '../components/Header';
import { ScenarioSelector } from '../components/ScenarioSelector';
import { SqlDiffEditor } from '../components/SqlDiffEditor';
import { ProbeMatrix } from '../components/ProbeMatrix';
import { ProofVoucherCard } from '../components/ProofVoucher';
import { MetricDictionary } from '../components/MetricDictionary';
import { CatalogViewer } from '../components/CatalogViewer';
import { FanoutGraph } from '../components/FanoutGraph';
import { AuditLedgerView } from '../components/AuditLedger';
import type { Catalog, Dialect, HealthInfo, Metric, Scenario, VerificationReport } from '../types';
import { getCatalog, getHealth, getMetrics, getScenarios, verifyQuery } from '../api/client';
import { ShieldAlert, FileCode2, BookOpen, Database, GitFork, History } from 'lucide-react';

export const Dashboard: React.FC = () => {
  const [dialect, setDialect] = useState<Dialect>('postgres');
  const [health, setHealth] = useState<HealthInfo | null>(null);
  const [scenarios, setScenarios] = useState<Scenario[]>([]);
  const [selectedScenarioId, setSelectedScenarioId] = useState<string>('');
  const [query, setQuery] = useState<string>('');
  const [report, setReport] = useState<VerificationReport | null>(null);
  const [catalog, setCatalog] = useState<Catalog | null>(null);
  const [metricsList, setMetricsList] = useState<Metric[]>([]);
  const [loading, setLoading] = useState<boolean>(false);
  const [activeTab, setActiveTab] = useState<'probes' | 'fanout' | 'voucher' | 'ledger' | 'metrics' | 'catalog'>('probes');

  // Initial load
  useEffect(() => {
    loadInitialData();
  }, []);

  const loadInitialData = async () => {
    try {
      const [hData, scData, catData, metData] = await Promise.all([
        getHealth().catch(() => null),
        getScenarios().catch(() => []),
        getCatalog().catch(() => null),
        getMetrics().catch(() => []),
      ]);

      if (hData) setHealth(hData);
      if (scData && scData.length > 0) {
        setScenarios(scData);
        setSelectedScenarioId(scData[0].id);
        setQuery(scData[0].sql);
        setDialect(scData[0].dialect || 'postgres');
        runVerification(scData[0].sql, scData[0].dialect || 'postgres');
      }
      if (catData) setCatalog(catData);
      if (metData) setMetricsList(metData);
    } catch (err) {
      console.error('Failed to load initial data:', err);
    }
  };

  const runVerification = async (sqlToVerify: string, targetDialect: Dialect = dialect) => {
    if (!sqlToVerify.trim()) return;
    setLoading(true);
    try {
      const rep = await verifyQuery(sqlToVerify, targetDialect);
      setReport(rep);
    } catch (err) {
      console.error('Verification error:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleSelectScenario = (sc: Scenario) => {
    setSelectedScenarioId(sc.id);
    setQuery(sc.sql);
    setDialect(sc.dialect);
    runVerification(sc.sql, sc.dialect);
  };

  return (
    <div className="min-h-screen bg-[#070a12] text-slate-100 flex flex-col selection:bg-cyan-500/30">
      <Header dialect={dialect} onDialectChange={(d) => { setDialect(d); runVerification(query, d); }} health={health} />

      <main className="flex-1 max-w-7xl w-full mx-auto p-4 md:p-6 space-y-5">
        {/* Scenario Carousel */}
        <ScenarioSelector
          scenarios={scenarios}
          selectedId={selectedScenarioId}
          onSelect={handleSelectScenario}
        />

        {/* SQL Diff & Mutation Studio */}
        <SqlDiffEditor
          query={query}
          onChangeQuery={setQuery}
          onVerify={() => runVerification(query, dialect)}
          report={report}
          loading={loading}
        />

        {/* Tab Navigation */}
        <div className="flex items-center space-x-1 border-b border-slate-800 pb-1 text-xs overflow-x-auto">
          <button
            onClick={() => setActiveTab('probes')}
            className={`flex items-center space-x-1.5 px-3.5 py-2 rounded-t-lg font-medium transition-colors whitespace-nowrap cursor-pointer ${
              activeTab === 'probes'
                ? 'bg-slate-900 text-cyan-300 border-t-2 border-cyan-500 border-x border-slate-800'
                : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            <ShieldAlert className="h-3.5 w-3.5" />
            <span>Falsification Probes</span>
            {report?.suite_result && (
              <span
                className={`ml-1 px-1.5 py-0.2 rounded text-[10px] ${
                  report.suite_result.passed
                    ? 'bg-emerald-950 text-emerald-400'
                    : 'bg-rose-950 text-rose-400'
                }`}
              >
                {report.suite_result.passed ? 'PASSED' : `${report.suite_result.all_violations.length} ISSUES`}
              </span>
            )}
          </button>

          <button
            onClick={() => setActiveTab('fanout')}
            className={`flex items-center space-x-1.5 px-3.5 py-2 rounded-t-lg font-medium transition-colors whitespace-nowrap cursor-pointer ${
              activeTab === 'fanout'
                ? 'bg-slate-900 text-cyan-300 border-t-2 border-cyan-500 border-x border-slate-800'
                : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            <GitFork className="h-3.5 w-3.5" />
            <span>Visual Fan-Out Flow</span>
          </button>

          <button
            onClick={() => setActiveTab('voucher')}
            className={`flex items-center space-x-1.5 px-3.5 py-2 rounded-t-lg font-medium transition-colors whitespace-nowrap cursor-pointer ${
              activeTab === 'voucher'
                ? 'bg-slate-900 text-cyan-300 border-t-2 border-cyan-500 border-x border-slate-800'
                : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            <FileCode2 className="h-3.5 w-3.5" />
            <span>Integrity Passport (proof.json)</span>
            {report?.voucher && (
              <span className="ml-1 px-1.5 py-0.2 rounded text-[10px] bg-cyan-950 text-cyan-400 font-mono">
                SIGNED
              </span>
            )}
          </button>

          <button
            onClick={() => setActiveTab('ledger')}
            className={`flex items-center space-x-1.5 px-3.5 py-2 rounded-t-lg font-medium transition-colors whitespace-nowrap cursor-pointer ${
              activeTab === 'ledger'
                ? 'bg-slate-900 text-cyan-300 border-t-2 border-cyan-500 border-x border-slate-800'
                : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            <History className="h-3.5 w-3.5" />
            <span>Audit Ledger & Trends</span>
          </button>

          <button
            onClick={() => setActiveTab('metrics')}
            className={`flex items-center space-x-1.5 px-3.5 py-2 rounded-t-lg font-medium transition-colors whitespace-nowrap cursor-pointer ${
              activeTab === 'metrics'
                ? 'bg-slate-900 text-cyan-300 border-t-2 border-cyan-500 border-x border-slate-800'
                : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            <BookOpen className="h-3.5 w-3.5" />
            <span>Semantic Layer</span>
            <span className="ml-1 px-1.5 py-0.2 rounded text-[10px] bg-slate-800 text-slate-400">
              {metricsList.length}
            </span>
          </button>

          <button
            onClick={() => setActiveTab('catalog')}
            className={`flex items-center space-x-1.5 px-3.5 py-2 rounded-t-lg font-medium transition-colors whitespace-nowrap cursor-pointer ${
              activeTab === 'catalog'
                ? 'bg-slate-900 text-cyan-300 border-t-2 border-cyan-500 border-x border-slate-800'
                : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            <Database className="h-3.5 w-3.5" />
            <span>Schema Catalog</span>
            <span className="ml-1 px-1.5 py-0.2 rounded text-[10px] bg-slate-800 text-slate-400">
              {catalog?.tables.length || 0}
            </span>
          </button>
        </div>

        {/* Tab Content Panes */}
        <div>
          {activeTab === 'probes' && <ProbeMatrix suiteResult={report?.suite_result || null} />}
          {activeTab === 'fanout' && <FanoutGraph report={report} />}
          {activeTab === 'voucher' && <ProofVoucherCard voucher={report?.voucher || null} />}
          {activeTab === 'ledger' && <AuditLedgerView />}
          {activeTab === 'metrics' && <MetricDictionary metrics={metricsList} />}
          {activeTab === 'catalog' && <CatalogViewer catalog={catalog} />}
        </div>
      </main>

      <footer className="border-t border-slate-800/80 bg-slate-950/60 py-4 px-6 text-center text-xs text-slate-500">
        <div className="max-w-7xl mx-auto flex items-center justify-between">
          <div className="flex items-center space-x-2">
            <span className="font-semibold text-slate-400">queryproof</span>
            <span>- Autonomous Text-to-SQL Verification Engine</span>
          </div>
          <div className="flex items-center space-x-3 text-[11px]">
            <span>Single-Binary Go Architecture (&lt; 20MB RAM)</span>
            <span>•</span>
            <span>React 19 Auditor Studio</span>
          </div>
        </div>
      </footer>
    </div>
  );
};

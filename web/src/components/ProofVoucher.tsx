import React, { useState } from 'react';
import type { ProofVoucher } from '../types';
import { ShieldCheck, Check, Copy, Download, KeyRound, Lock, CheckCircle2, AlertCircle } from 'lucide-react';
import { verifyPassport } from '../api/client';

interface ProofVoucherProps {
  voucher: ProofVoucher | null;
}

export const ProofVoucherCard: React.FC<ProofVoucherProps> = ({ voucher }) => {
  const [copied, setCopied] = useState(false);
  const [verifying, setVerifying] = useState(false);
  const [verifyResult, setVerifyResult] = useState<{ valid: boolean; error?: string } | null>(null);

  if (!voucher) {
    return (
      <div className="bg-slate-900/60 border border-slate-800 rounded-xl p-8 text-center text-slate-400 text-xs">
        No integrity voucher generated yet. Execute verification to issue a cryptographically signed passport.
      </div>
    );
  }

  const jsonStr = JSON.stringify(voucher, null, 2);

  const copyVoucher = () => {
    navigator.clipboard.writeText(jsonStr);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const downloadVoucher = () => {
    const blob = new Blob([jsonStr], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `proof-${voucher.voucher_id}.json`;
    a.click();
    URL.revokeObjectURL(url);
  };

  const checkSignature = async () => {
    setVerifying(true);
    setVerifyResult(null);
    try {
      const res = await verifyPassport(voucher);
      setVerifyResult(res);
    } catch (err: any) {
      setVerifyResult({ valid: false, error: err.message });
    } finally {
      setVerifying(false);
    }
  };

  return (
    <div className="bg-slate-900/60 border border-slate-800 rounded-xl overflow-hidden">
      {/* Header */}
      <div className="px-5 py-4 bg-slate-950/80 border-b border-slate-800 flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center space-x-3">
          <div className="h-8 w-8 rounded bg-gradient-to-tr from-cyan-600 to-indigo-600 flex items-center justify-center text-white shadow-md shadow-cyan-950">
            <Lock className="h-4 w-4" />
          </div>
          <div>
            <div className="flex items-center space-x-2">
              <h3 className="text-sm font-semibold text-slate-200">
                Cryptographic Integrity Passport (proof.json)
              </h3>
              <span className="px-1.5 py-0.5 rounded text-[10px] font-mono bg-cyan-950 text-cyan-300 border border-cyan-800">
                {voucher.version}
              </span>
            </div>
            <p className="text-xs text-slate-400">
              Deterministic tamper-evident certificate signed by queryproof root authority
            </p>
          </div>
        </div>

        <div className="flex items-center space-x-2">
          <button
            onClick={checkSignature}
            disabled={verifying}
            className="flex items-center space-x-1.5 px-3 py-1.5 rounded-lg bg-cyan-950 hover:bg-cyan-900 text-cyan-300 border border-cyan-800 text-xs font-medium transition-colors cursor-pointer"
          >
            <ShieldCheck className="h-3.5 w-3.5" />
            <span>{verifying ? 'Verifying...' : 'Verify Signature'}</span>
          </button>

          <button
            onClick={copyVoucher}
            className="flex items-center space-x-1 px-3 py-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-200 text-xs font-medium transition-colors cursor-pointer"
          >
            {copied ? <Check className="h-3.5 w-3.5 text-emerald-400" /> : <Copy className="h-3.5 w-3.5" />}
            <span>{copied ? 'Copied' : 'Copy'}</span>
          </button>

          <button
            onClick={downloadVoucher}
            className="flex items-center space-x-1 px-3 py-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-200 text-xs font-medium transition-colors cursor-pointer"
          >
            <Download className="h-3.5 w-3.5" />
            <span>Download</span>
          </button>
        </div>
      </div>

      {/* Signature Verification Banner */}
      {verifyResult && (
        <div
          className={`px-4 py-2 text-xs flex items-center space-x-2 border-b ${
            verifyResult.valid
              ? 'bg-emerald-950/60 border-emerald-800 text-emerald-300'
              : 'bg-rose-950/60 border-rose-800 text-rose-300'
          }`}
        >
          {verifyResult.valid ? (
            <>
              <CheckCircle2 className="h-4 w-4" />
              <span>
                Signature Verified Valid: SHA-256 HMAC digest matches voucher payload. Zero tampering detected.
              </span>
            </>
          ) : (
            <>
              <AlertCircle className="h-4 w-4" />
              <span>Signature Verification Failed: {verifyResult.error}</span>
            </>
          )}
        </div>
      )}

      {/* Metadata Badges */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-3 p-4 bg-slate-900/40 border-b border-slate-800 text-xs">
        <div className="bg-slate-950/60 border border-slate-800/80 rounded p-2.5">
          <span className="text-[10px] text-slate-400 uppercase tracking-wider block">Voucher ID</span>
          <span className="font-mono text-slate-200 font-medium truncate block mt-0.5">
            {voucher.voucher_id}
          </span>
        </div>
        <div className="bg-slate-950/60 border border-slate-800/80 rounded p-2.5">
          <span className="text-[10px] text-slate-400 uppercase tracking-wider block">Status</span>
          <span
            className={`font-semibold text-xs block mt-0.5 ${
              voucher.status === 'CERTIFIED'
                ? 'text-emerald-400'
                : voucher.status === 'FALSIFIED_AND_REWRITTEN'
                ? 'text-cyan-400'
                : 'text-rose-400'
            }`}
          >
            {voucher.status}
          </span>
        </div>
        <div className="bg-slate-950/60 border border-slate-800/80 rounded p-2.5">
          <span className="text-[10px] text-slate-400 uppercase tracking-wider block">Issued At</span>
          <span className="font-mono text-slate-200 text-[11px] block mt-0.5">
            {new Date(voucher.issued_at).toLocaleTimeString()} UTC
          </span>
        </div>
        <div className="bg-slate-950/60 border border-slate-800/80 rounded p-2.5">
          <span className="text-[10px] text-slate-400 uppercase tracking-wider block">Key Authority</span>
          <span className="font-mono text-cyan-300 text-[11px] flex items-center mt-0.5">
            <KeyRound className="h-3 w-3 mr-1" />
            {voucher.signature?.key_id || 'root-key'}
          </span>
        </div>
      </div>

      {/* JSON Viewer */}
      <div className="p-4 bg-slate-950/90 font-mono text-xs text-slate-300 overflow-x-auto max-h-[350px]">
        <pre className="leading-relaxed">{jsonStr}</pre>
      </div>
    </div>
  );
};

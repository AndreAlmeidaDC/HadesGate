"use client";

import React, { useCallback, useEffect, useState } from "react";
import { Copy, Download, Info, KeyRound, RefreshCw, RotateCcw, Save, ShieldCheck, Trash2, Upload } from "lucide-react";
import type { Language } from "@/lib/i18n";
import { translations } from "@/lib/i18n";
import type { SystemNotification } from "@/components/ui/system-toast";

type AdminRole = "viewer" | "operator" | "admin";

interface AdminCredential {
  key_id: string;
  label: string;
  role: AdminRole;
  enabled: boolean;
  created_at: string;
  last_used_at?: string;
  secret?: string;
}

interface RetentionPolicy {
  tenant_id: string;
  retention_days: number;
  max_rows: number;
  updated_at: string;
}

interface AuditEvent {
  id: string;
  actor_key_id: string;
  actor_role: AdminRole;
  action: string;
  target_type: string;
  target_id?: string;
  timestamp: string;
  nonce: string;
  signature: string;
}

interface GovernanceProps {
  gatewayUrl: string;
  apiKey: string;
  lang: Language;
  onNotify?: (notification: SystemNotification) => void;
}

export function Governance({ gatewayUrl, apiKey, lang, onNotify }: GovernanceProps) {
  const t = translations[lang].settings;
  const headers = { Authorization: `Bearer ${apiKey}`, "Content-Type": "application/json" };
  const [credentials, setCredentials] = useState<AdminCredential[]>([]);
  const [retention, setRetention] = useState<RetentionPolicy[]>([]);
  const [events, setEvents] = useState<AuditEvent[]>([]);
  const [label, setLabel] = useState("");
  const [role, setRole] = useState<AdminRole>("viewer");
  const [oneTimeSecret, setOneTimeSecret] = useState("");
  const [tenantID, setTenantID] = useState("*");
  const [retentionDays, setRetentionDays] = useState(30);
  const [maxRows, setMaxRows] = useState(1000);
  const [backupPayload, setBackupPayload] = useState("");
  const [loading, setLoading] = useState(true);
  const [adminRestricted, setAdminRestricted] = useState(false);
  const [isBootstrap, setIsBootstrap] = useState(false);
  const [rotatingKeyID, setRotatingKeyID] = useState<string | null>(null);
  const [rotatedCredential, setRotatedCredential] = useState<{
    keyID: string;
    label: string;
    secret: string;
  } | null>(null);

  const notify = useCallback((type: "success" | "error", message: string) => {
    onNotify?.({ type, title: "HADESGATE // GOVERNANCE", message });
  }, [onNotify]);

  const load = useCallback(async () => {
    setLoading(true);
    const [credentialResponse, retentionResponse, auditResponse, authResponse] = await Promise.all([
      fetch(`${gatewayUrl}/api/admin-credentials`, { headers }),
      fetch(`${gatewayUrl}/api/retention`, { headers }),
      fetch(`${gatewayUrl}/api/audit/events?limit=100`, { headers }),
      fetch(`${gatewayUrl}/api/auth/verify`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ key: apiKey }),
      }).catch(() => null),
    ]);
    setAdminRestricted(credentialResponse.status === 403);
    if (credentialResponse.ok) setCredentials(await credentialResponse.json());
    if (retentionResponse.ok) setRetention(await retentionResponse.json());
    if (auditResponse.ok) {
      const data = await auditResponse.json();
      setEvents(data.events || []);
    }
    if (authResponse && authResponse.ok) {
      const authData = await authResponse.json();
      setIsBootstrap(authData.key_id === "bootstrap");
    }
    setLoading(false);
  }, [apiKey, gatewayUrl]);

  useEffect(() => { void load(); }, [load]);

  const createCredential = async () => {
    const response = await fetch(`${gatewayUrl}/api/admin-credentials`, { method: "POST", headers, body: JSON.stringify({ label, role }) });
    if (!response.ok) { notify("error", t.adminOnly); return; }
    const credential: AdminCredential = await response.json();
    setOneTimeSecret(credential.secret || "");
    setLabel("");
    notify("success", t.oneTimeSecret);
    await load();
  };

  const rotateCredential = async (keyID: string, credentialLabel: string) => {
    if (!isBootstrap) {
      notify("error", t.onlyBootstrapCanRotate);
      return;
    }
    const confirmed = confirm(
      `${t.rotateCredentialConfirm}\n\nID: ${keyID}\nRótulo: ${credentialLabel}`
    );
    if (!confirmed) return;

    setRotatingKeyID(keyID);
    try {
      const response = await fetch(`${gatewayUrl}/api/admin-credentials/rotate`, {
        method: "POST",
        headers,
        body: JSON.stringify({ key_id: keyID }),
      });
      if (!response.ok) {
        const data = await response.json().catch(() => ({}));
        throw new Error(data.error?.message || t.onlyBootstrapCanRotate);
      }
      const data: AdminCredential = await response.json();
      if (data.secret) {
        setRotatedCredential({
          keyID: data.key_id,
          label: data.label || credentialLabel,
          secret: data.secret,
        });
        notify("success", t.rotateSuccess);
      }
      await load();
    } catch (err: any) {
      notify("error", err.message);
    } finally {
      setRotatingKeyID(null);
    }
  };

  const revokeCredential = async (keyID: string) => {
    const response = await fetch(`${gatewayUrl}/api/admin-credentials?key_id=${encodeURIComponent(keyID)}`, { method: "DELETE", headers });
    if (!response.ok) { notify("error", t.adminOnly); return; }
    await load();
  };

  const saveRetention = async () => {
    const response = await fetch(`${gatewayUrl}/api/retention`, { method: "PUT", headers, body: JSON.stringify({ tenant_id: tenantID, retention_days: retentionDays, max_rows: maxRows }) });
    if (!response.ok) { notify("error", t.adminOnly); return; }
    notify("success", t.savedSuccess);
    await load();
  };

  const downloadJSON = (filename: string, data: unknown) => {
    const url = URL.createObjectURL(new Blob([JSON.stringify(data, null, 2)], { type: "application/json" }));
    const anchor = document.createElement("a"); anchor.href = url; anchor.download = filename; anchor.click(); URL.revokeObjectURL(url);
  };

  const exportAudit = async () => {
    const response = await fetch(`${gatewayUrl}/api/audit/events?limit=1000&download=1`, { headers });
    if (response.ok) downloadJSON("hadesgate-audit.json", await response.json());
  };

  const exportBackup = async () => {
    const response = await fetch(`${gatewayUrl}/api/config-backup`, { headers });
    if (!response.ok) { notify("error", t.adminOnly); return; }
    downloadJSON("hadesgate-config-backup.json", await response.json());
  };

  const restoreBackup = async () => {
    let envelope = backupPayload.trim();
    try { const parsed = JSON.parse(envelope); envelope = parsed.backup || envelope; } catch { /* envelope pasted directly */ }
    const response = await fetch(`${gatewayUrl}/api/config-backup`, { method: "POST", headers, body: JSON.stringify({ backup: envelope }) });
    if (!response.ok) { notify("error", t.adminOnly); return; }
    setBackupPayload(""); notify("success", t.savedSuccess); await load();
  };

  if (loading) return <div className="border-2 border-zinc-900 dark:border-[#38384A] bg-card p-8 font-mono text-xs uppercase"><RefreshCw className="inline h-4 w-4 animate-spin mr-2" />{translations[lang].common.loading}</div>;

  return (
    <div className="border-2 border-zinc-900 dark:border-[#38384A] bg-card p-6 space-y-6">
      <header className="border-b-2 border-zinc-900 dark:border-[#38384A] pb-4">
        <h3 className="font-display text-xl uppercase text-foreground">{t.governanceTitle}</h3>
        <p className="font-mono text-[11px] uppercase tracking-[0.05em] text-muted-foreground mt-2">{t.governanceDesc}</p>
      </header>

      <section className="grid lg:grid-cols-[1fr_2fr] gap-px bg-border border border-border">
        <div className="bg-card p-4 space-y-3">
          <h4 className="font-mono text-xs font-bold uppercase"><KeyRound className="inline h-4 w-4 mr-2 text-purple-600" />{t.adminCredentials}</h4>
          {adminRestricted && <p className="border border-red-600 p-2 font-mono text-[10px] uppercase text-red-600">[ {t.adminOnly} ]</p>}
          {!adminRestricted && <>
            <input value={label} onChange={(event) => setLabel(event.target.value)} placeholder={t.credentialLabel} className="w-full h-10 px-3 border border-border bg-secondary font-mono text-xs" />
            <select value={role} onChange={(event) => setRole(event.target.value as AdminRole)} className="w-full h-10 px-3 border border-border bg-secondary font-mono text-xs uppercase"><option value="viewer">viewer</option><option value="operator">operator</option><option value="admin">admin</option></select>
            <button onClick={createCredential} disabled={!label.trim()} className="w-full h-10 border-2 border-purple-700 bg-purple-700 text-white font-mono text-xs font-bold uppercase disabled:opacity-40"><KeyRound className="inline h-4 w-4 mr-2" />{t.createCredential}</button>
          </>}
          {oneTimeSecret && <div className="border-2 border-purple-600 p-3 bg-purple-600/10"><p className="font-mono text-[10px] font-bold uppercase text-purple-600 mb-2">{t.oneTimeSecret}</p><code className="font-mono text-[10px] break-all">{oneTimeSecret}</code><button onClick={() => navigator.clipboard.writeText(oneTimeSecret)} className="block mt-2 font-mono text-[10px] uppercase"><Copy className="inline h-3 w-3 mr-1" />COPY</button></div>}
        </div>
        <div className="bg-card p-4 overflow-x-auto space-y-3">
          {rotatedCredential && (
            <div className="border-2 border-purple-600 bg-purple-950/40 p-3.5 space-y-2">
              <div className="flex items-center justify-between">
                <span className="font-mono text-[11px] font-bold uppercase text-purple-300 flex items-center gap-1.5">
                  <KeyRound className="h-3.5 w-3.5 text-purple-400" />
                  {t.rotatedSecretTitle}
                </span>
                <button
                  type="button"
                  onClick={() => setRotatedCredential(null)}
                  className="font-mono text-[9px] text-muted-foreground hover:text-foreground uppercase"
                >
                  [ FECHAR ]
                </button>
              </div>
              <p className="font-mono text-[10px] text-muted-foreground uppercase">
                Credencial: <strong className="text-foreground">{rotatedCredential.label}</strong> ({rotatedCredential.keyID}). {t.rotatedSecretDesc}
              </p>
              <div className="flex items-center gap-2 bg-secondary p-2 border border-border">
                <code className="font-mono text-[11px] text-purple-300 break-all select-all flex-1">
                  {rotatedCredential.secret}
                </code>
                <button
                  type="button"
                  onClick={() => {
                    navigator.clipboard.writeText(rotatedCredential.secret);
                    notify("success", t.keyCopiedOnce);
                  }}
                  className="h-7 px-3 border border-purple-600 bg-purple-700 hover:bg-purple-800 text-white font-mono text-[10px] font-bold uppercase flex items-center gap-1 transition-colors shrink-0"
                >
                  <Copy className="h-3 w-3" />
                  <span>COPIAR</span>
                </button>
              </div>
            </div>
          )}

          {credentials.length === 0 ? (
            <p className="font-mono text-xs text-muted-foreground">[ {t.noGovernanceData} ]</p>
          ) : (
            <table className="w-full font-mono text-[10px] uppercase">
              <thead>
                <tr className="border-b border-border text-left">
                  <th className="p-2">KEY_ID</th>
                  <th className="p-2">LABEL</th>
                  <th className="p-2">ROLE</th>
                  <th className="p-2">LAST_USED</th>
                  <th className="p-2 text-right">AÇÕES</th>
                </tr>
              </thead>
              <tbody>
                {credentials.map((credential) => (
                  <tr key={credential.key_id} className="border-b border-border">
                    <td className="p-2 font-bold">{credential.key_id}</td>
                    <td className="p-2">{credential.label}</td>
                    <td className="p-2 text-purple-600">{credential.role}</td>
                    <td className="p-2">
                      {credential.last_used_at
                        ? new Date(credential.last_used_at).toLocaleString(lang)
                        : "—"}
                    </td>
                    <td className="p-2 text-right">
                      <div className="flex items-center justify-end gap-1.5">
                        {isBootstrap && (
                          <button
                            type="button"
                            onClick={() => rotateCredential(credential.key_id, credential.label)}
                            disabled={rotatingKeyID === credential.key_id}
                            title={t.rotateCredentialTitle}
                            className="px-2 py-1 border border-purple-600/50 bg-purple-950/20 hover:bg-purple-900/40 text-purple-300 font-mono text-[10px] font-bold uppercase flex items-center gap-1 transition-colors disabled:opacity-40"
                          >
                            {rotatingKeyID === credential.key_id ? (
                              <RefreshCw className="h-3 w-3 animate-spin" />
                            ) : (
                              <RotateCcw className="h-3 w-3 text-purple-400" />
                            )}
                            <span className="hidden sm:inline">{t.rotateCredential}</span>
                          </button>
                        )}
                        <button
                          type="button"
                          onClick={() => revokeCredential(credential.key_id)}
                          title={t.revokeCredential}
                          className="p-1 border border-rose-900/40 bg-rose-950/20 hover:bg-rose-900/40 text-rose-400 transition-colors"
                        >
                          <Trash2 className="h-3.5 w-3.5" />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>
      </section>

      <section className="border-2 border-zinc-900 dark:border-[#38384A] bg-card p-5 space-y-4">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2 border-b border-border pb-3">
          <div>
            <h4 className="font-mono text-xs font-bold uppercase tracking-wider flex items-center gap-2 text-foreground">
              <Save className="h-4 w-4 text-purple-500" />
              <span>{t.retentionTitle}</span>
            </h4>
            <p className="font-mono text-[10px] text-muted-foreground uppercase tracking-wider mt-1">
              Políticas de expurgo automático de telemetria e traces Zero-PII
            </p>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            {retention.map((policy) => (
              <span
                key={policy.tenant_id}
                className="border border-purple-600/40 bg-purple-950/20 px-2.5 py-1 font-mono text-[10px] uppercase font-bold text-purple-300"
              >
                [{policy.tenant_id}] {policy.retention_days}D / {policy.max_rows} ROWS
              </span>
            ))}
          </div>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {/* Campo 1: Escopo / Tenant */}
          <div className="space-y-1.5">
            <div className="flex items-center justify-between">
              <label className="font-mono text-[11px] font-bold uppercase text-foreground flex items-center gap-1.5">
                <span>{t.retentionTenantLabel}</span>
              </label>
              <span
                className="flex items-center gap-1 font-mono text-[9px] uppercase text-purple-400 bg-purple-950/40 px-1.5 py-0.5 border border-purple-800/40"
                title={t.retentionTenantDesc}
              >
                <Info className="h-3 w-3 text-purple-400" />
                <span>INFO</span>
              </span>
            </div>
            <input
              id="retention_tenant_scope"
              name="retention_tenant_scope"
              value={tenantID}
              onChange={(event) => setTenantID(event.target.value)}
              placeholder="* (Global) ou ID do Tenant"
              className="w-full h-10 px-3 border border-border bg-secondary font-mono text-xs text-foreground focus:outline-none focus:border-purple-600"
            />
            <p className="font-mono text-[10px] text-muted-foreground flex items-start gap-1">
              <span className="text-purple-500 font-bold">&gt;</span>
              <span>{t.retentionTenantDesc}</span>
            </p>
          </div>

          {/* Campo 2: Dias de Retenção */}
          <div className="space-y-1.5">
            <div className="flex items-center justify-between">
              <label className="font-mono text-[11px] font-bold uppercase text-foreground flex items-center gap-1.5">
                <span>{t.retentionDaysLabel}</span>
              </label>
              <span
                className="flex items-center gap-1 font-mono text-[9px] uppercase text-purple-400 bg-purple-950/40 px-1.5 py-0.5 border border-purple-800/40"
                title={t.retentionDaysDesc}
              >
                <Info className="h-3 w-3 text-purple-400" />
                <span>INFO</span>
              </span>
            </div>
            <input
              id="retention_days_limit"
              name="retention_days_limit"
              type="number"
              min={0}
              value={retentionDays}
              onChange={(event) => setRetentionDays(Number(event.target.value))}
              placeholder="30"
              className="w-full h-10 px-3 border border-border bg-secondary font-mono text-xs text-foreground focus:outline-none focus:border-purple-600"
            />
            <p className="font-mono text-[10px] text-muted-foreground flex items-start gap-1">
              <span className="text-purple-500 font-bold">&gt;</span>
              <span>{t.retentionDaysDesc}</span>
            </p>
          </div>

          {/* Campo 3: Máximo de Registros */}
          <div className="space-y-1.5">
            <div className="flex items-center justify-between">
              <label className="font-mono text-[11px] font-bold uppercase text-foreground flex items-center gap-1.5">
                <span>{t.retentionMaxRowsLabel}</span>
              </label>
              <span
                className="flex items-center gap-1 font-mono text-[9px] uppercase text-purple-400 bg-purple-950/40 px-1.5 py-0.5 border border-purple-800/40"
                title={t.retentionMaxRowsDesc}
              >
                <Info className="h-3 w-3 text-purple-400" />
                <span>INFO</span>
              </span>
            </div>
            <input
              id="retention_max_rows"
              name="retention_max_rows"
              type="number"
              min={0}
              value={maxRows}
              onChange={(event) => setMaxRows(Number(event.target.value))}
              placeholder="1000"
              className="w-full h-10 px-3 border border-border bg-secondary font-mono text-xs text-foreground focus:outline-none focus:border-purple-600"
            />
            <p className="font-mono text-[10px] text-muted-foreground flex items-start gap-1">
              <span className="text-purple-500 font-bold">&gt;</span>
              <span>{t.retentionMaxRowsDesc}</span>
            </p>
          </div>
        </div>

        <div className="pt-2 flex justify-end">
          <button
            onClick={saveRetention}
            className="h-10 px-6 border-2 border-purple-700 bg-purple-700 hover:bg-purple-800 text-white font-mono text-xs font-bold uppercase tracking-[0.08em] flex items-center gap-2 transition-colors"
          >
            <Save className="h-4 w-4" />
            <span>{t.saveRetention}</span>
          </button>
        </div>
      </section>

      <section className="border border-border p-4"><div className="flex items-center justify-between mb-3"><h4 className="font-mono text-xs font-bold uppercase"><ShieldCheck className="inline h-4 w-4 mr-2 text-purple-600" />{t.auditTrail}</h4><button onClick={exportAudit} className="border border-border px-3 py-2 font-mono text-[10px] uppercase"><Download className="inline h-3 w-3 mr-1" />{t.downloadAudit}</button></div><div className="max-h-72 overflow-auto">{events.length === 0 ? <p className="font-mono text-xs text-muted-foreground">[ {t.noGovernanceData} ]</p> : events.map((event) => <div key={event.id} className="grid md:grid-cols-[160px_1fr_1fr] gap-2 border-t border-border py-2 font-mono text-[10px] uppercase"><time>{new Date(event.timestamp).toLocaleString(lang)}</time><span>{event.actor_key_id} [{event.actor_role}]</span><span>{event.action} &gt; {event.target_type}:{event.target_id || "global"}<small className="block text-muted-foreground truncate">SIG {event.signature.slice(0, 20)}… / NONCE {event.nonce.slice(0, 12)}…</small></span></div>)}</div></section>

      <section className="border border-border p-4 space-y-3"><h4 className="font-mono text-xs font-bold uppercase">{t.backupTitle}</h4><div className="flex gap-2"><button onClick={exportBackup} className="h-10 px-4 border border-border font-mono text-xs font-bold uppercase"><Download className="inline h-4 w-4 mr-2" />{t.exportBackup}</button></div><textarea value={backupPayload} onChange={(event) => setBackupPayload(event.target.value)} placeholder={t.backupPayload} className="w-full min-h-24 p-3 border border-border bg-secondary font-mono text-[10px]" /><button onClick={restoreBackup} disabled={!backupPayload.trim()} className="h-10 px-4 border-2 border-red-600 text-red-600 font-mono text-xs font-bold uppercase disabled:opacity-40"><Upload className="inline h-4 w-4 mr-2" />{t.restoreBackup}</button></section>
    </div>
  );
}

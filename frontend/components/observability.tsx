"use client";

import React, { useState, useEffect, useCallback } from "react";
import { RefreshCw, Trash2, Search, ChevronLeft, ChevronRight } from "lucide-react";
import type { Language } from "@/lib/i18n";
import { translations } from "@/lib/i18n";

interface Trace {
  id: string;
  tenant?: string;
  policy_action?: string;
  timestamp: string;
  provider: string;
  model: string;
  language: string;
  stream: boolean;
  status_code: number;
  duration_ms: number;
  anon_duration_us: number;
  entities_count: number;
  entity_types: string[];
  pii_types?: string[];
  secret_types?: string[];
  sanitized_prompt: string;
}

interface Summary {
  total_requests: number;
  masked_requests: number;
  masked_rate_pct: number;
  blocked_requests: number;
  total_entities: number;
  avg_scan_us: number;
  avg_scan_ms: number;
  avg_duration_ms: number;
  requests_last_hour: number;
  provider_distribution?: Record<string, number>;
  model_distribution?: Record<string, number>;
  entity_type_counts?: Record<string, number>;
  pii_type_counts?: Record<string, number>;
  secret_type_counts?: Record<string, number>;
}

interface ObservabilityProps {
  gatewayUrl: string;
  apiKey: string;
  lang?: Language;
}

const SECRET_LABELS = new Set([
  "API_KEY",
  "SECRET",
  "PASSWORD",
  "JWT",
  "PRIVATE_KEY",
  "DB_URI",
  "SSH_KEY",
  "CONFIDENTIAL",
]);

/**
 * Renders a sanitized prompt replacing [HG_<ns>_<TYPE>_<N>], [PG_<ns>_<TYPE>_<N>] or [[<TYPE>_<N>]]
 * with sharp 90-degree Industrial Brutalist purple token blocks [[TYPE_N]].
 */
export function renderMaskedPrompt(text: string): React.ReactNode {
  if (!text) {
    return (
      <span className="text-muted-foreground font-mono text-xs uppercase">
        [ EMPTY_PAYLOAD ]
      </span>
    );
  }

  const tokenRegex =
    /\[(?:HG|PG)_[a-fA-F0-9]+_([A-Z0-9_]+_\d+)\]|\[\[([A-Z0-9_]+_\d+)\]\]/g;
  const parts: React.ReactNode[] = [];
  let lastIndex = 0;
  let match: RegExpExecArray | null;

  while ((match = tokenRegex.exec(text)) !== null) {
    if (match.index > lastIndex) {
      parts.push(
        <React.Fragment key={`t-${lastIndex}`}>
          {text.slice(lastIndex, match.index)}
        </React.Fragment>
      );
    }
    const label = match[1] || match[2];
    parts.push(
      <samp
        key={`m-${match.index}`}
        className="inline-block mx-0.5 px-1.5 py-0.5 border border-purple-700 dark:border-purple-400 bg-purple-700/15 dark:bg-purple-500/20 text-purple-900 dark:text-purple-200 font-mono text-xs font-bold tracking-[0.04em] align-baseline"
      >
        [[{label}]]
      </samp>
    );
    lastIndex = match.index + match[0].length;
  }

  if (lastIndex < text.length) {
    parts.push(
      <React.Fragment key="t-end">{text.slice(lastIndex)}</React.Fragment>
    );
  }

  return parts;
}

function splitFallbackTypes(trace: Trace): { pii: string[]; secrets: string[] } {
  if (
    (trace.pii_types && trace.pii_types.length > 0) ||
    (trace.secret_types && trace.secret_types.length > 0)
  ) {
    return {
      pii: trace.pii_types || [],
      secrets: trace.secret_types || [],
    };
  }

  const pii: string[] = [];
  const secrets: string[] = [];
  const seenPii = new Set<string>();
  const seenSec = new Set<string>();

  for (const t of trace.entity_types || []) {
    if (SECRET_LABELS.has(t)) {
      if (!seenSec.has(t)) {
        seenSec.add(t);
        secrets.push(t);
      }
    } else if (!seenPii.has(t)) {
      seenPii.add(t);
      pii.push(t);
    }
  }
  return { pii, secrets };
}

export function Observability({
  gatewayUrl,
  apiKey,
  lang = "pt-BR",
}: ObservabilityProps) {
  const t = translations[lang];

  const [traces, setTraces] = useState<Trace[]>([]);
  const [summary, setSummary] = useState<Summary | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [autoRefresh, setAutoRefresh] = useState(true);
  const [expandedIds, setExpandedIds] = useState<Record<string, boolean>>({});
  const [hasAutoExpanded, setHasAutoExpanded] = useState(false);
  const [selectedProvider, setSelectedProvider] = useState<string>("all");
  const [searchQuery, setSearchQuery] = useState<string>("");

  // Paginação no SQLite
  const [page, setPage] = useState<number>(1);
  const [pageSize, setPageSize] = useState<number>(10);
  const [totalCount, setTotalCount] = useState<number>(0);
  const [totalPages, setTotalPages] = useState<number>(1);

  const fetchTraces = useCallback(
    async (currentPage = page, currentLimit = pageSize) => {
      try {
        setIsLoading(true);
        const headers: Record<string, string> = {};
        if (apiKey) headers["Authorization"] = `Bearer ${apiKey}`;

        const url = `${gatewayUrl}/api/traces?page=${currentPage}&limit=${currentLimit}`;
        const res = await fetch(url, { headers });
        if (res.ok) {
          const data = await res.json();
          const list: Trace[] = data.traces || [];
          setTraces(list);
          setSummary(data.summary || null);

          if (data.pagination) {
            setTotalCount(data.pagination.total || 0);
            setTotalPages(Math.max(1, data.pagination.total_pages || 1));
          } else {
            setTotalCount(list.length);
            setTotalPages(1);
          }

          if (!hasAutoExpanded && list.length > 0) {
            setExpandedIds({ [list[0].id]: true });
            setHasAutoExpanded(true);
          }
        }
      } catch (e) {
        console.warn("Failed to fetch traces:", e);
      } finally {
        setIsLoading(false);
      }
    },
    [gatewayUrl, apiKey, page, pageSize, hasAutoExpanded]
  );

  useEffect(() => {
    fetchTraces(page, pageSize);
  }, [fetchTraces, page, pageSize]);

  useEffect(() => {
    if (!autoRefresh) return;
    const interval = setInterval(() => {
      fetchTraces(page, pageSize);
    }, 3000);
    return () => clearInterval(interval);
  }, [autoRefresh, fetchTraces, page, pageSize]);

  const handleClearTraces = async () => {
    const confirmMsg =
      lang === "pt-BR"
        ? "CONFIRMAR EXPURGO DE TELEMETRIA NO SQLITE?"
        : "CONFIRM PURGING TELEMETRY RECORDS IN SQLITE?";
    if (!confirm(confirmMsg)) return;

    try {
      const headers: Record<string, string> = {};
      if (apiKey) headers["Authorization"] = `Bearer ${apiKey}`;

      await fetch(`${gatewayUrl}/api/traces`, {
        method: "DELETE",
        headers,
      });
      setExpandedIds({});
      setHasAutoExpanded(false);
      setPage(1);
      fetchTraces(1, pageSize);
    } catch (e) {
      console.warn("Failed to clear traces:", e);
    }
  };

  const toggleRow = (id: string) => {
    setExpandedIds((prev) => ({
      ...prev,
      [id]: !prev[id],
    }));
  };

  const formatTime = (ts: string) => {
    try {
      const date = new Date(ts);
      return date.toLocaleTimeString(lang === "pt-BR" ? "pt-BR" : "en-US", {
        hour12: false,
      });
    } catch {
      return ts;
    }
  };

  // Cálculos de KPI diretos
  const totalRequests = summary?.total_requests ?? totalCount;
  const maskedRequests =
    summary?.masked_requests ?? traces.filter((t) => t.entities_count > 0).length;
  const maskedPct =
    totalRequests > 0 ? Math.round((maskedRequests / totalRequests) * 100) : 0;
  const totalEntities =
    summary?.total_entities ?? traces.reduce((acc, t) => acc + t.entities_count, 0);
  const avgScanUs = summary?.avg_scan_us ?? 0;
  const avgScanMs = summary?.avg_scan_ms ?? 0;
  const requestsLastHour = summary?.requests_last_hour ?? traces.length;

  const scanDisplayValue =
    totalRequests === 0
      ? "0"
      : avgScanUs >= 1000
      ? avgScanMs >= 10
        ? String(Math.round(avgScanMs))
        : avgScanMs.toFixed(1)
      : String(avgScanUs);

  const scanDisplayUnit =
    totalRequests === 0
      ? "µS AVG"
      : avgScanUs >= 1000
      ? "MS AVG"
      : `µS AVG (${avgScanMs.toFixed(2)} MS)`;

  const providerCounts: Record<string, number> =
    summary?.provider_distribution &&
    Object.keys(summary.provider_distribution).length > 0
      ? summary.provider_distribution
      : traces.reduce<Record<string, number>>((acc, t) => {
          const p = t.provider || "openai";
          acc[p] = (acc[p] || 0) + 1;
          return acc;
        }, {});

  const filteredTraces = traces.filter((t) => {
    const prov = t.provider || "openai";
    if (selectedProvider !== "all" && prov !== selectedProvider) return false;
    if (!searchQuery.trim()) return true;
    const q = searchQuery.toLowerCase();
    const { pii, secrets } = splitFallbackTypes(t);
    return (
      prov.toLowerCase().includes(q) ||
      (t.model && t.model.toLowerCase().includes(q)) ||
      (t.language && t.language.toLowerCase().includes(q)) ||
      (t.sanitized_prompt && t.sanitized_prompt.toLowerCase().includes(q)) ||
      pii.some((item) => item.toLowerCase().includes(q)) ||
      secrets.some((item) => item.toLowerCase().includes(q))
    );
  });

  const startRecord = (page - 1) * pageSize + 1;
  const endRecord = Math.min(page * pageSize, totalCount);

  return (
    <section className="relative border-2 border-zinc-900 dark:border-[#3A3A4C] bg-card">
      {/* Corner Mechanical Crosshairs */}
      <span
        aria-hidden="true"
        className="pointer-events-none select-none absolute -top-2.5 -left-2 font-mono text-xs font-bold text-purple-600 dark:text-purple-400"
      >
        +
      </span>
      <span
        aria-hidden="true"
        className="pointer-events-none select-none absolute -top-2.5 -right-2 font-mono text-xs font-bold text-purple-600 dark:text-purple-400"
      >
        +
      </span>
      <span
        aria-hidden="true"
        className="pointer-events-none select-none absolute -bottom-2.5 -left-2 font-mono text-xs font-bold text-purple-600 dark:text-purple-400"
      >
        +
      </span>
      <span
        aria-hidden="true"
        className="pointer-events-none select-none absolute -bottom-2.5 -right-2 font-mono text-xs font-bold text-purple-600 dark:text-purple-400"
      >
        +
      </span>

      {/* Top Structural Header Bar */}
      <div className="border-b-2 border-zinc-900 dark:border-[#3A3A4C] border-l-4 border-l-purple-700 dark:border-l-purple-500 px-5 py-4 flex flex-wrap items-center justify-between gap-4 brutalist-hatch">
        <div className="space-y-1">
          <div className="flex flex-wrap items-center gap-3">
            <h1 className="font-display text-2xl sm:text-3xl uppercase tracking-[-0.04em] leading-[0.9] text-foreground">
              HADESGATE<span className="text-purple-700 dark:text-purple-400">™</span>
            </h1>
            <span className="border border-purple-700 dark:border-purple-500 bg-purple-700/15 dark:bg-purple-500/20 px-2 py-0.5 font-mono text-[10px] font-bold uppercase tracking-[0.1em] text-purple-900 dark:text-purple-200">
              [ SQLITE // ZERO-RAW-PII ]
            </span>
          </div>
          <p className="font-mono text-[10px] uppercase tracking-[0.1em] text-muted-foreground">
            &gt;&gt;&gt; REAL-TIME DATA ANONYMIZATION &amp; SECRET INTERCEPTION MATRIX
          </p>
        </div>

        {/* Right Telemetry Controls & Single Phosphor Status Readout */}
        <div className="flex items-center gap-2">
          <button
            onClick={() => fetchTraces(page, pageSize)}
            disabled={isLoading}
            title={t.common.refresh}
            className="h-8 px-3 border border-zinc-800 dark:border-[#38384A] bg-card hover:border-purple-600 dark:hover:border-purple-400 font-mono text-[10px] font-bold uppercase tracking-[0.08em] text-foreground flex items-center gap-1.5 transition-colors"
          >
            <RefreshCw className={`h-3 w-3 ${isLoading ? "animate-spin" : ""}`} />
            <span className="hidden sm:inline">{t.common.refresh}</span>
          </button>

          {totalCount > 0 && (
            <button
              onClick={handleClearTraces}
              title={t.common.purge}
              className="h-8 px-3 border border-zinc-800 dark:border-[#38384A] bg-card hover:border-red-600 hover:bg-red-600 hover:text-white font-mono text-[10px] font-bold uppercase tracking-[0.08em] text-muted-foreground flex items-center gap-1.5 transition-colors"
            >
              <Trash2 className="h-3 w-3" />
              <span className="hidden sm:inline">{t.common.purge}</span>
            </button>
          )}

          <button
            onClick={() => setAutoRefresh((v) => !v)}
            title={
              autoRefresh
                ? "Telemetria em tempo real ativa (3s)"
                : "Telemetria pausada — clique para retomar"
            }
            className="h-8 px-3.5 border border-zinc-900 dark:border-[#4A4A5E] bg-secondary font-mono text-[10px] font-bold uppercase tracking-[0.12em] text-foreground flex items-center gap-2 transition-colors hover:border-purple-500"
          >
            <span
              className={`h-2 w-2 ${
                autoRefresh
                  ? "bg-[#16A34A] dark:bg-[#4AF626] animate-pulse"
                  : "bg-zinc-500"
              }`}
            />
            <span>{autoRefresh ? "LIVE // 3S" : "PAUSED"}</span>
          </button>
        </div>
      </div>

      {/* 4-Column Blueprint KPI Grid */}
      <dl className="grid grid-cols-2 lg:grid-cols-4 gap-px bg-zinc-900 dark:bg-[#2E2E3D] border-b-2 border-zinc-900 dark:border-[#3A3A4C]">
        {/* KPI 01: REQUESTS */}
        <div className="bg-card px-5 py-5 flex flex-col justify-between">
          <dt className="font-mono text-[10px] font-bold uppercase tracking-[0.1em] text-muted-foreground flex items-center justify-between">
            <span>01 // {t.observability.kpiRequests}</span>
            <span className="text-purple-700 dark:text-purple-400">///</span>
          </dt>
          <dd className="mt-2">
            <data
              value={totalRequests}
              className="block font-display text-4xl sm:text-5xl tracking-[-0.04em] leading-none text-foreground"
            >
              {totalRequests}
            </data>
            <span className="mt-2 block font-mono text-[10px] uppercase tracking-[0.08em] text-muted-foreground">
              {t.observability.totalLabel}
            </span>
          </dd>
        </div>

        {/* KPI 02: MASKED */}
        <div className="bg-card px-5 py-5 flex flex-col justify-between">
          <dt className="font-mono text-[10px] font-bold uppercase tracking-[0.1em] text-muted-foreground flex items-center justify-between">
            <span>02 // {t.observability.kpiMasked}</span>
            <span className="text-purple-700 dark:text-purple-400">[PII+SEC]</span>
          </dt>
          <dd className="mt-2">
            <data
              value={maskedPct}
              className="block font-display text-4xl sm:text-5xl tracking-[-0.04em] leading-none text-purple-700 dark:text-purple-400"
            >
              {maskedPct}%
            </data>
            <div className="mt-2 flex flex-wrap items-center gap-2 font-mono text-[10px] uppercase tracking-[0.06em] text-muted-foreground">
              <span>
                {maskedRequests} {t.common.of} {totalRequests}
              </span>
              <span className="border border-purple-700/60 dark:border-purple-500/60 bg-purple-700/10 dark:bg-purple-500/15 px-1.5 py-0.2 text-purple-900 dark:text-purple-200 font-bold">
                {totalEntities} {t.observability.securedRatio}
              </span>
            </div>
          </dd>
        </div>

        {/* KPI 03: SCAN TIME */}
        <div className="bg-card px-5 py-5 flex flex-col justify-between">
          <dt className="font-mono text-[10px] font-bold uppercase tracking-[0.1em] text-muted-foreground flex items-center justify-between">
            <span>03 // {t.observability.kpiScanTime}</span>
            <span className="text-purple-700 dark:text-purple-400">LATENCY</span>
          </dt>
          <dd className="mt-2">
            <data
              value={avgScanUs}
              className="block font-display text-4xl sm:text-5xl tracking-[-0.04em] leading-none text-purple-700 dark:text-purple-400"
            >
              {scanDisplayValue}
            </data>
            <span className="mt-2 block font-mono text-[10px] uppercase tracking-[0.08em] text-muted-foreground">
              {scanDisplayUnit}
            </span>
          </dd>
        </div>

        {/* KPI 04: RATE */}
        <div className="bg-card px-5 py-5 flex flex-col justify-between">
          <dt className="font-mono text-[10px] font-bold uppercase tracking-[0.1em] text-muted-foreground flex items-center justify-between">
            <span>04 // {t.observability.kpiRate}</span>
            <span className="text-purple-700 dark:text-purple-400">60M</span>
          </dt>
          <dd className="mt-2">
            <data
              value={requestsLastHour}
              className="block font-display text-4xl sm:text-5xl tracking-[-0.04em] leading-none text-foreground"
            >
              {requestsLastHour}
            </data>
            <span className="mt-2 block font-mono text-[10px] uppercase tracking-[0.08em] text-muted-foreground">
              {t.observability.perHour}
            </span>
          </dd>
        </div>
      </dl>

      {/* Provider / IA Filter Matrix & Query Input */}
      <div className="px-5 py-3 bg-secondary/60 border-b border-border flex flex-col md:flex-row md:items-center justify-between gap-3">
        <div className="flex flex-wrap items-center gap-1.5 font-mono text-[10px] uppercase tracking-[0.08em]">
          <span className="text-muted-foreground font-bold mr-1">
            {t.observability.providerMatrix}
          </span>
          <button
            onClick={() => setSelectedProvider("all")}
            className={`px-2.5 py-1 border font-bold transition-colors ${
              selectedProvider === "all"
                ? "border-purple-700 dark:border-purple-400 bg-purple-700 dark:bg-purple-600 text-white"
                : "border-border bg-card text-muted-foreground hover:text-foreground hover:border-purple-500"
            }`}
          >
            [ {t.common.all}: {totalCount || traces.length} ]
          </button>
          {Object.entries(providerCounts).map(([prov, count]) => (
            <button
              key={prov}
              onClick={() => setSelectedProvider(prov)}
              className={`px-2.5 py-1 border font-bold transition-colors ${
                selectedProvider === prov
                  ? "border-purple-700 dark:border-purple-400 bg-purple-700 dark:bg-purple-600 text-white"
                  : "border-border bg-card text-muted-foreground hover:text-foreground hover:border-purple-500"
              }`}
            >
              [ {prov}: {count} ]
            </button>
          ))}
        </div>

        <div className="relative w-full md:w-72">
          <Search className="h-3.5 w-3.5 absolute left-2.5 top-1/2 -translate-y-1/2 text-muted-foreground" />
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder={t.observability.searchPlaceholder}
            className="w-full h-8 bg-card border border-border pl-8 pr-3 font-mono text-[11px] uppercase tracking-[0.05em] text-foreground placeholder:text-muted-foreground focus:outline-none focus:border-purple-600 dark:focus:border-purple-400"
          />
        </div>
      </div>

      {/* High-Density Expandable Telemetry Table */}
      <div className="overflow-x-auto">
        <table className="w-full text-left border-collapse">
          <thead>
            <tr className="border-b-2 border-zinc-900 dark:border-[#3A3A4C] bg-secondary font-mono text-[10px] font-bold uppercase tracking-[0.1em] text-muted-foreground">
              <th className="py-3 pl-5 pr-3 w-36">{t.observability.thTime}</th>
              <th className="py-3 px-3">{t.observability.thProviderModel}</th>
              <th className="py-3 px-3 w-20">{t.observability.thLang}</th>
              <th className="py-3 px-3">{t.observability.thPiiDetected}</th>
              <th className="py-3 pl-3 pr-5">{t.observability.thSecrets}</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-border">
            {filteredTraces.length === 0 ? (
              <tr>
                <td colSpan={5} className="py-16 px-6 text-center font-mono">
                  <p className="text-xs font-bold uppercase tracking-[0.1em] text-foreground">
                    [ {t.observability.noTracesTitle} ]
                  </p>
                  <p className="text-[11px] text-muted-foreground mt-2 uppercase tracking-[0.06em]">
                    {t.observability.noTracesDesc}
                  </p>
                </td>
              </tr>
            ) : (
              filteredTraces.map((trace) => {
                const isExpanded = !!expandedIds[trace.id];
                const { pii, secrets } = splitFallbackTypes(trace);
                const providerLabel = trace.provider || "openai";

                return (
                  <React.Fragment key={trace.id}>
                    <tr
                      onClick={() => toggleRow(trace.id)}
                      className={`cursor-pointer transition-colors ${
                        isExpanded
                          ? "bg-purple-700/10 dark:bg-purple-500/10"
                          : "hover:bg-secondary/80"
                      }`}
                    >
                      {/* TIME + ASCII Tactical Marker */}
                      <td className="py-3.5 pl-5 pr-3 whitespace-nowrap font-mono text-xs">
                        <div className="flex items-center gap-2">
                          <span
                            className={`font-bold text-[10px] w-6 ${
                              isExpanded
                                ? "text-purple-700 dark:text-purple-400"
                                : "text-muted-foreground"
                            }`}
                          >
                            {isExpanded ? "vvv" : ">>>"}
                          </span>
                          <span className="text-foreground font-medium">
                            {formatTime(trace.timestamp)}
                          </span>
                        </div>
                      </td>

                      {/* PROVIDER // MODEL */}
                      <td className="py-3.5 px-3 whitespace-nowrap font-mono">
                        <div className="flex items-center gap-2">
                          <span className="border border-purple-700/70 dark:border-purple-500/70 bg-purple-700/10 dark:bg-purple-500/15 px-1.5 py-0.5 text-[10px] font-bold uppercase tracking-[0.08em] text-purple-800 dark:text-purple-300">
                            {providerLabel}
                          </span>
                          <span className="text-xs font-medium text-foreground">
                            {trace.model || "default"}
                          </span>
                        </div>
                      </td>

                      {/* LANG */}
                      <td className="py-3.5 px-3 whitespace-nowrap font-mono text-xs font-bold uppercase text-foreground">
                        {trace.language || "--"}
                      </td>

                      {/* PII DETECTED */}
                      <td className="py-3.5 px-3">
                        {pii.length > 0 ? (
                          <div className="flex flex-wrap gap-1.5">
                            {pii.map((label) => (
                              <span
                                key={label}
                                className="inline-flex items-center border border-zinc-500 dark:border-[#45455A] bg-secondary px-2 py-0.5 font-mono text-[10px] font-bold uppercase tracking-[0.06em] text-foreground"
                              >
                                {label}
                              </span>
                            ))}
                          </div>
                        ) : (
                          <span className="font-mono text-xs text-muted-foreground">
                            —
                          </span>
                        )}
                      </td>

                      {/* SECRETS */}
                      <td className="py-3.5 pl-3 pr-5">
                        <div className="flex flex-wrap items-center gap-1.5">
                          {secrets.length > 0 ? (
                            secrets.map((label) => (
                              <span
                                key={label}
                                className="inline-flex items-center border border-red-600 dark:border-[#FF2A2A] bg-red-600/10 dark:bg-[#FF2A2A]/15 px-2 py-0.5 font-mono text-[10px] font-bold uppercase tracking-[0.06em] text-red-700 dark:text-[#FF5555]"
                              >
                                {label}
                              </span>
                            ))
                          ) : (
                            <span className="font-mono text-xs text-muted-foreground">
                              —
                            </span>
                          )}
                          {trace.policy_action === "block" && (
                            <span className="inline-flex items-center bg-red-600 px-1.5 py-0.5 font-mono text-[10px] font-bold uppercase tracking-[0.08em] text-white">
                              BLOCKED
                            </span>
                          )}
                        </div>
                      </td>
                    </tr>

                    {/* Expanded MASKED PROMPT Readout */}
                    {isExpanded && (
                      <tr className="bg-secondary/40">
                        <td colSpan={5} className="px-5 pb-5 pt-2">
                          <div className="border-2 border-zinc-900 dark:border-[#3A3A4C] border-l-4 border-l-purple-700 dark:border-l-purple-500 bg-background p-4">
                            <div className="flex flex-wrap items-center justify-between gap-2 pb-2.5 mb-3 border-b border-border font-mono text-[10px] uppercase tracking-[0.08em]">
                              <span className="font-bold text-purple-700 dark:text-purple-400">
                                {t.observability.maskedPromptTitle}
                              </span>
                              <div className="flex flex-wrap items-center gap-3 text-muted-foreground">
                                <span>
                                  SCAN:{" "}
                                  <strong className="text-foreground">
                                    {trace.anon_duration_us} µS
                                  </strong>
                                </span>
                                <span>///</span>
                                <span>
                                  SECURED:{" "}
                                  <strong className="text-purple-700 dark:text-purple-300">
                                    {trace.entities_count}
                                  </strong>
                                </span>
                                {trace.policy_action && (
                                  <>
                                    <span>///</span>
                                    <span>
                                      POLICY:{" "}
                                      <strong className="text-foreground">
                                        {trace.policy_action}
                                      </strong>
                                    </span>
                                  </>
                                )}
                              </div>
                            </div>

                            <output className="block text-sm text-foreground leading-relaxed whitespace-pre-wrap break-words">
                              {renderMaskedPrompt(trace.sanitized_prompt)}
                            </output>
                          </div>
                        </td>
                      </tr>
                    )}
                  </React.Fragment>
                );
              })
            )}
          </tbody>
        </table>
      </div>

      {/* Industrial Brutalist Pagination Controls */}
      {totalCount > 0 && (
        <div className="border-t-2 border-zinc-900 dark:border-[#3A3A4C] px-5 py-3.5 bg-secondary/80 flex flex-wrap items-center justify-between gap-4 font-mono text-[11px] uppercase tracking-[0.06em]">
          <div className="flex items-center gap-3 text-muted-foreground">
            <span>
              {t.observability.paginationShowing}:{" "}
              <strong className="text-foreground">
                {totalCount === 0 ? 0 : startRecord}-{endRecord}
              </strong>{" "}
              {t.common.of}{" "}
              <strong className="text-purple-700 dark:text-purple-400">
                {totalCount}
              </strong>
            </span>

            {/* Page Size Selector */}
            <div className="flex items-center gap-1.5 ml-2 border-l border-border pl-3">
              <span className="hidden sm:inline">{t.common.rowsPerPage}:</span>
              <select
                value={pageSize}
                onChange={(e) => {
                  const newLimit = Number(e.target.value);
                  setPageSize(newLimit);
                  setPage(1);
                  fetchTraces(1, newLimit);
                }}
                className="h-7 bg-card border border-border px-2 font-mono text-[11px] text-foreground focus:outline-none focus:border-purple-500"
              >
                <option value={10}>10</option>
                <option value={25}>25</option>
                <option value={50}>50</option>
                <option value={100}>100</option>
              </select>
            </div>
          </div>

          <div className="flex items-center gap-2">
            <span className="text-muted-foreground mr-1">
              {t.common.page} <strong className="text-foreground">{page}</strong> {t.common.of}{" "}
              <strong className="text-foreground">{totalPages}</strong>
            </span>

            <button
              onClick={() => {
                if (page > 1) {
                  const prevPage = page - 1;
                  setPage(prevPage);
                  fetchTraces(prevPage, pageSize);
                }
              }}
              disabled={page <= 1 || isLoading}
              className="h-7 px-3 border border-border bg-card font-bold hover:border-purple-600 dark:hover:border-purple-400 disabled:opacity-40 disabled:pointer-events-none flex items-center gap-1 transition-colors"
            >
              <ChevronLeft className="h-3.5 w-3.5" />
              <span>{t.common.previous}</span>
            </button>

            <button
              onClick={() => {
                if (page < totalPages) {
                  const nextPage = page + 1;
                  setPage(nextPage);
                  fetchTraces(nextPage, pageSize);
                }
              }}
              disabled={page >= totalPages || isLoading}
              className="h-7 px-3 border border-border bg-card font-bold hover:border-purple-600 dark:hover:border-purple-400 disabled:opacity-40 disabled:pointer-events-none flex items-center gap-1 transition-colors"
            >
              <span>{t.common.next}</span>
              <ChevronRight className="h-3.5 w-3.5" />
            </button>
          </div>
        </div>
      )}
    </section>
  );
}

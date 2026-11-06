"use client";

import React, { useState, useRef } from "react";
import { Play, RefreshCw } from "lucide-react";
import { Button } from "./ui/button";
import { Badge } from "./ui/badge";
import { renderMaskedPrompt } from "./observability";
import type { Language } from "@/lib/i18n";
import { translations } from "@/lib/i18n";

interface PlaygroundProps {
  gatewayUrl: string;
  apiKey: string;
  lang?: Language;
}

const PRESETS = [
  {
    name: "LGPD // CPF + NOME",
    prompt:
      "Envie um e-mail para João da Silva, CPF 529.982.247-25, cobrando a fatura pendente no contato joao.silva@empresa.com.br.",
  },
  {
    name: "SUPPORT // API_KEY (EN)",
    prompt:
      "Please help me respond to the support ticket from Maria Silva at maria.silva@company.com regarding the sk-proj-1234567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890 authentication error.",
  },
  {
    name: "PIX + CONTA BANCÁRIA",
    prompt:
      "Favor transferir o valor para a chave pix 123e4567-e89b-12d3-a456-426614174000 ou no Banco Itaú agência 1234 conta corrente 56789-0.",
  },
  {
    name: "IBAN + IP + E.164",
    prompt:
      "Transferir para IBAN DE89 3704 0044 0532 0130 00 a partir do IP 192.168.15.20. Contato: +1 (415) 555-2671.",
  },
  {
    name: "SECRETS // DB_URI + PWD",
    prompt:
      "Verifique o erro de login com password=MySup3r$ecretPass99 na string postgres://admin:SuperSecretPass123@db.empresa.com:5432/producao_db.",
  },
];

const isSecretCategory = (type: string, subType?: string) => {
  const key = subType || type;
  return (
    key === "API_KEY" ||
    key === "SECRET" ||
    key === "PASSWORD" ||
    key === "JWT" ||
    key === "PRIVATE_KEY" ||
    key === "DB_URI" ||
    key === "SSH_KEY" ||
    key === "CONFIDENTIAL"
  );
};

export function Playground({
  gatewayUrl,
  apiKey,
  lang = "pt-BR",
}: PlaygroundProps) {
  const t = translations[lang];
  const [prompt, setPrompt] = useState(PRESETS[0].prompt);
  const [model, setModel] = useState("mock");
  const [isLoading, setIsLoading] = useState(false);

  const [sanitizedPrompt, setSanitizedPrompt] = useState("");
  const [detectedEntities, setDetectedEntities] = useState<any[]>([]);
  const [streamOutput, setStreamOutput] = useState("");
  const [rawUpstreamPreview, setRawUpstreamPreview] = useState("");
  const [latencyMicros, setLatencyMicros] = useState<number | null>(null);

  const abortControllerRef = useRef<AbortController | null>(null);

  const handleRunStream = async () => {
    if (!prompt.trim()) return;

    setIsLoading(true);
    setStreamOutput("");
    setRawUpstreamPreview("");
    setSanitizedPrompt("");
    setDetectedEntities([]);
    setLatencyMicros(null);

    let localTokenMap: Record<string, string> = {};

    try {
      const inspectHeaders: Record<string, string> = {
        "Content-Type": "application/json",
      };
      if (apiKey) inspectHeaders["Authorization"] = `Bearer ${apiKey}`;

      const inspectRes = await fetch(`${gatewayUrl}/api/test-anonymize`, {
        method: "POST",
        headers: inspectHeaders,
        body: JSON.stringify({ prompt }),
      });
      if (inspectRes.ok) {
        const inspectData = await inspectRes.json();
        setSanitizedPrompt(inspectData.sanitized);
        setDetectedEntities(inspectData.entities || []);
        localTokenMap = inspectData.token_map || {};
        setLatencyMicros(inspectData.latency_micros);
      }
    } catch (e) {
      console.warn("Inspector preflight error:", e);
    }

    const controller = new AbortController();
    abortControllerRef.current = controller;

    try {
      const headers: Record<string, string> = {
        "Content-Type": "application/json",
      };
      if (apiKey) {
        headers["Authorization"] = `Bearer ${apiKey}`;
      }

      const isAnthropicModel = model.startsWith("claude-");
      const isGeminiModel = model.startsWith("gemini-");
      const targetEndpoint = isAnthropicModel
        ? `${gatewayUrl}/api/playground/v1/messages`
        : isGeminiModel
          ? `${gatewayUrl}/api/playground/v1beta/models/${model}:streamGenerateContent?alt=sse`
          : `${gatewayUrl}/api/playground/v1/chat/completions`;

      const reqBody = isAnthropicModel
        ? {
            model,
            max_tokens: 1024,
            messages: [{ role: "user", content: prompt }],
            stream: true,
          }
        : isGeminiModel
          ? {
              contents: [{ role: "user", parts: [{ text: prompt }] }],
            }
          : {
              model,
              messages: [{ role: "user", content: prompt }],
              stream: true,
            };

      const response = await fetch(targetEndpoint, {
        method: "POST",
        headers,
        signal: controller.signal,
        body: JSON.stringify(reqBody),
      });

      if (!response.ok) {
        const errText = await response.text();
        setStreamOutput(`GATEWAY_BLOCK_OR_ERR (${response.status}): ${errText}`);
        setIsLoading(false);
        return;
      }

      const reader = response.body?.getReader();
      const decoder = new TextDecoder("utf-8");

      if (!reader) {
        setIsLoading(false);
        return;
      }

      let done = false;
      let accumulatedClient = "";
      let accumulatedUpstreamSim = "";
      let sseBuffer = "";

      const processLine = (line: string) => {
        const trimmed = line.trim();
        if (!trimmed || trimmed === "data: [DONE]") return;

        if (trimmed.startsWith("data:")) {
          try {
            const jsonPayload = JSON.parse(trimmed.slice(5).trim());
            const geminiParts = jsonPayload.candidates?.[0]?.content?.parts;
            const geminiDelta = Array.isArray(geminiParts)
              ? geminiParts
                  .map((part: { text?: string }) => part.text || "")
                  .join("")
              : "";
            // Suporte unificado SSE para OpenAI, Anthropic e Google Gemini
            const delta =
              jsonPayload.choices?.[0]?.delta?.content ||
              jsonPayload.delta?.text ||
              geminiDelta ||
              "";
            if (!delta) return;

            accumulatedClient += delta;
            setStreamOutput(accumulatedClient);

            let upstreamChunk = delta;
            for (const [tok, orig] of Object.entries(localTokenMap)) {
              if (orig && upstreamChunk.includes(orig)) {
                upstreamChunk = upstreamChunk.replaceAll(orig, tok);
              }
            }
            accumulatedUpstreamSim += upstreamChunk;
            setRawUpstreamPreview(accumulatedUpstreamSim);
          } catch {
            // Keepalive or non-JSON line
          }
        }
      };

      while (!done) {
        const { value, done: streamDone } = await reader.read();
        done = streamDone;
        if (value) {
          sseBuffer += decoder.decode(value, { stream: true });
          const lines = sseBuffer.split("\n");
          sseBuffer = lines.pop() ?? "";
          for (const line of lines) {
            processLine(line);
          }
        }
      }
      if (sseBuffer.trim()) {
        processLine(sseBuffer);
      }
    } catch (err: any) {
      if (err.name !== "AbortError") {
        setStreamOutput(`CONNECTION_ERR: ${err.message}`);
      }
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="space-y-6">
      {/* Header & Tactical Presets */}
      <div className="border-2 border-zinc-900 dark:border-[#3A3A4C] border-l-4 border-l-purple-700 dark:border-l-purple-500 bg-card p-5">
        <div className="flex flex-col lg:flex-row lg:items-center justify-between gap-4">
          <div>
            <h2 className="font-display text-xl uppercase tracking-[-0.04em] text-foreground">
              {t.playground.title}
            </h2>
            <p className="font-mono text-[10px] uppercase tracking-[0.08em] text-muted-foreground mt-1">
              {t.playground.subtitle}
            </p>
          </div>

          <div className="flex flex-wrap items-center gap-1.5">
            {PRESETS.map((p, idx) => (
              <button
                key={idx}
                onClick={() => setPrompt(p.prompt)}
                className="px-2.5 py-1 border border-border bg-secondary hover:border-purple-600 dark:hover:border-purple-400 font-mono text-[10px] font-bold uppercase tracking-[0.06em] text-foreground transition-colors"
              >
                [{p.name}]
              </button>
            ))}
          </div>
        </div>

        {/* Input Area */}
        <div className="mt-5 pt-4 border-t border-border space-y-3">
          <div className="flex flex-wrap items-center justify-between gap-2 font-mono text-[10px] uppercase tracking-[0.1em]">
            <span className="font-bold text-foreground">
              {t.playground.inputPayload}
            </span>
            <div className="flex items-center gap-2">
              <span className="text-muted-foreground">{t.playground.targetModel}</span>
              <select
                value={model}
                onChange={(e) => setModel(e.target.value)}
                className="h-8 bg-background border border-zinc-800 dark:border-[#3A3A4C] px-2.5 font-mono text-[11px] text-foreground focus:outline-none focus:border-purple-500"
              >
                <option value="mock">mock // HADESGATE_SIMULATOR</option>
                <option value="claude-sonnet-4-6">anthropic // claude-sonnet-4-6</option>
                <option value="claude-haiku-4-5-20251001">anthropic // claude-haiku-4-5</option>
                <option value="claude-opus-4-6">anthropic // claude-opus-4-6</option>
                <option value="gemini-2.5-flash">gemini // gemini-2.5-flash</option>
                <option value="gemini-2.5-pro">gemini // gemini-2.5-pro</option>
                <option value="gemini-2.0-flash">gemini // gemini-2.0-flash</option>
                <option value="gpt-4o">openai // gpt-4o</option>
                <option value="gpt-4o-mini">openai // gpt-4o-mini</option>
                <option value="ollama/llama3">ollama // llama3</option>
              </select>
            </div>
          </div>

          <textarea
            value={prompt}
            onChange={(e) => setPrompt(e.target.value)}
            rows={3}
            className="w-full bg-background border border-zinc-800 dark:border-[#3A3A4C] p-3 font-mono text-xs text-foreground placeholder:text-muted-foreground focus:outline-none focus:border-purple-600 dark:focus:border-purple-400"
            placeholder={t.playground.inputPlaceholder}
          />

          <div className="flex flex-wrap items-center justify-between gap-3 pt-1">
            <div className="flex items-center gap-4 font-mono text-[11px] uppercase tracking-[0.08em] text-muted-foreground">
              {latencyMicros !== null && (
                <span>
                  {t.playground.scanTime}{" "}
                  <strong className="text-purple-700 dark:text-purple-400">
                    {latencyMicros} µS
                  </strong>
                </span>
              )}
              {detectedEntities.length > 0 && (
                <span>
                  {t.playground.securedItems}{" "}
                  <strong className="text-foreground">
                    {detectedEntities.length}
                  </strong>
                </span>
              )}
            </div>

            <Button
              variant="purple"
              size="default"
              onClick={handleRunStream}
              disabled={isLoading}
            >
              {isLoading ? (
                <>
                  <RefreshCw className="h-3.5 w-3.5 mr-2 animate-spin" />
                  {t.playground.intercepting}
                </>
              ) : (
                <>
                  <Play className="h-3.5 w-3.5 mr-2 fill-current" />
                  {t.playground.executeInterception}
                </>
              )}
            </Button>
          </div>
        </div>
      </div>

      {/* Dual Telemetry Output Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-px bg-zinc-900 dark:bg-[#3A3A4C] border-2 border-zinc-900 dark:border-[#3A3A4C]">
        {/* Panel 01: Masked Payload */}
        <div className="bg-card p-5 flex flex-col justify-between">
          <div>
            <div className="flex items-center justify-between pb-3 mb-3 border-b border-border">
              <span className="font-mono text-xs font-bold uppercase tracking-[0.08em] text-purple-700 dark:text-purple-400">
                {t.playground.maskedPayloadTitle}
              </span>
              <Badge variant="purple">[ ZERO_PII_LEAK ]</Badge>
            </div>

            <div className="bg-background border border-border p-3.5 text-sm text-foreground min-h-[120px] whitespace-pre-wrap leading-relaxed">
              {sanitizedPrompt ? (
                renderMaskedPrompt(sanitizedPrompt)
              ) : (
                <span className="font-mono text-xs uppercase tracking-[0.06em] text-muted-foreground">
                  {t.common.loading}
                </span>
              )}
            </div>
          </div>

          {detectedEntities.length > 0 && (
            <div className="mt-4 pt-3 border-t border-border">
              <span className="font-mono text-[10px] font-bold uppercase tracking-[0.1em] text-muted-foreground block mb-2">
                {t.playground.detectedTags} ({detectedEntities.length}):
              </span>
              <div className="flex flex-wrap gap-1.5">
                {detectedEntities.map((ent, i) => {
                  const label = ent.sub_type || ent.type;
                  const secret = isSecretCategory(ent.type, ent.sub_type);
                  return (
                    <Badge key={i} variant={secret ? "secret" : "pii"}>
                      {label}
                    </Badge>
                  );
                })}
              </div>
            </div>
          )}
        </div>

        {/* Panel 02: Restored SSE Stream */}
        <div className="bg-card p-5 flex flex-col justify-between">
          <div>
            <div className="flex items-center justify-between pb-3 mb-3 border-b border-border">
              <span className="font-mono text-xs font-bold uppercase tracking-[0.08em] text-foreground">
                {t.playground.reconstitutedStreamTitle}
              </span>
              <Badge variant="outline">[ SLIDING_WINDOW ]</Badge>
            </div>

            <div className="bg-background border border-border p-3.5 font-mono text-xs text-foreground min-h-[120px] whitespace-pre-wrap leading-relaxed">
              {streamOutput ? (
                <div>
                  {streamOutput}
                  {isLoading && (
                    <span className="inline-block w-2 h-3.5 bg-purple-600 ml-1 animate-pulse" />
                  )}
                </div>
              ) : (
                <span className="uppercase tracking-[0.06em] text-muted-foreground">
                  [ AWAITING SSE STREAM... ]
                </span>
              )}
            </div>
          </div>

          {rawUpstreamPreview && (
            <div className="mt-4 pt-3 border-t border-border">
              <span className="font-mono text-[10px] font-bold uppercase tracking-[0.1em] text-muted-foreground block mb-1.5">
                {t.playground.rawUpstreamWire}
              </span>
              <p className="font-mono text-[11px] text-muted-foreground truncate bg-background px-2.5 py-1.5 border border-border">
                {rawUpstreamPreview}
              </p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

"use client";

import React, { useState } from "react";
import { Copy, Check } from "lucide-react";
import type { Language } from "@/lib/i18n";
import { translations } from "@/lib/i18n";

interface CodeSnippetsProps {
  gatewayUrl: string;
  apiKey: string;
  lang?: Language;
}

export function CodeSnippets({
  gatewayUrl,
  apiKey,
  lang = "pt-BR",
}: CodeSnippetsProps) {
  const t = translations[lang];
  const [copiedTab, setCopiedTab] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<
    "python" | "gemini" | "curl" | "node"
  >("python");

  const effectiveKey = apiKey || "hadesgate_proxy_key";

  const snippets = {
    python: `from openai import OpenAI

# Aponte qualquer SDK compatível com OpenAI para o HadesGate
client = OpenAI(
    base_url="${gatewayUrl}/v1",
    api_key="${effectiveKey}"
)

# Dados sensíveis e segredos são mascarados em microssegundos
# antes do envio e restaurados no streaming SSE
response = client.chat.completions.create(
    model="gpt-4o",  # ou "gemini-2.5-flash", "ollama/llama3", "mock"
    messages=[
        {"role": "user", "content": "Cobrar fatura de João da Silva, CPF 529.982.247-25."}
    ],
    stream=True
)

for chunk in response:
    if chunk.choices[0].delta.content:
        print(chunk.choices[0].delta.content, end="", flush=True)`,

    gemini: `from google import genai
from google.genai import types

# SDK nativo Google GenAI apontando para o HadesGate (/v1beta/models/*)
client = genai.Client(
    api_key="${effectiveKey}",
    http_options=types.HttpOptions(base_url="${gatewayUrl}"),
)

for chunk in client.models.generate_content_stream(
    model="gemini-2.5-flash",
    contents="Cobrar fatura de João da Silva, CPF 529.982.247-25.",
):
    print(chunk.text or "", end="", flush=True)`,

    curl: `curl ${gatewayUrl}/v1/chat/completions \\
  -H "Authorization: Bearer ${effectiveKey}" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "gpt-4o",
    "messages": [
      {"role": "user", "content": "Envie e-mail para João da Silva, CPF 529.982.247-25."}
    ],
    "stream": true
  }'`,

    node: `import OpenAI from "openai";

const client = new OpenAI({
  baseURL: "${gatewayUrl}/v1",
  apiKey: "${effectiveKey}",
});

async function main() {
  const stream = await client.chat.completions.create({
    model: "gpt-4o",
    messages: [{ role: "user", content: "Notificar João da Silva, CPF 529.982.247-25." }],
    stream: true,
  });

  for await (const chunk of stream) {
    process.stdout.write(chunk.choices[0]?.delta?.content || "");
  }
}

main();`,
  };

  const copyToClipboard = (text: string, tab: string) => {
    navigator.clipboard.writeText(text);
    setCopiedTab(tab);
    setTimeout(() => setCopiedTab(null), 2000);
  };

  return (
    <div className="border-2 border-zinc-900 dark:border-[#3A3A4C] bg-card">
      <div className="px-5 py-4 border-b-2 border-zinc-900 dark:border-[#3A3A4C] border-l-4 border-l-purple-700 dark:border-l-purple-500 flex flex-col sm:flex-row sm:items-center justify-between gap-3">
        <div>
          <h2 className="font-display text-xl uppercase tracking-[-0.04em] text-foreground">
            {t.codeSnippets.title}
          </h2>
          <p className="font-mono text-[10px] uppercase tracking-[0.08em] text-muted-foreground mt-1">
            {t.codeSnippets.subtitle}
          </p>
        </div>

        <div className="flex flex-wrap divide-x divide-border border border-zinc-800 dark:border-[#3A3A4C] self-start">
          {(["python", "gemini", "curl", "node"] as const).map((tab) => (
            <button
              key={tab}
              onClick={() => setActiveTab(tab)}
              className={`px-3.5 py-1.5 font-mono text-[10px] font-bold uppercase tracking-[0.08em] transition-colors ${
                activeTab === tab
                  ? "bg-purple-700 dark:bg-purple-600 text-white"
                  : "bg-secondary text-muted-foreground hover:text-foreground"
              }`}
            >
              {tab === "python"
                ? "PYTHON"
                : tab === "gemini"
                  ? t.codeSnippets.tabGemini
                  : tab === "curl"
                    ? "CURL"
                    : "NODE.JS"}
            </button>
          ))}
        </div>
      </div>

      <div className="relative bg-[#0D0D11] text-[#EAEAEA] p-5">
        <div className="absolute top-4 right-4">
          <button
            onClick={() => copyToClipboard(snippets[activeTab], activeTab)}
            className="flex items-center gap-1.5 px-3 py-1.5 border border-[#3A3A4C] bg-[#16161E] hover:border-purple-500 font-mono text-[10px] font-bold uppercase tracking-[0.08em] text-[#EAEAEA] transition-colors"
          >
            {copiedTab === activeTab ? (
              <>
                <Check className="h-3 w-3 text-purple-400" /> {t.codeSnippets.copied}
              </>
            ) : (
              <>
                <Copy className="h-3 w-3 text-purple-400" /> {t.codeSnippets.copy}
              </>
            )}
          </button>
        </div>

        <pre className="font-mono text-xs overflow-x-auto leading-relaxed pr-28">
          <code>{snippets[activeTab]}</code>
        </pre>
      </div>
    </div>
  );
}

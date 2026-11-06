"use client";

import React, { useState } from "react";
import { Eye, EyeOff, ArrowRight, AlertTriangle, RefreshCw, Sun, Moon, Globe } from "lucide-react";
import { Button } from "./ui/button";
import { HadesGateLogo } from "./hadesgate-logo";
import type { ThemeMode } from "@/app/page";
import type { Language } from "@/lib/i18n";
import { translations } from "@/lib/i18n";

interface LoginProps {
  initialGatewayUrl: string;
  onLoginSuccess: (url: string, key: string) => void;
  theme: ThemeMode;
  onToggleTheme: () => void;
  lang: Language;
  onToggleLang: () => void;
}

export function Login({
  initialGatewayUrl,
  onLoginSuccess,
  theme,
  onToggleTheme,
  lang,
  onToggleLang,
}: LoginProps) {
  const t = translations[lang];
  const [gatewayUrl, setGatewayUrl] = useState(
    initialGatewayUrl || "http://localhost:8080"
  );
  const [apiKey, setApiKey] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [errorMessage, setErrorMessage] = useState("");

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    const targetUrl = gatewayUrl.trim().replace(/\/$/, "");
    const trimmedKey = apiKey.trim();

    if (!targetUrl || !trimmedKey) {
      setErrorMessage(
        lang === "pt-BR"
          ? "INFORME A URL DO GATEWAY E A CHAVE DE ACESSO."
          : "ENTER GATEWAY URL AND ACCESS KEY."
      );
      return;
    }

    setIsLoading(true);
    setErrorMessage("");

    try {
      const res = await fetch(`${targetUrl}/api/auth/verify`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ key: trimmedKey }),
      });

      if (!res.ok) {
        throw new Error(`CONNECTION_ERR (${res.status} ${res.statusText})`);
      }

      const data = await res.json();
      if (data.valid) {
        onLoginSuccess(targetUrl, trimmedKey);
      } else {
        setErrorMessage("CREDENCIAIS INVÁLIDAS PARA ESTA UNIDADE HADESGATE.");
      }
    } catch (err: any) {
      setErrorMessage(
        err.message || "FALHA DE CONEXÃO COM O GATEWAY. VERIFIQUE A URL."
      );
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex flex-col justify-between p-4 sm:p-8 bg-background text-foreground">
      {/* Top Bar with Language + Theme Switcher */}
      <div className="max-w-7xl w-full mx-auto flex items-center justify-between font-mono text-[10px] uppercase tracking-[0.12em] text-muted-foreground">
        <span>{t.login.clearanceRequired}</span>
        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={onToggleLang}
            className="h-8 px-3 border border-zinc-800 dark:border-[#38384A] bg-card hover:border-purple-600 dark:hover:border-purple-400 font-bold text-foreground flex items-center gap-1.5"
          >
            <Globe className="h-3.5 w-3.5 text-purple-600 dark:text-purple-400" />
            <span>{lang === "pt-BR" ? "PT-BR" : "EN"}</span>
          </button>
          <button
            type="button"
            onClick={onToggleTheme}
            className="h-8 px-3 border border-zinc-800 dark:border-[#38384A] bg-card hover:border-purple-600 dark:hover:border-purple-400 font-bold text-foreground flex items-center gap-1.5"
          >
            {theme === "dark" ? (
              <>
                <Sun className="h-3.5 w-3.5 text-purple-400" />
                <span>{t.common.light}</span>
              </>
            ) : (
              <>
                <Moon className="h-3.5 w-3.5 text-purple-700" />
                <span>{t.common.dark}</span>
              </>
            )}
          </button>
        </div>
      </div>

      {/* Central Brutalist Authentication & Tactical Hologram Terminal */}
      <div className="my-auto mx-auto w-full max-w-4xl relative border-2 border-zinc-900 dark:border-[#3A3A4C] bg-card shadow-[10px_10px_0px_rgba(0,0,0,0.85)]">
        {/* Purple Top Structural Bar */}
        <div className="h-2 w-full bg-purple-700 dark:bg-purple-600" />

        <div className="grid grid-cols-1 lg:grid-cols-12 divide-y-2 lg:divide-y-0 lg:divide-x-2 divide-zinc-900 dark:divide-[#3A3A4C]">
          {/* Left Column: Form & Brand */}
          <div className="lg:col-span-7 flex flex-col justify-between">
            <div className="p-6 sm:p-8 border-b border-border brutalist-hatch">
              <div className="flex items-center justify-between font-mono text-[10px] uppercase tracking-[0.12em] text-muted-foreground mb-3">
                <span>{t.login.authUnit}</span>
                <span className="text-purple-700 dark:text-purple-400 font-bold">
                  {t.login.zeroLeakShield}
                </span>
              </div>
              <div className="flex items-center gap-3.5">
                <HadesGateLogo className="h-9 w-auto shrink-0 text-zinc-950 dark:text-white" />
                <h1 className="font-display text-4xl uppercase tracking-[-0.05em] leading-[0.88] text-foreground">
                  HADESGATE<span className="text-purple-700 dark:text-purple-400">™</span>
                </h1>
              </div>
              <p className="mt-2 font-mono text-[11px] uppercase tracking-[0.06em] text-muted-foreground">
                {t.login.subheading}
              </p>
            </div>

            <div className="p-6 sm:p-8 flex-1 flex flex-col justify-center">
              {errorMessage && (
                <div className="mb-5 p-3 border border-red-600 dark:border-[#FF2A2A] bg-red-600/10 dark:bg-[#FF2A2A]/15 text-red-700 dark:text-[#FF5555] font-mono text-[11px] font-bold uppercase tracking-[0.05em] flex items-start gap-2">
                  <AlertTriangle className="h-4 w-4 shrink-0 mt-0.5" />
                  <span>{errorMessage}</span>
                </div>
              )}

              <form onSubmit={handleSubmit} className="space-y-5">
                <div className="space-y-1.5">
                  <label className="font-mono text-[10px] font-bold uppercase tracking-[0.1em] text-foreground flex items-center justify-between">
                    <span>{t.login.gatewayEndpoint}</span>
                    <span className="text-muted-foreground">[REQ]</span>
                  </label>
                  <input
                    type="text"
                    required
                    value={gatewayUrl}
                    onChange={(e) => setGatewayUrl(e.target.value)}
                    placeholder="http://localhost:8080"
                    className="w-full h-10 bg-background border border-zinc-800 dark:border-[#3A3A4C] px-3 font-mono text-xs text-foreground placeholder:text-muted-foreground focus:outline-none focus:border-purple-600 dark:focus:border-purple-400"
                  />
                </div>

                <div className="space-y-1.5">
                  <label className="font-mono text-[10px] font-bold uppercase tracking-[0.1em] text-foreground flex items-center justify-between">
                    <span>{t.login.adminKey}</span>
                    <span className="text-purple-700 dark:text-purple-400">[SECRET]</span>
                  </label>
                  <div className="relative">
                    <input
                      type={showPassword ? "text" : "password"}
                      required
                      value={apiKey}
                      onChange={(e) => setApiKey(e.target.value)}
                      placeholder="hadesgate_sec_..."
                      className="w-full h-10 bg-background border border-zinc-800 dark:border-[#3A3A4C] px-3 pr-10 font-mono text-xs text-foreground placeholder:text-muted-foreground focus:outline-none focus:border-purple-600 dark:focus:border-purple-400"
                    />
                    <button
                      type="button"
                      onClick={() => setShowPassword(!showPassword)}
                      className="absolute right-2.5 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                    >
                      {showPassword ? (
                        <EyeOff className="h-4 w-4" />
                      ) : (
                        <Eye className="h-4 w-4" />
                      )}
                    </button>
                  </div>
                </div>

                <Button
                  type="submit"
                  variant="purple"
                  size="lg"
                  disabled={isLoading}
                  className="w-full"
                >
                  {isLoading ? (
                    <>
                      <RefreshCw className="h-3.5 w-3.5 mr-2 animate-spin" />
                      {t.login.authenticating}
                    </>
                  ) : (
                    <>
                      {t.login.enterConsole}
                      <ArrowRight className="h-3.5 w-3.5 ml-2" />
                    </>
                  )}
                </Button>
              </form>
            </div>

            {/* Micro Telemetry Bar */}
            <div className="px-6 py-2.5 border-t border-border bg-secondary/40 flex items-center justify-between font-mono text-[9px] uppercase tracking-[0.08em] text-muted-foreground">
              <span>LATÊNCIA: &lt; 1MS</span>
              <span className="text-purple-600 dark:text-purple-400 font-bold">SHA-256 VAULT</span>
            </div>
          </div>

          {/* Right Column: Hades Hologram Display */}
          <div className="lg:col-span-5 relative bg-[#0c0c10] dark:bg-[#07070a] p-6 flex flex-col justify-between overflow-hidden select-none min-h-[340px] lg:min-h-full">
            {/* Tactical Crosshairs at the 4 corners */}
            <span className="absolute top-2 left-2.5 font-mono text-xs font-bold text-purple-500/50 z-20">+</span>
            <span className="absolute top-2 right-2.5 font-mono text-xs font-bold text-purple-500/50 z-20">+</span>
            <span className="absolute bottom-2 left-2.5 font-mono text-xs font-bold text-purple-500/50 z-20">+</span>
            <span className="absolute bottom-2 right-2.5 font-mono text-xs font-bold text-purple-500/50 z-20">+</span>

            {/* Background Holographic Grid Pattern */}
            <div className="absolute inset-0 hologram-grid opacity-60 pointer-events-none" />

            {/* Glowing Holographic Radial Core */}
            <div
              className="absolute inset-0 pointer-events-none"
              style={{
                background:
                  "radial-gradient(ellipse at 50% 55%, rgba(147, 51, 234, 0.28) 0%, rgba(147, 51, 234, 0.08) 50%, transparent 75%)",
              }}
            />

            {/* Side Shield Visual Header */}
            <div className="relative z-10 flex items-center justify-between border-b border-purple-500/25 pb-2.5 font-mono text-[9px] uppercase tracking-[0.1em]">
              <span className="text-purple-400 font-bold flex items-center gap-1.5">
                <span className="inline-block h-1.5 w-1.5 rounded-full bg-purple-500 animate-pulse" />
                {t.login.hologramUnit}
              </span>
              <span className="text-zinc-500">[ONLINE]</span>
            </div>

            {/* Center: Shield Visual Representation */}
            <div className="relative z-10 my-4 flex flex-col items-center justify-center">
              <div className="relative w-full max-w-[260px] aspect-[4/5] flex items-center justify-center">
                {/* Shield Visual Image */}
                <img
                  src="/hades.webp"
                  alt="HadesGate Privacy Shield"
                  className="w-full h-full object-contain filter drop-shadow-[0_0_24px_rgba(147,51,234,0.45)] brightness-105 contrast-110"
                  loading="eager"
                />

                {/* Scanning Beam (Light sweep line animating vertically) */}
                <div className="absolute left-0 right-0 h-[2px] bg-gradient-to-r from-transparent via-purple-400 to-transparent shadow-[0_0_10px_#a855f7] animate-hologram-sweep pointer-events-none z-20" />

                {/* CRT Scanline Overlay specifically on the visual panel */}
                <div className="absolute inset-0 hologram-scanlines opacity-40 pointer-events-none z-10" />
              </div>
            </div>

            {/* Shield Bottom Telemetry Readout */}
            <div className="relative z-10 border-t border-purple-500/25 pt-2.5 space-y-1 font-mono text-[9px] uppercase tracking-[0.08em]">
              <div className="flex items-center justify-between text-zinc-400">
                <span>{t.login.hologramStatus}</span>
                <span className="text-purple-400 font-bold">[ARMED]</span>
              </div>
              <div className="text-[8px] text-zinc-500 truncate">
                {t.login.hologramProtocol}
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Bottom Industrial Stamp */}
      <div className="max-w-7xl w-full mx-auto flex items-center justify-between font-mono text-[10px] uppercase tracking-[0.12em] text-muted-foreground">
        <span>||| || |||| | ||| |||| ||</span>
        <div className="flex items-center gap-3">
          <span>REV 2.0 // {t.login.deterministicEngine}</span>
          <a
            href="https://github.com/rodolfoportalles/HadesGate"
            target="_blank"
            rel="noreferrer"
            className="font-bold text-purple-500 hover:text-purple-400 underline underline-offset-2"
          >
            SOURCE // AGPL-3.0
          </a>
        </div>
      </div>
    </div>
  );
}

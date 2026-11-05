"use client";

import React, { useState, useEffect } from "react";
import { Navbar } from "@/components/navbar";
import { Login } from "@/components/login";
import { Observability } from "@/components/observability";
import { Playground } from "@/components/playground";
import { CodeSnippets } from "@/components/code-snippets";
import { Settings } from "@/components/settings";
import { SystemToast, type SystemNotification } from "@/components/ui/system-toast";
import type { Language } from "@/lib/i18n";

export type ThemeMode = "dark" | "light";

export default function Home() {
  const [isAuthenticated, setIsAuthenticated] = useState<boolean>(false);
  const [isCheckingAuth, setIsCheckingAuth] = useState<boolean>(true);
  const [gatewayUrl, setGatewayUrl] = useState<string>("http://localhost:8080");
  const [apiKey, setApiKey] = useState<string>("");
  const [activeTab, setActiveTab] = useState<
    "observability" | "playground" | "integration" | "settings"
  >("observability");
  const [theme, setTheme] = useState<ThemeMode>("dark");
  const [lang, setLang] = useState<Language>("pt-BR");
  const [systemNotif, setSystemNotif] = useState<SystemNotification | null>(null);

  useEffect(() => {
    if (typeof window !== "undefined") {
      // 1. Load Language (default PT-BR)
      const savedLang = localStorage.getItem("hadesgate_lang") as Language | null;
      if (savedLang === "en" || savedLang === "pt-BR") {
        setLang(savedLang);
      }

      // 2. Load Theme (default DARK)
      const savedTheme = localStorage.getItem("hadesgate_theme") as ThemeMode | null;
      const resolvedTheme: ThemeMode = savedTheme === "light" ? "light" : "dark";
      setTheme(resolvedTheme);
      document.documentElement.classList.toggle("dark", resolvedTheme === "dark");

      // 2. Load Auth (supports hadesgate_* with fallback to legacy privagate_*)
      const storedAuth =
        sessionStorage.getItem("hadesgate_auth") ||
        sessionStorage.getItem("privagate_auth");
      const storedUrl =
        sessionStorage.getItem("hadesgate_url") ||
        sessionStorage.getItem("privagate_url");
      const storedKey =
        sessionStorage.getItem("hadesgate_key") ||
        sessionStorage.getItem("privagate_key");

      let initialUrl = "http://localhost:8080";
      if (window.location.origin && !window.location.origin.includes(":3000")) {
        initialUrl = window.location.origin;
      }

      setGatewayUrl(storedUrl || initialUrl);
      if (storedKey) setApiKey(storedKey);

      if (
        storedAuth === "true" &&
        storedKey &&
        storedKey.trim().length > 0 &&
        storedUrl &&
        storedUrl.trim().length > 0
      ) {
        setIsAuthenticated(true);
      } else {
        setIsAuthenticated(false);
        sessionStorage.removeItem("hadesgate_auth");
        sessionStorage.removeItem("hadesgate_key");
      }
    }
    setIsCheckingAuth(false);
  }, []);

  const handleToggleTheme = () => {
    const nextTheme: ThemeMode = theme === "dark" ? "light" : "dark";
    setTheme(nextTheme);
    if (typeof window !== "undefined") {
      localStorage.setItem("hadesgate_theme", nextTheme);
      document.documentElement.classList.toggle("dark", nextTheme === "dark");
    }
  };

  const handleToggleLang = () => {
    const nextLang: Language = lang === "pt-BR" ? "en" : "pt-BR";
    setLang(nextLang);
    if (typeof window !== "undefined") {
      localStorage.setItem("hadesgate_lang", nextLang);
    }
  };

  const handleSelectLang = (newLang: Language) => {
    setLang(newLang);
    if (typeof window !== "undefined") {
      localStorage.setItem("hadesgate_lang", newLang);
    }
  };

  const handleLoginSuccess = (url: string, key: string) => {
    setGatewayUrl(url);
    setApiKey(key);
    setIsAuthenticated(true);

    if (typeof window !== "undefined") {
      sessionStorage.setItem("hadesgate_auth", "true");
      sessionStorage.setItem("hadesgate_url", url);
      sessionStorage.setItem("hadesgate_key", key);
    }
  };

  const handleLogout = () => {
    setIsAuthenticated(false);
    setApiKey("");
    if (typeof window !== "undefined") {
      sessionStorage.removeItem("hadesgate_auth");
      sessionStorage.removeItem("hadesgate_key");
      sessionStorage.removeItem("privagate_auth");
      sessionStorage.removeItem("privagate_key");
    }
  };

  if (isCheckingAuth) {
    return (
      <div className="min-h-screen bg-background flex items-center justify-center font-mono text-xs uppercase tracking-[0.12em] text-muted-foreground">
        <span>[ INITIALIZING HADESGATE TELEMETRY... ]</span>
      </div>
    );
  }

  if (!isAuthenticated) {
    return (
      <Login
        initialGatewayUrl={gatewayUrl}
        onLoginSuccess={handleLoginSuccess}
        theme={theme}
        onToggleTheme={handleToggleTheme}
        lang={lang}
        onToggleLang={handleToggleLang}
      />
    );
  }

  return (
    <div className="min-h-screen flex flex-col bg-background text-foreground">
      <SystemToast
        notification={systemNotif}
        onClose={() => setSystemNotif(null)}
      />
      <Navbar
        activeTab={activeTab}
        onTabChange={setActiveTab}
        gatewayUrl={gatewayUrl}
        onLogout={handleLogout}
        theme={theme}
        onToggleTheme={handleToggleTheme}
        lang={lang}
        onToggleLang={handleToggleLang}
        onSelectLang={handleSelectLang}
      />

      <main className="flex-1 max-w-7xl w-full mx-auto px-4 sm:px-6 py-8">
        {activeTab === "observability" && (
          <Observability gatewayUrl={gatewayUrl} apiKey={apiKey} lang={lang} />
        )}

        {activeTab === "playground" && (
          <Playground gatewayUrl={gatewayUrl} apiKey={apiKey} lang={lang} />
        )}

        {activeTab === "integration" && (
          <CodeSnippets gatewayUrl={gatewayUrl} apiKey={apiKey} lang={lang} />
        )}

        {activeTab === "settings" && (
          <Settings
            gatewayUrl={gatewayUrl}
            apiKey={apiKey}
            lang={lang}
            onNotify={setSystemNotif}
          />
        )}
      </main>

      <footer className="border-t-2 border-zinc-900 dark:border-[#343444] bg-card py-3.5 px-6 font-mono text-[10px] uppercase tracking-[0.1em] text-muted-foreground">
        <div className="max-w-7xl mx-auto flex flex-col sm:flex-row items-center justify-between gap-2">
          <div className="flex items-center gap-2">
            <span className="inline-block h-2 w-2 bg-purple-600" />
            <strong className="text-foreground">HADESGATE™</strong>
            <span>/// DETERMINISTIC PII &amp; SECRET ANONYMIZATION GATEWAY</span>
          </div>
          <div>
            <span>REV 2.0 // SQLITE_ZERO_PII // GO_STATIC_BINARY</span>
          </div>
        </div>
      </footer>
    </div>
  );
}

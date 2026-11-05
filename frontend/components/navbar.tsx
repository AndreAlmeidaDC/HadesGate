"use client";

import React, { useState, useRef, useEffect } from "react";
import { Sun, Moon, LogOut, Settings, ChevronDown } from "lucide-react";
import type { ThemeMode } from "@/app/page";
import type { Language } from "@/lib/i18n";
import { translations } from "@/lib/i18n";
import { HadesGateLogo } from "./hadesgate-logo";

interface NavbarProps {
  activeTab: "observability" | "playground" | "integration" | "settings";
  onTabChange: (tab: "observability" | "playground" | "integration" | "settings") => void;
  gatewayUrl: string;
  onLogout: () => void;
  theme: ThemeMode;
  onToggleTheme: () => void;
  lang: Language;
  onToggleLang?: () => void;
  onSelectLang?: (lang: Language) => void;
}

export function Navbar({
  activeTab,
  onTabChange,
  onLogout,
  theme,
  onToggleTheme,
  lang,
  onToggleLang,
  onSelectLang,
}: NavbarProps) {
  const t = translations[lang];
  const [langMenuOpen, setLangMenuOpen] = useState(false);
  const langMenuRef = useRef<HTMLDivElement>(null);

  const navItems: Array<{
    id: "observability" | "playground" | "integration";
    code: string;
    label: string;
  }> = [
    { id: "observability", code: "01", label: t.common.telemetry },
    { id: "playground", code: "02", label: t.common.playground },
    { id: "integration", code: "03", label: t.common.sdkIntegration },
  ];

  // Close language dropdown when clicking outside
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (
        langMenuRef.current &&
        !langMenuRef.current.contains(event.target as Node)
      ) {
        setLangMenuOpen(false);
      }
    }
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  const handleChooseLang = (target: Language) => {
    if (onSelectLang) {
      onSelectLang(target);
    } else if (onToggleLang && target !== lang) {
      onToggleLang();
    }
    setLangMenuOpen(false);
  };

  return (
    <header className="border-b-2 border-zinc-900 dark:border-[#343444] bg-card sticky top-0 z-40">
      <div className="max-w-7xl mx-auto px-3 sm:px-6 h-14 flex items-stretch justify-between gap-2 overflow-x-auto sm:overflow-visible scrollbar-none">
        {/* Brand Block */}
        <div className="flex items-center gap-2.5 sm:gap-3 py-2 pr-3 sm:pr-4 border-r border-border shrink-0">
          <HadesGateLogo className="h-6 sm:h-7 w-auto shrink-0 text-zinc-950 dark:text-white" />
          <div className="leading-none">
            <div className="flex items-center gap-1.5">
              <span className="font-display text-sm sm:text-base uppercase tracking-[-0.04em] text-foreground whitespace-nowrap">
                HADESGATE<sup className="text-[9px] font-mono ml-0.5 text-purple-600 dark:text-purple-400">™</sup>
              </span>
            </div>
            <span className="hidden md:block font-mono text-[9px] uppercase tracking-[0.12em] text-muted-foreground mt-0.5 whitespace-nowrap">
              UNIT // ZERO-LEAK SHIELD
            </span>
          </div>
        </div>

        {/* Mechanical Segmented Navigation (Main Tabs: 01, 02, 03) */}
        <nav className="flex items-stretch divide-x divide-border border-x border-border shrink-0">
          {navItems.map((item) => {
            const active = activeTab === item.id;
            return (
              <button
                key={item.id}
                onClick={() => onTabChange(item.id)}
                className={`px-3 sm:px-4 md:px-5 py-2 font-mono text-[11px] font-bold uppercase tracking-[0.08em] transition-colors flex items-center gap-1.5 sm:gap-2 whitespace-nowrap ${
                  active
                    ? "bg-purple-700 dark:bg-purple-600 text-white"
                    : "bg-transparent text-muted-foreground hover:text-foreground hover:bg-secondary"
                }`}
              >
                <span
                  className={
                    active
                      ? "text-purple-200"
                      : "text-purple-700 dark:text-purple-400"
                  }
                >
                  [{item.code}]
                </span>
                <span>{item.label}</span>
              </button>
            );
          })}
        </nav>

        {/* Right Corner: Connected Tactical Button Group (No Gaps) */}
        <div className="flex items-center py-2 pl-2 shrink-0">
          <div className="flex items-stretch border border-zinc-800 dark:border-[#38384A] bg-secondary divide-x divide-zinc-800 dark:divide-[#38384A] h-9">
            {/* 1. Settings / Policies Gear Icon */}
            <button
              onClick={() => onTabChange("settings")}
              title={t.common.settings}
              aria-label={t.common.settings}
              className={`w-9 h-full flex items-center justify-center transition-colors ${
                activeTab === "settings"
                  ? "bg-purple-700 dark:bg-purple-600 text-white"
                  : "bg-transparent text-muted-foreground hover:text-foreground hover:bg-secondary/80"
              }`}
            >
              <Settings
                className={`h-4 w-4 transition-transform duration-200 ${
                  activeTab === "settings" ? "rotate-45" : "hover:rotate-45"
                }`}
              />
            </button>

            {/* 2. Language Dropdown (BR / EN) */}
            <div className="relative" ref={langMenuRef}>
              <button
                onClick={() => setLangMenuOpen((prev) => !prev)}
                title={lang === "pt-BR" ? "Idioma (Português)" : "Language (English)"}
                className="h-full px-2.5 flex items-center gap-1 font-mono text-[10px] font-bold uppercase tracking-wider text-foreground hover:bg-secondary/80 transition-colors"
              >
                <span>{lang === "pt-BR" ? "BR" : "EN"}</span>
                <ChevronDown
                  className={`h-3 w-3 text-muted-foreground transition-transform duration-150 ${
                    langMenuOpen ? "rotate-180" : ""
                  }`}
                />
              </button>

              {langMenuOpen && (
                <div className="absolute right-0 top-full mt-1 w-32 border-2 border-zinc-900 dark:border-[#38384A] bg-card shadow-[4px_4px_0px_rgba(0,0,0,0.85)] z-50 divide-y divide-border font-mono text-xs">
                  <button
                    onClick={() => handleChooseLang("pt-BR")}
                    className={`w-full text-left px-3 py-2 flex items-center justify-between uppercase transition-colors ${
                      lang === "pt-BR"
                        ? "bg-purple-950/40 text-purple-400 font-bold"
                        : "hover:bg-secondary text-foreground"
                    }`}
                  >
                    <span>[BR] PT-BR</span>
                    {lang === "pt-BR" && <span className="text-[10px]">✓</span>}
                  </button>
                  <button
                    onClick={() => handleChooseLang("en")}
                    className={`w-full text-left px-3 py-2 flex items-center justify-between uppercase transition-colors ${
                      lang === "en"
                        ? "bg-purple-950/40 text-purple-400 font-bold"
                        : "hover:bg-secondary text-foreground"
                    }`}
                  >
                    <span>[EN] ENGLISH</span>
                    {lang === "en" && <span className="text-[10px]">✓</span>}
                  </button>
                </div>
              )}
            </div>

            {/* 3. Theme Toggle Icon (Sun / Moon) */}
            <button
              onClick={onToggleTheme}
              title={
                theme === "dark"
                  ? "Alternar para Modo Claro"
                  : "Alternar para Modo Escuro"
              }
              aria-label={
                theme === "dark"
                  ? "Alternar para Modo Claro"
                  : "Alternar para Modo Escuro"
              }
              className="w-9 h-full flex items-center justify-center text-foreground hover:bg-secondary/80 transition-colors"
            >
              {theme === "dark" ? (
                <Sun className="h-4 w-4 text-purple-400" />
              ) : (
                <Moon className="h-4 w-4 text-purple-700" />
              )}
            </button>

            {/* 4. Logout Icon (Turns red on hover) */}
            <button
              onClick={onLogout}
              title={t.common.exit}
              aria-label={t.common.exit}
              className="w-9 h-full flex items-center justify-center text-muted-foreground hover:bg-red-600 hover:text-white transition-colors"
            >
              <LogOut className="h-4 w-4" />
            </button>
          </div>
        </div>
      </div>
    </header>
  );
}

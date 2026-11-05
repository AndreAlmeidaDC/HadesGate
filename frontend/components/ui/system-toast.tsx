"use client";

import React, { useEffect } from "react";
import { CheckCircle2, AlertTriangle, X, Terminal } from "lucide-react";

export interface SystemNotification {
  type: "success" | "error" | "info";
  title?: string;
  message: string;
  timestamp?: string;
}

interface SystemToastProps {
  notification: SystemNotification | null;
  onClose: () => void;
  durationMs?: number;
}

export function SystemToast({
  notification,
  onClose,
  durationMs = 4500,
}: SystemToastProps) {
  useEffect(() => {
    if (!notification) return;
    const timer = setTimeout(() => {
      onClose();
    }, durationMs);
    return () => clearTimeout(timer);
  }, [notification, onClose, durationMs]);

  if (!notification) return null;

  const isSuccess = notification.type === "success";
  const isError = notification.type === "error";

  const borderColor = isSuccess
    ? "border-green-600 border-l-[6px] border-l-green-500"
    : isError
    ? "border-red-600 border-l-[6px] border-l-red-500"
    : "border-purple-600 border-l-[6px] border-l-purple-500";

  const timeString =
    notification.timestamp ||
    new Date().toLocaleTimeString("pt-BR", { hour12: false });

  return (
    <div
      role="alert"
      aria-live="assertive"
      className="fixed bottom-6 right-4 sm:right-6 z-50 max-w-md w-[calc(100%-2rem)] sm:w-auto min-w-[340px] pointer-events-auto transition-all animate-in fade-in slide-in-from-bottom-3 duration-150"
    >
      <div
        className={`border-2 bg-zinc-950 text-white shadow-[6px_6px_0px_rgba(0,0,0,0.85)] ${borderColor}`}
      >
        {/* Header Bar */}
        <div className="px-3 py-1.5 border-b border-zinc-800 bg-zinc-900/90 flex items-center justify-between gap-3">
          <div className="flex items-center gap-1.5 font-mono text-[10px] font-bold uppercase tracking-[0.1em] text-zinc-300">
            <Terminal className="h-3.5 w-3.5 text-purple-400" />
            <span>
              {notification.title || "HADESGATE // NOTIFICAÇÃO DO SISTEMA"}
            </span>
          </div>

          <div className="flex items-center gap-2">
            <span className="font-mono text-[9px] text-zinc-500 tracking-wider">
              [{timeString}]
            </span>
            <button
              onClick={onClose}
              title="Fechar (Esc)"
              className="p-0.5 text-zinc-400 hover:text-white hover:bg-zinc-800 transition-colors"
            >
              <X className="h-3.5 w-3.5" />
            </button>
          </div>
        </div>

        {/* Content Body */}
        <div className="p-3 flex items-start gap-2.5">
          {isSuccess ? (
            <CheckCircle2 className="h-4 w-4 shrink-0 text-green-400 mt-0.5" />
          ) : isError ? (
            <AlertTriangle className="h-4 w-4 shrink-0 text-red-400 mt-0.5" />
          ) : (
            <Terminal className="h-4 w-4 shrink-0 text-purple-400 mt-0.5" />
          )}

          <div className="flex-1 font-mono text-xs font-bold uppercase tracking-[0.06em] leading-snug">
            <span
              className={
                isSuccess
                  ? "text-green-400"
                  : isError
                  ? "text-red-400"
                  : "text-purple-300"
              }
            >
              {notification.message}
            </span>
          </div>
        </div>
      </div>
    </div>
  );
}


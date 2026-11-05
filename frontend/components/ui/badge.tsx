import * as React from "react";
import { cn } from "@/lib/utils";

export interface BadgeProps extends React.HTMLAttributes<HTMLDivElement> {
  className?: string;
  variant?:
    | "default"
    | "secondary"
    | "outline"
    | "purple"
    | "pii"
    | "entity"
    | "secret"
    | "api_key"
    | "confidential"
    | "blocked";
}

function Badge({ className, variant = "default", ...props }: BadgeProps) {
  const variants: Record<string, string> = {
    default:
      "border border-border bg-secondary text-secondary-foreground",
    secondary:
      "border border-border bg-muted text-muted-foreground",
    outline:
      "border border-border bg-transparent text-foreground",
    purple:
      "border border-purple-700 dark:border-purple-500 bg-purple-700/10 dark:bg-purple-500/15 text-purple-800 dark:text-purple-300",
    pii:
      "border border-zinc-400 dark:border-[#38384A] bg-zinc-200/70 dark:bg-[#1A1A24] text-zinc-900 dark:text-[#E2E2EC]",
    entity:
      "border border-zinc-400 dark:border-[#38384A] bg-zinc-200/70 dark:bg-[#1A1A24] text-zinc-900 dark:text-[#E2E2EC]",
    api_key:
      "border border-red-600 dark:border-[#FF2A2A] bg-red-600/10 dark:bg-[#FF2A2A]/15 text-red-700 dark:text-[#FF5555]",
    secret:
      "border border-red-600 dark:border-[#FF2A2A] bg-red-600/10 dark:bg-[#FF2A2A]/15 text-red-700 dark:text-[#FF5555]",
    confidential:
      "border border-red-600 dark:border-[#FF2A2A] bg-red-600/10 dark:bg-[#FF2A2A]/15 text-red-700 dark:text-[#FF5555]",
    blocked:
      "border border-red-600 bg-red-600 text-white",
  };

  return (
    <div
      className={cn(
        "inline-flex items-center px-2 py-0.5 font-mono text-[10px] font-bold uppercase tracking-[0.08em]",
        variants[variant] || variants.default,
        className
      )}
      {...props}
    />
  );
}

export { Badge };

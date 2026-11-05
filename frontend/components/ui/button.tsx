import * as React from "react";
import { cn } from "@/lib/utils";

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: "default" | "secondary" | "outline" | "ghost" | "destructive" | "purple";
  size?: "default" | "sm" | "lg" | "icon";
}

const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant = "default", size = "default", ...props }, ref) => {
    const baseStyles =
      "inline-flex items-center justify-center whitespace-nowrap font-mono text-[11px] font-bold uppercase tracking-[0.08em] transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-purple-500 disabled:pointer-events-none disabled:opacity-40";

    const variants = {
      default:
        "bg-foreground text-background hover:bg-purple-700 hover:text-white dark:hover:bg-purple-600 dark:hover:text-white border border-foreground",
      purple:
        "bg-purple-700 dark:bg-purple-600 text-white hover:bg-purple-800 dark:hover:bg-purple-500 border border-purple-800 dark:border-purple-400",
      secondary:
        "bg-secondary text-secondary-foreground hover:bg-purple-700/15 dark:hover:bg-purple-500/20 border border-border",
      outline:
        "border border-zinc-800 dark:border-[#38384A] bg-transparent hover:border-purple-600 hover:text-purple-700 dark:hover:border-purple-400 dark:hover:text-purple-300 text-foreground",
      ghost:
        "hover:bg-secondary text-muted-foreground hover:text-foreground border border-transparent hover:border-border",
      destructive:
        "bg-red-600/10 text-red-600 dark:text-[#FF4444] border border-red-600 dark:border-[#FF2A2A] hover:bg-red-600 hover:text-white",
    };

    const sizes = {
      default: "h-9 px-4 py-2",
      sm: "h-8 px-3",
      lg: "h-10 px-6 text-xs",
      icon: "h-8 w-8",
    };

    return (
      <button
        className={cn(baseStyles, variants[variant], sizes[size], className)}
        ref={ref}
        {...props}
      />
    );
  }
);
Button.displayName = "Button";

export { Button };

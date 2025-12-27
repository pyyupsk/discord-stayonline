import type { Component } from "vue";

import { Activity, CheckCircle2, Info, Loader2, XCircle } from "lucide-vue-next";

export type LogAction =
  | "backoff"
  | "config"
  | "connected"
  | "connecting"
  | "disconnected"
  | "error"
  | "system";

export type LogLevel = "error" | "info" | "warn";

export function formatActivityDate(date: Date): string {
  const today = new Date();
  const yesterday = new Date(today);
  yesterday.setDate(yesterday.getDate() - 1);

  if (date.toDateString() === today.toDateString()) {
    return "Today";
  } else if (date.toDateString() === yesterday.toDateString()) {
    return "Yesterday";
  }
  return date.toLocaleDateString("en-US", { day: "numeric", month: "short" });
}

export function formatActivityTime(date: Date): string {
  return date.toLocaleTimeString("en-US", {
    hour: "2-digit",
    hour12: false,
    minute: "2-digit",
    second: "2-digit",
  });
}

export function getActionBgColor(action?: string): string {
  switch (action) {
    case "backoff":
    case "connecting":
      return "bg-yellow-500";
    case "connected":
      return "text-green-500";
    case "disconnected":
    case "error":
      return "bg-destructive";
    default:
      return "bg-muted-foreground";
  }
}

export function getActionIcon(action?: string): Component {
  switch (action) {
    case "backoff":
    case "connecting":
      return Loader2;
    case "config":
    case "system":
      return Info;
    case "connected":
      return CheckCircle2;
    case "disconnected":
    case "error":
      return XCircle;
    default:
      return Activity;
  }
}

export function getActionTextColor(action?: string): string {
  switch (action) {
    case "backoff":
    case "connecting":
      return "text-yellow-500";
    case "connected":
      return "text-green-500";
    case "disconnected":
    case "error":
      return "text-destructive";
    case "system":
      return "text-primary";
    default:
      return "text-muted-foreground";
  }
}

export function getLevelBadgeVariant(level: string): "destructive" | "outline" | "secondary" {
  switch (level) {
    case "error":
      return "destructive";
    case "warn":
      return "secondary";
    default:
      return "outline";
  }
}

export function isSpinningAction(action?: string): boolean {
  return action === "connecting" || action === "backoff";
}

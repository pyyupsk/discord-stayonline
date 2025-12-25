<script setup lang="ts">
import {
  AlertCircle,
  CheckCircle2,
  Info,
  Loader2,
  RefreshCw,
  Settings,
  Trash2,
  Unplug,
  Wifi,
} from "lucide-vue-next";
import { computed } from "vue";

import type { LogEntry } from "@/types";

import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";

type LogFilter = "all" | LogEntry["level"];

const props = defineProps<{
  filter: LogFilter;
  logs: LogEntry[];
}>();

const emit = defineEmits<{
  clear: [];
  "update:filter": [value: LogFilter];
}>();

const filterOptions: { label: string; value: LogFilter }[] = [
  { label: "All", value: "all" },
  { label: "Info", value: "info" },
  { label: "Warn", value: "warn" },
  { label: "Error", value: "error" },
];

// Reverse logs to show newest first
const reversedLogs = computed(() => [...props.logs].reverse());

function formatTime(date: Date): string {
  return date.toLocaleTimeString("en-US", {
    hour: "2-digit",
    hour12: false,
    minute: "2-digit",
    second: "2-digit",
  });
}

function getActionClass(action?: LogEntry["action"], isLatest = false): string {
  switch (action) {
    case "backoff":
      return isLatest ? "text-warning animate-spin" : "text-warning";
    case "config":
      return "text-primary";
    case "connected":
      return "text-success";
    case "connecting":
      return isLatest ? "text-muted-foreground animate-spin" : "text-muted-foreground";
    case "disconnected":
      return "text-muted-foreground";
    case "error":
      return "text-destructive";
    default:
      return "text-muted-foreground";
  }
}

function getActionIcon(action?: LogEntry["action"]) {
  switch (action) {
    case "backoff":
      return RefreshCw;
    case "config":
      return Settings;
    case "connected":
      return CheckCircle2;
    case "connecting":
      return Loader2;
    case "disconnected":
      return Unplug;
    case "error":
      return AlertCircle;
    default:
      return Info;
  }
}

function getMessageClass(log: LogEntry): string {
  if (log.action === "connected") return "text-success";
  if (log.action === "error" || log.level === "error") return "text-destructive";
  if (log.action === "backoff" || log.level === "warn") return "text-warning";
  return "text-foreground/80";
}
</script>

<template>
  <div class="flex flex-col gap-3">
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-2">
        <h3 class="text-sm font-medium">Activity Log</h3>
        <span class="text-muted-foreground text-xs">({{ logs.length }} events)</span>
      </div>
      <div class="flex items-center gap-2">
        <div class="border-border/50 flex rounded-md border">
          <Button
            v-for="option in filterOptions"
            :key="option.value"
            :variant="filter === option.value ? 'secondary' : 'ghost'"
            size="sm"
            class="h-7 rounded-none px-3 text-xs first:rounded-l-md last:rounded-r-md"
            @click="emit('update:filter', option.value)"
          >
            {{ option.label }}
          </Button>
        </div>
        <Button
          variant="ghost"
          size="sm"
          class="press-effect text-muted-foreground hover:text-foreground h-7 px-2"
          title="Clear logs"
          @click="emit('clear')"
        >
          <Trash2 class="h-3.5 w-3.5" />
        </Button>
      </div>
    </div>

    <ScrollArea class="terminal-log border-border/50 h-64 rounded-lg border">
      <div
        v-if="logs.length === 0"
        class="text-muted-foreground flex h-full min-h-[200px] flex-col items-center justify-center gap-2"
      >
        <Wifi class="h-8 w-8 opacity-20" />
        <span class="text-sm opacity-50">No activity yet</span>
        <span class="text-xs opacity-30">Events will appear here as they happen</span>
      </div>
      <div v-else class="space-y-0.5 p-3">
        <div
          v-for="(log, index) in reversedLogs"
          :key="index"
          class="fade-in group flex items-start gap-3 rounded-md px-2 py-1.5 transition-colors hover:bg-white/5"
        >
          <!-- Time -->
          <span class="text-muted-foreground shrink-0 font-mono text-xs tabular-nums">
            {{ formatTime(log.time) }}
          </span>

          <!-- Action Icon -->
          <component
            :is="getActionIcon(log.action)"
            class="mt-0.5 h-3.5 w-3.5 shrink-0"
            :class="getActionClass(log.action, index === 0)"
          />

          <!-- Message -->
          <span class="flex-1 text-sm" :class="getMessageClass(log)">
            {{ log.message }}
          </span>

          <!-- Server Badge -->
          <span
            v-if="log.serverName"
            class="bg-muted text-muted-foreground shrink-0 rounded px-1.5 py-0.5 text-xs opacity-0 transition-opacity group-hover:opacity-100"
          >
            {{ log.serverName }}
          </span>
        </div>
      </div>
    </ScrollArea>
  </div>
</template>

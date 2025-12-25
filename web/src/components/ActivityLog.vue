<script setup lang="ts">
import { Trash2 } from "lucide-vue-next";

import type { LogEntry } from "@/types";

import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";

type LogFilter = "all" | LogEntry["level"];

defineProps<{
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

function formatTime(date: Date): string {
  return date.toLocaleTimeString();
}

function getLevelClass(level: LogEntry["level"]): string {
  switch (level) {
    case "debug":
      return "text-muted-foreground";
    case "error":
      return "text-destructive";
    case "warn":
      return "text-warning";
    default:
      return "text-success";
  }
}

function getLevelIndicator(level: LogEntry["level"]): string {
  switch (level) {
    case "debug":
      return "bg-muted-foreground";
    case "error":
      return "bg-destructive";
    case "warn":
      return "bg-warning";
    default:
      return "bg-success";
  }
}
</script>

<template>
  <div class="flex flex-col gap-3">
    <div class="flex items-center justify-between">
      <h3 class="text-sm font-medium">Activity Log</h3>
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
          size="icon-sm"
          class="press-effect text-muted-foreground hover:text-foreground text-xs"
          @click="emit('clear')"
        >
          <Trash2 />
        </Button>
      </div>
    </div>

    <ScrollArea class="terminal-log border-border/50 h-52 rounded-lg border p-4">
      <div
        v-if="logs.length === 0"
        class="text-muted-foreground flex h-full items-center justify-center text-sm"
      >
        <span class="opacity-50">No activity yet</span>
      </div>
      <div v-else class="space-y-2 font-mono text-xs">
        <div v-for="(log, index) in logs" :key="index" class="fade-in flex items-start gap-3">
          <span class="text-muted-foreground shrink-0 tabular-nums">
            {{ formatTime(log.time) }}
          </span>
          <span class="flex shrink-0 items-center gap-1.5">
            <span class="h-1.5 w-1.5 rounded-full" :class="getLevelIndicator(log.level)" />
            <span :class="getLevelClass(log.level)" class="font-medium uppercase">
              {{ log.level }}
            </span>
          </span>
          <span class="text-foreground/90">{{ log.message }}</span>
        </div>
      </div>
    </ScrollArea>
  </div>
</template>

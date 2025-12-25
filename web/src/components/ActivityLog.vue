<script setup lang="ts">
import { ScrollArea } from "@/components/ui/scroll-area";
import { Button } from "@/components/ui/button";
import { Trash2 } from "lucide-vue-next";
import type { LogEntry } from "@/types";

type LogFilter = LogEntry["level"] | "all";

const props = defineProps<{
  logs: LogEntry[];
  filter: LogFilter;
}>();

const emit = defineEmits<{
  clear: [];
  "update:filter": [value: LogFilter];
}>();

const filterOptions: { value: LogFilter; label: string }[] = [
  { value: "all", label: "All" },
  { value: "info", label: "Info" },
  { value: "warn", label: "Warn" },
  { value: "error", label: "Error" },
];

function formatTime(date: Date): string {
  return date.toLocaleTimeString();
}

function getLevelClass(level: LogEntry["level"]): string {
  switch (level) {
    case "error":
      return "text-destructive";
    case "warn":
      return "text-yellow-500";
    case "debug":
      return "text-muted-foreground";
    default:
      return "text-blue-500";
  }
}
</script>

<template>
  <div class="flex flex-col gap-2">
    <div class="flex items-center justify-between">
      <h3 class="text-sm font-medium">Activity Log</h3>
      <div class="flex items-center gap-1">
        <div class="flex rounded-md border">
          <Button
            v-for="option in filterOptions"
            :key="option.value"
            :variant="filter === option.value ? 'secondary' : 'ghost'"
            size="sm"
            class="h-7 rounded-none px-2 text-xs first:rounded-l-md last:rounded-r-md"
            @click="emit('update:filter', option.value)"
          >
            {{ option.label }}
          </Button>
        </div>
        <Button
          variant="ghost"
          size="sm"
          class="h-7 px-2 text-xs"
          @click="emit('clear')"
        >
          <Trash2 />
        </Button>
      </div>
    </div>

    <ScrollArea class="h-48 rounded-md border bg-muted/30 p-3">
      <div
        v-if="logs.length === 0"
        class="text-center text-sm text-muted-foreground"
      >
        No activity yet
      </div>
      <div v-else class="space-y-1 font-mono text-xs">
        <div v-for="(log, index) in logs" :key="index" class="flex gap-2">
          <span class="text-muted-foreground">{{ formatTime(log.time) }}</span>
          <span :class="getLevelClass(log.level)" class="uppercase">
            [{{ log.level }}]
          </span>
          <span class="text-foreground">{{ log.message }}</span>
        </div>
      </div>
    </ScrollArea>
  </div>
</template>

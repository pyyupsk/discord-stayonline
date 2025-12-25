<script setup lang="ts">
import { ScrollArea } from "@/components/ui/scroll-area";
import { Button } from "@/components/ui/button";
import { Trash2 } from "lucide-vue-next";
import type { LogEntry } from "@/types";

const props = defineProps<{
  logs: LogEntry[];
}>();

const emit = defineEmits<{
  clear: [];
}>();

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
      <Button
        variant="ghost"
        size="sm"
        class="h-7 px-2 text-xs"
        @click="emit('clear')"
      >
        <Trash2 />
        Clear
      </Button>
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

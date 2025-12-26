<script setup lang="ts">
import { Activity, CheckCircle2, Filter, Info, Loader2, Trash2, XCircle } from "lucide-vue-next";
import { computed, ref } from "vue";

import type { LogEntry, ServerEntry } from "@/types";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

const props = defineProps<{
  filter: string;
  logs: LogEntry[];
  servers: ServerEntry[];
}>();

const emit = defineEmits<{
  clear: [];
  "update:filter": [filter: string];
}>();

const serverFilter = ref<string>("all");

const filteredLogs = computed(() => {
  let result = [...props.logs];

  // Filter by level
  if (props.filter !== "all") {
    result = result.filter((log) => log.level === props.filter);
  }

  // Filter by server
  if (serverFilter.value !== "all") {
    result = result.filter((log) => log.serverId === serverFilter.value);
  }

  return result;
});

function formatDate(date: Date): string {
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

function formatTime(date: Date): string {
  return date.toLocaleTimeString("en-US", {
    hour: "2-digit",
    hour12: false,
    minute: "2-digit",
    second: "2-digit",
  });
}

function getActionColor(action?: string): string {
  switch (action) {
    case "backoff":
    case "connecting":
      return "text-warning";
    case "connected":
      return "text-success";
    case "disconnected":
    case "error":
      return "text-destructive";
    case "system":
      return "text-primary";
    default:
      return "text-muted-foreground";
  }
}

function getActionIcon(action?: string) {
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

function getLevelBadgeVariant(level: string) {
  switch (level) {
    case "error":
      return "destructive";
    case "warn":
      return "secondary";
    default:
      return "outline";
  }
}
</script>

<template>
  <div class="flex h-full flex-col space-y-4">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold">Activity Log</h1>
        <p class="text-muted-foreground">{{ filteredLogs.length }} entries</p>
      </div>

      <div class="flex items-center gap-3">
        <!-- Server Filter -->
        <Select v-model="serverFilter">
          <SelectTrigger class="w-[180px]">
            <Filter />
            <SelectValue placeholder="All Servers" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All Servers</SelectItem>
            <SelectItem v-for="server in servers" :key="server.id" :value="server.id">
              {{ server.guild_name || `Server ${server.guild_id.slice(-4)}` }}
            </SelectItem>
          </SelectContent>
        </Select>

        <!-- Level Filter -->
        <Select
          :model-value="filter"
          @update:model-value="(val) => emit('update:filter', String(val))"
        >
          <SelectTrigger class="w-[140px]">
            <SelectValue placeholder="All Levels" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All Levels</SelectItem>
            <SelectItem value="info">Info</SelectItem>
            <SelectItem value="warn">Warning</SelectItem>
            <SelectItem value="error">Error</SelectItem>
          </SelectContent>
        </Select>

        <!-- Clear Button -->
        <Button variant="outline" size="sm" @click="emit('clear')">
          <Trash2 />
          Clear
        </Button>
      </div>
    </div>

    <!-- Log List -->
    <ScrollArea class="border-border/50 bg-card flex-1 rounded-xl border">
      <div class="divide-border/50 divide-y">
        <div
          v-for="(log, index) in filteredLogs"
          :key="index"
          class="hover:bg-muted/30 flex items-start gap-4 p-4 transition-colors"
        >
          <!-- Time -->
          <div class="w-20 shrink-0 text-right">
            <p class="text-muted-foreground font-mono text-sm">{{ formatTime(log.time) }}</p>
            <p class="text-muted-foreground/60 text-xs">{{ formatDate(log.time) }}</p>
          </div>

          <!-- Icon -->
          <div class="pt-0.5">
            <component
              :is="getActionIcon(log.action)"
              class="h-5 w-5"
              :class="[
                getActionColor(log.action),
                { 'animate-spin': log.action === 'connecting' || log.action === 'backoff' },
              ]"
            />
          </div>

          <!-- Content -->
          <div class="min-w-0 flex-1">
            <p class="text-sm">{{ log.message }}</p>
            <div class="mt-1 flex items-center gap-2">
              <Badge v-if="log.serverName" variant="outline" class="text-xs">
                {{ log.serverName }}
              </Badge>
              <Badge :variant="getLevelBadgeVariant(log.level)" class="text-xs capitalize">
                {{ log.level }}
              </Badge>
            </div>
          </div>
        </div>

        <!-- Empty State -->
        <div
          v-if="filteredLogs.length === 0"
          class="flex flex-col items-center justify-center py-16"
        >
          <Activity class="text-muted-foreground/50 mb-4 h-12 w-12" />
          <p class="text-lg font-medium">No activity yet</p>
          <p class="text-muted-foreground text-sm">Logs will appear here as events occur</p>
        </div>
      </div>
    </ScrollArea>
  </div>
</template>

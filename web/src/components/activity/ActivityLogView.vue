<script setup lang="ts">
import { Activity, Trash2 } from "lucide-vue-next";
import { computed, ref } from "vue";

import type { LogEntry, ServerEntry } from "@/types";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  formatActivityDate,
  formatActivityTime,
  getActionIcon,
  getActionTextColor,
  getLevelBadgeVariant,
  isSpinningAction,
} from "@/lib/activity";

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

  if (props.filter !== "all") {
    result = result.filter((log) => log.level === props.filter);
  }

  if (serverFilter.value !== "all") {
    result = result.filter((log) => log.serverId === serverFilter.value);
  }

  return result;
});
</script>

<template>
  <div class="space-y-4">
    <!-- Header -->
    <div class="flex flex-wrap items-center justify-between gap-4">
      <div>
        <h1 class="text-2xl font-bold">Activity Log</h1>
        <p class="text-muted-foreground text-sm">{{ filteredLogs.length }} entries</p>
      </div>

      <div class="flex flex-wrap items-center gap-2">
        <Select v-model="serverFilter">
          <SelectTrigger class="w-[160px]">
            <SelectValue placeholder="All Servers" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All Servers</SelectItem>
            <SelectItem v-for="server in servers" :key="server.id" :value="server.id">
              {{ server.guild_name || `Server ${server.guild_id.slice(-4)}` }}
            </SelectItem>
          </SelectContent>
        </Select>

        <Select
          :model-value="filter"
          @update:model-value="(val) => emit('update:filter', String(val))"
        >
          <SelectTrigger class="w-[130px]">
            <SelectValue placeholder="All Levels" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All Levels</SelectItem>
            <SelectItem value="info">Info</SelectItem>
            <SelectItem value="warn">Warning</SelectItem>
            <SelectItem value="error">Error</SelectItem>
          </SelectContent>
        </Select>

        <Button variant="outline" size="sm" @click="emit('clear')">
          <Trash2 class="size-4" />
          Clear
        </Button>
      </div>
    </div>

    <!-- Log List -->
    <div class="bg-card border-border/50 rounded-lg border">
      <div class="divide-border/50 divide-y">
        <div
          v-for="(log, index) in filteredLogs"
          :key="index"
          class="hover:bg-muted/30 flex items-start gap-4 px-4 py-3 transition-colors"
        >
          <!-- Time -->
          <div class="w-16 shrink-0 pt-0.5 text-right">
            <p class="text-muted-foreground font-mono text-xs">
              {{ formatActivityTime(log.time) }}
            </p>
            <p class="text-muted-foreground/60 text-[10px]">{{ formatActivityDate(log.time) }}</p>
          </div>

          <!-- Icon -->
          <component
            :is="getActionIcon(log.action)"
            class="mt-0.5 size-4 shrink-0"
            :class="[
              getActionTextColor(log.action),
              { 'animate-spin': isSpinningAction(log.action) },
            ]"
          />

          <!-- Content -->
          <div class="min-w-0 flex-1">
            <p class="text-sm">{{ log.message }}</p>
            <div class="mt-1 flex flex-wrap items-center gap-1.5">
              <Badge v-if="log.serverName" variant="outline" class="text-[10px]">
                {{ log.serverName }}
              </Badge>
              <Badge :variant="getLevelBadgeVariant(log.level)" class="text-[10px] capitalize">
                {{ log.level }}
              </Badge>
            </div>
          </div>
        </div>

        <!-- Empty State -->
        <div v-if="filteredLogs.length === 0" class="flex flex-col items-center py-12">
          <Activity class="text-muted-foreground/50 mb-3 size-10" />
          <p class="font-medium">No activity yet</p>
          <p class="text-muted-foreground text-sm">Logs will appear here as events occur</p>
        </div>
      </div>
    </div>
  </div>
</template>

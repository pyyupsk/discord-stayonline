<script setup lang="ts">
import { Activity, Radio, Server, TrendingUp } from "lucide-vue-next";
import { computed } from "vue";

import type { ConnectionStatus, LogEntry, ServerEntry } from "@/types";

import { Badge } from "@/components/ui/badge";
import {
  formatActivityTime,
  getActionIcon,
  getActionTextColor,
  isSpinningAction,
} from "@/lib/activity";

import StatCard from "./StatCard.vue";

const props = defineProps<{
  logs: LogEntry[];
  servers: ServerEntry[];
  serverStatuses: Map<string, ConnectionStatus>;
}>();

const connectedCount = computed(() => {
  let count = 0;
  props.servers.forEach((server) => {
    if (props.serverStatuses.get(server.id) === "connected") {
      count++;
    }
  });
  return count;
});

const successRate = computed(() => {
  const connectionLogs = props.logs.filter((l) => l.action === "connected" || l.action === "error");
  if (connectionLogs.length === 0) return 100;

  const successCount = connectionLogs.filter((l) => l.action === "connected").length;
  return Math.round((successCount / connectionLogs.length) * 100);
});

const recentActivity = computed(() => props.logs.slice(0, 5));

const serverStats = computed(() =>
  props.servers.map((server) => ({
    server,
    status: props.serverStatuses.get(server.id) || ("disconnected" as ConnectionStatus),
  })),
);

function getGuildIconUrl(guildId: string, iconHash: string) {
  return `https://cdn.discordapp.com/icons/${guildId}/${iconHash}.png?size=64`;
}

function getStatusBadge(status: ConnectionStatus) {
  switch (status) {
    case "backoff":
      return {
        class: "bg-yellow-500/10 text-yellow-500 border-yellow-500/20",
        label: "Reconnecting",
      };
    case "connected":
      return { class: "bg-green-500/10 text-green-500 border-green-500/20", label: "Connected" };
    case "connecting":
      return {
        class: "bg-yellow-500/10 text-yellow-500 border-yellow-500/20",
        label: "Connecting",
      };
    case "error":
      return { class: "bg-red-500/10 text-red-500 border-red-500/20", label: "Error" };
    default:
      return { class: "bg-muted text-muted-foreground", label: "Disconnected" };
  }
}
</script>

<template>
  <div class="space-y-6">
    <!-- Stats -->
    <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
      <StatCard
        title="Connections"
        :value="`${connectedCount}/${servers.length}`"
        subtitle="Active voice connections"
        :icon="Radio"
        :variant="connectedCount === servers.length ? 'success' : 'primary'"
      />
      <StatCard
        title="Success Rate"
        :value="`${successRate}%`"
        subtitle="Connection reliability"
        :icon="TrendingUp"
        :variant="successRate >= 90 ? 'success' : successRate >= 70 ? 'warning' : 'destructive'"
      />
      <StatCard
        title="Servers"
        :value="servers.length.toString()"
        subtitle="Configured servers"
        :icon="Server"
        variant="default"
      />
      <StatCard
        title="Events"
        :value="logs.length.toString()"
        subtitle="Activity log entries"
        :icon="Activity"
        variant="default"
      />
    </div>

    <!-- Two Column Layout -->
    <div class="grid gap-6 lg:grid-cols-2">
      <!-- Servers -->
      <div class="bg-card border-border/50 rounded-lg border">
        <div class="border-border/50 flex items-center justify-between border-b px-4 py-3">
          <h2 class="font-semibold">Servers</h2>
          <Badge variant="outline" class="text-xs">{{ servers.length }}</Badge>
        </div>
        <div class="divide-border/50 divide-y">
          <div
            v-for="{ server, status } in serverStats"
            :key="server.id"
            class="flex items-center gap-3 px-4 py-3"
          >
            <img
              :src="
                server.guild_icon
                  ? getGuildIconUrl(server.guild_id, server.guild_icon)
                  : `https://ui-avatars.com/api/?name=${(server.guild_name || server.guild_id).slice(0, 2).toUpperCase()}&background=random`
              "
              :alt="server.guild_name || 'Server'"
              class="size-9 rounded-full object-cover"
            />
            <div class="min-w-0 flex-1">
              <p class="truncate text-sm font-medium">
                {{ server.guild_name || `Server ${server.guild_id.slice(-4)}` }}
              </p>
              <p class="text-muted-foreground truncate text-xs">
                {{ server.channel_name || server.channel_id }}
              </p>
            </div>
            <Badge variant="outline" class="shrink-0 text-xs" :class="getStatusBadge(status).class">
              {{ getStatusBadge(status).label }}
            </Badge>
          </div>
          <div v-if="servers.length === 0" class="text-muted-foreground px-4 py-8 text-center">
            <Server class="mx-auto mb-2 size-8 opacity-50" />
            <p class="text-sm">No servers configured</p>
          </div>
        </div>
      </div>

      <!-- Recent Activity -->
      <div class="bg-card border-border/50 rounded-lg border">
        <div class="border-border/50 flex items-center justify-between border-b px-4 py-3">
          <h2 class="font-semibold">Recent Activity</h2>
          <Badge variant="outline" class="text-xs">{{ logs.length }}</Badge>
        </div>
        <div class="divide-border/50 divide-y">
          <div
            v-for="(log, index) in recentActivity"
            :key="index"
            class="flex items-start gap-3 px-4 py-3"
          >
            <component
              :is="getActionIcon(log.action)"
              class="mt-0.5 size-4 shrink-0"
              :class="[
                getActionTextColor(log.action),
                { 'animate-spin': isSpinningAction(log.action) },
              ]"
            />
            <div class="min-w-0 flex-1">
              <p class="text-sm">{{ log.message }}</p>
              <p class="text-muted-foreground text-xs">
                {{ formatActivityTime(log.time) }}
                <span v-if="log.serverName" class="text-muted-foreground/60">
                  · {{ log.serverName }}
                </span>
              </p>
            </div>
          </div>
          <div
            v-if="recentActivity.length === 0"
            class="text-muted-foreground px-4 py-8 text-center"
          >
            <Activity class="mx-auto mb-2 size-8 opacity-50" />
            <p class="text-sm">No recent activity</p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

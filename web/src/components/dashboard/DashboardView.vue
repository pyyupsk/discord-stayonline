<script setup lang="ts">
import { Activity, Clock, Server, Zap } from "lucide-vue-next";
import { computed } from "vue";

import type { ConnectionStatus, LogEntry, ServerEntry } from "@/types";

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

const recentActivity = computed(() => {
  return props.logs.slice(0, 5);
});

const serverStats = computed(() => {
  return props.servers.map((server) => ({
    server,
    status: props.serverStatuses.get(server.id) || "disconnected",
  }));
});

function getGuildIconUrl(guildId: string, iconHash: string) {
  return `https://cdn.discordapp.com/icons/${guildId}/${iconHash}.png?size=64`;
}

function getStatusColor(status: ConnectionStatus) {
  switch (status) {
    case "backoff":
    case "connecting":
      return "bg-yellow-500";
    case "connected":
      return "bg-green-500";
    case "error":
      return "bg-destructive";
    default:
      return "bg-muted-foreground";
  }
}
</script>

<template>
  <div class="space-y-6">
    <!-- Stats Grid -->
    <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
      <StatCard
        title="Active Connections"
        :value="connectedCount.toString()"
        :subtitle="`of ${servers.length} servers`"
        :icon="Server"
        variant="primary"
      />
      <StatCard
        title="Success Rate"
        :value="`${successRate}%`"
        subtitle="connection success"
        :icon="Zap"
        :variant="successRate >= 90 ? 'success' : successRate >= 70 ? 'warning' : 'destructive'"
      />
      <StatCard
        title="Total Servers"
        :value="servers.length.toString()"
        subtitle="configured"
        :icon="Activity"
        variant="default"
      />
      <StatCard
        title="Session"
        value="Active"
        subtitle="WebSocket connected"
        :icon="Clock"
        variant="success"
      />
    </div>

    <!-- Server Overview -->
    <div class="border-border/50 bg-card rounded-xl border">
      <div class="border-border/50 border-b p-4">
        <h2 class="font-semibold">Server Overview</h2>
      </div>
      <div class="divide-border/50 divide-y">
        <div
          v-for="{ server, status } in serverStats"
          :key="server.id"
          class="flex items-center justify-between p-4"
        >
          <div class="flex items-center gap-3">
            <div class="relative">
              <img
                :src="
                  server.guild_icon
                    ? getGuildIconUrl(server.guild_id, server.guild_icon)
                    : `https://ui-avatars.com/api/?name=${(server.guild_name || server.guild_id).slice(0, 2).toUpperCase()}`
                "
                :alt="server.guild_name || 'Server'"
                class="h-10 w-10 rounded-full object-cover"
              />
              <span
                class="border-background absolute -right-0.5 -bottom-0.5 size-3 rounded-full border"
                :class="getStatusColor(serverStatuses.get(server.id) || 'disconnected')"
              />
            </div>
            <div>
              <p class="font-medium">
                {{ server.guild_name || `Server ${server.guild_id.slice(-4)}` }}
              </p>
              <p class="text-muted-foreground text-sm">
                {{ server.channel_name || server.channel_id }}
              </p>
            </div>
          </div>
          <div
            class="rounded-full px-3 py-1 text-xs font-medium capitalize"
            :class="getStatusColor(status)"
          >
            {{ status }}
          </div>
        </div>
        <div v-if="servers.length === 0" class="text-muted-foreground p-8 text-center">
          <Server class="mx-auto mb-2 h-8 w-8 opacity-50" />
          <p>No servers configured</p>
          <p class="text-sm">Click the + button in the sidebar to add a server</p>
        </div>
      </div>
    </div>

    <!-- Recent Activity -->
    <div class="border-border/50 bg-card rounded-xl border">
      <div class="border-border/50 border-b p-4">
        <h2 class="font-semibold">Recent Activity</h2>
      </div>
      <div class="divide-border/50 divide-y">
        <div
          v-for="(log, index) in recentActivity"
          :key="index"
          class="flex items-center gap-3 p-4"
        >
          <component
            :is="getActionIcon(log.action)"
            class="h-4 w-4 shrink-0"
            :class="[
              getActionTextColor(log.action),
              { 'animate-spin': isSpinningAction(log.action) },
            ]"
          />
          <div class="flex-1">
            <p class="text-sm">{{ log.message }}</p>
            <p class="text-muted-foreground text-xs">
              {{ formatActivityTime(log.time) }}
              <span v-if="log.serverName"> &middot; {{ log.serverName }}</span>
            </p>
          </div>
        </div>
        <div v-if="recentActivity.length === 0" class="text-muted-foreground p-8 text-center">
          <Activity class="mx-auto mb-2 h-8 w-8 opacity-50" />
          <p>No recent activity</p>
        </div>
      </div>
    </div>
  </div>
</template>

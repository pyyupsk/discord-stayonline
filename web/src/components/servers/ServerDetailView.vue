<script setup lang="ts">
import { Activity, Loader2, Pencil, Play, RotateCcw, Square, Trash2 } from "lucide-vue-next";
import { computed } from "vue";

import type { ConnectionStatus, LogEntry, ServerEntry } from "@/types";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  formatActivityTime,
  getActionIcon,
  getActionTextColor,
  isSpinningAction,
} from "@/lib/activity";

const props = defineProps<{
  isLoading: boolean;
  logs: LogEntry[];
  server: ServerEntry;
  status: ConnectionStatus;
}>();

const emit = defineEmits<{
  delete: [];
  edit: [];
  exit: [];
  join: [];
  rejoin: [];
}>();

const displayName = computed(() => {
  return props.server.guild_name || `Server ${props.server.guild_id.slice(-4)}`;
});

const channelName = computed(() => {
  return props.server.channel_name || props.server.channel_id;
});

const isConnected = computed(() => props.status === "connected");
const isConnecting = computed(() => props.status === "connecting" || props.status === "backoff");

const recentLogs = computed(() => props.logs.slice(0, 10));

function getGuildIconUrl(guildId: string, iconHash: string) {
  return `https://cdn.discordapp.com/icons/${guildId}/${iconHash}.png?size=128`;
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
    <!-- Header -->
    <div class="flex flex-wrap items-start justify-between gap-4">
      <div class="flex items-center gap-4">
        <img
          :src="
            server.guild_icon
              ? getGuildIconUrl(server.guild_id, server.guild_icon)
              : `https://ui-avatars.com/api/?name=${(server.guild_name || server.guild_id).slice(0, 2).toUpperCase()}&background=random&size=128`
          "
          :alt="displayName"
          class="size-14 rounded-xl object-cover"
        />
        <div>
          <div class="flex items-center gap-2">
            <h1 class="text-2xl font-bold">{{ displayName }}</h1>
            <Badge variant="outline" :class="getStatusBadge(status).class">
              <Loader2 v-if="isLoading || isConnecting" class="size-3 animate-spin" />
              {{ getStatusBadge(status).label }}
            </Badge>
          </div>
          <p class="text-muted-foreground">{{ channelName }}</p>
        </div>
      </div>

      <div class="flex gap-2">
        <Button variant="outline" size="sm" @click="emit('edit')">
          <Pencil class="size-4" />
          Edit
        </Button>
        <Button
          variant="outline"
          size="sm"
          class="text-destructive hover:bg-destructive/10"
          @click="emit('delete')"
        >
          <Trash2 class="size-4" />
          Delete
        </Button>
      </div>
    </div>

    <!-- Actions -->
    <div class="flex flex-wrap gap-2">
      <Button v-if="!isConnected" :disabled="isLoading || isConnecting" @click="emit('join')">
        <Loader2 v-if="isLoading" class="animate-spin" />
        <Play v-else class="size-4" />
        Connect
      </Button>
      <Button v-if="isConnected" variant="secondary" :disabled="isLoading" @click="emit('rejoin')">
        <Loader2 v-if="isLoading" class="animate-spin" />
        <RotateCcw v-else class="size-4" />
        Reconnect
      </Button>
      <Button
        v-if="isConnected || isConnecting"
        variant="outline"
        class="text-destructive hover:bg-destructive/10"
        :disabled="isLoading"
        @click="emit('exit')"
      >
        <Square class="size-4" />
        Disconnect
      </Button>
    </div>

    <!-- Two Column Layout -->
    <div class="grid gap-6 lg:grid-cols-2">
      <!-- Server Details -->
      <div class="bg-card border-border/50 rounded-lg border">
        <div class="border-border/50 border-b px-4 py-3">
          <h2 class="font-semibold">Details</h2>
        </div>
        <div class="divide-border/50 divide-y text-sm">
          <div class="flex items-center justify-between px-4 py-3">
            <span class="text-muted-foreground">Guild ID</span>
            <code class="bg-muted rounded px-2 py-0.5 font-mono text-xs">{{
              server.guild_id
            }}</code>
          </div>
          <div class="flex items-center justify-between px-4 py-3">
            <span class="text-muted-foreground">Channel ID</span>
            <code class="bg-muted rounded px-2 py-0.5 font-mono text-xs">{{
              server.channel_id
            }}</code>
          </div>
          <div class="flex items-center justify-between px-4 py-3">
            <span class="text-muted-foreground">Auto-connect</span>
            <Badge variant="outline" class="text-xs">
              {{ server.connect_on_start ? "Enabled" : "Disabled" }}
            </Badge>
          </div>
          <div class="flex items-center justify-between px-4 py-3">
            <span class="text-muted-foreground">Priority</span>
            <Badge variant="outline" class="text-xs">{{ server.priority }}</Badge>
          </div>
        </div>
      </div>

      <!-- Recent Activity -->
      <div class="bg-card border-border/50 rounded-lg border">
        <div class="border-border/50 flex items-center justify-between border-b px-4 py-3">
          <h2 class="font-semibold">Activity</h2>
          <Badge variant="outline" class="text-xs">{{ logs.length }}</Badge>
        </div>
        <div class="divide-border/50 divide-y">
          <div
            v-for="(log, index) in recentLogs"
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
              <p class="text-muted-foreground text-xs">{{ formatActivityTime(log.time) }}</p>
            </div>
          </div>
          <div v-if="recentLogs.length === 0" class="text-muted-foreground px-4 py-8 text-center">
            <Activity class="mx-auto mb-2 size-8 opacity-50" />
            <p class="text-sm">No activity for this server</p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

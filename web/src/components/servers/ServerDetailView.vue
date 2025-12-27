<script setup lang="ts">
import { Loader2, Pencil, Play, RotateCcw, Square, Trash2 } from "lucide-vue-next";
import { computed } from "vue";

import type { ConnectionStatus, LogEntry, ServerEntry } from "@/types";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";

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

const statusConfig = computed(() => {
  switch (props.status) {
    case "backoff":
      return {
        class: "bg-yellow-500/20 text-yellow-500 border-yellow-500/50",
        label: "Reconnecting",
        variant: "secondary" as const,
      };
    case "connected":
      return {
        class: "bg-green-500/20 text-green-500 border-green-500/50",
        label: "Connected",
        variant: "default" as const,
      };
    case "connecting":
      return {
        class: "bg-yellow-500/20 text-yellow-500 border-yellow-500/50",
        label: "Connecting",
        variant: "secondary" as const,
      };
    case "error":
      return {
        class: "bg-destructive/20 text-destructive border-destructive/50",
        label: "Error",
        variant: "destructive" as const,
      };
    default:
      return { class: "", label: "Disconnected", variant: "secondary" as const };
  }
});

const recentLogs = computed(() => {
  return props.logs.slice(0, 10);
});

function getGuildIconUrl(guildId: string, iconHash: string) {
  return `https://cdn.discordapp.com/icons/${guildId}/${iconHash}.png?size=128`;
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
    <!-- Server Header -->
    <div class="flex items-start justify-between">
      <div class="flex items-center gap-4">
        <div class="relative">
          <img
            :src="
              server.guild_icon
                ? getGuildIconUrl(server.guild_id, server.guild_icon)
                : `https://ui-avatars.com/api/?name=${(server.guild_name || server.guild_id).slice(0, 2).toUpperCase()}`
            "
            :alt="displayName"
            class="h-16 w-16 rounded-2xl object-cover"
          />
          <span
            class="border-background absolute -right-1 -bottom-1 size-4 rounded-full border-2"
            :class="getStatusColor(props.status || 'disconnected')"
          />
        </div>
        <div>
          <h1 class="inline-flex items-center gap-2 text-2xl font-bold">
            <span>{{ displayName }}</span>
            <Badge :class="statusConfig.class">
              <Loader2 v-if="isLoading || isConnecting" class="animate-spin" />
              {{ statusConfig.label }}
            </Badge>
          </h1>
          <p class="text-muted-foreground">{{ channelName }}</p>
        </div>
      </div>

      <div class="flex gap-2">
        <Button variant="outline" size="sm" @click="emit('edit')">
          <Pencil />
          Edit
        </Button>
        <Button variant="destructive" size="sm" @click="emit('delete')">
          <Trash2 />
          Delete
        </Button>
      </div>
    </div>

    <!-- Actions -->
    <div class="flex gap-3">
      <Button
        v-if="!isConnected"
        :disabled="isLoading || isConnecting"
        class="flex-1"
        @click="emit('join')"
      >
        <Loader2 v-if="isLoading" class="animate-spin" />
        <Play v-else />
        Connect
      </Button>
      <Button
        v-if="isConnected"
        variant="secondary"
        :disabled="isLoading"
        class="flex-1"
        @click="emit('rejoin')"
      >
        <Loader2 v-if="isLoading" class="animate-spin" />
        <RotateCcw v-else />
        Reconnect
      </Button>
      <Button
        v-if="isConnected || isConnecting"
        variant="destructive"
        :disabled="isLoading"
        class="flex-1"
        @click="emit('exit')"
      >
        <Square />
        Disconnect
      </Button>
    </div>

    <Separator />

    <!-- Server Details -->
    <div class="border-border/50 bg-card rounded-xl border">
      <div class="border-border/50 border-b p-4">
        <h2 class="font-semibold">Server Details</h2>
      </div>
      <div class="divide-border/50 divide-y">
        <div class="flex justify-between p-4">
          <span class="text-muted-foreground">Guild ID</span>
          <code class="font-mono text-sm">{{ server.guild_id }}</code>
        </div>
        <div class="flex justify-between p-4">
          <span class="text-muted-foreground">Channel ID</span>
          <code class="font-mono text-sm">{{ server.channel_id }}</code>
        </div>
        <div class="flex justify-between p-4">
          <span class="text-muted-foreground">Auto-connect</span>
          <span>{{ server.connect_on_start ? "Yes" : "No" }}</span>
        </div>
      </div>
    </div>

    <!-- Server Activity -->
    <div class="border-border/50 bg-card rounded-xl border">
      <div class="border-border/50 border-b p-4">
        <h2 class="font-semibold">Recent Activity</h2>
      </div>
      <div class="divide-border/50 divide-y">
        <div v-for="(log, index) in recentLogs" :key="index" class="flex items-center gap-3 p-4">
          <div
            class="h-2 w-2 rounded-full"
            :class="{
              'bg-green-500': log.action === 'connected',
              'bg-destructive': log.action === 'error' || log.action === 'disconnected',
              'bg-yellow-500': log.action === 'connecting' || log.action === 'backoff',
              'bg-primary': log.action === 'system',
              'bg-muted-foreground': !log.action,
            }"
          />
          <div class="flex-1">
            <p class="text-sm">{{ log.message }}</p>
            <p class="text-muted-foreground text-xs">{{ log.time.toLocaleTimeString() }}</p>
          </div>
        </div>
        <div v-if="recentLogs.length === 0" class="text-muted-foreground p-8 text-center">
          <p>No activity for this server</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Pencil, Play, RotateCcw, Square, Trash2 } from "lucide-vue-next";
import { computed } from "vue";

import type { ConnectionStatus, ServerEntry } from "@/types";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";

const props = defineProps<{
  loading?: boolean;
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

const statusVariant = computed(() => {
  switch (props.status) {
    case "backoff":
    case "connecting":
      return "secondary";
    case "connected":
      return "default";
    case "error":
      return "destructive";
    default:
      return "outline";
  }
});

const isConnecting = computed(() => {
  return props.status === "connecting" || props.status === "backoff";
});

const statusLabel = computed(() => {
  switch (props.status) {
    case "backoff":
      return "Reconnecting...";
    case "connected":
      return "Connected";
    case "connecting":
      return "Connecting...";
    case "error":
      return "Error";
    default:
      return "Disconnected";
  }
});

const displayName = computed(() => {
  return props.server.guild_name || `Server ${props.server.guild_id.slice(-4)}`;
});

const channelDisplay = computed(() => {
  return props.server.channel_name || props.server.channel_id;
});
</script>

<template>
  <Card class="hover-lift border-border/50 hover:border-border transition-colors">
    <CardContent class="flex items-center justify-between gap-4">
      <div class="min-w-0 flex-1">
        <div class="flex items-center gap-2">
          <div
            class="h-2 w-2 rounded-full"
            :class="{
              'bg-success status-glow': status === 'connected',
              'bg-muted-foreground pulse-connecting': isConnecting,
              'bg-destructive': status === 'error',
              'bg-muted-foreground/50': status === 'disconnected',
            }"
          />
          <span class="font-medium">{{ displayName }}</span>
          <Badge
            :variant="statusVariant"
            class="text-xs"
            :class="{ 'pulse-connecting': isConnecting }"
          >
            {{ statusLabel }}
          </Badge>
        </div>
        <p class="text-muted-foreground mt-1 truncate pl-4 text-sm">
          {{ channelDisplay }}
        </p>
      </div>

      <div class="flex items-center gap-1">
        <Button
          variant="ghost"
          size="icon"
          class="press-effect"
          :disabled="loading || status === 'connected' || status === 'connecting'"
          title="Join"
          @click="emit('join')"
        >
          <Play />
        </Button>
        <Button
          variant="ghost"
          size="icon"
          class="press-effect"
          :disabled="loading"
          title="Rejoin"
          @click="emit('rejoin')"
        >
          <RotateCcw />
        </Button>
        <Button
          variant="ghost"
          size="icon"
          class="press-effect"
          :disabled="loading || status === 'disconnected'"
          title="Exit"
          @click="emit('exit')"
        >
          <Square />
        </Button>
        <div class="bg-border mx-1 h-4 w-px" />
        <Button variant="ghost" size="icon" class="press-effect" title="Edit" @click="emit('edit')">
          <Pencil />
        </Button>
        <Button
          variant="ghost"
          size="icon"
          class="press-effect text-destructive hover:bg-destructive/10 hover:text-destructive"
          title="Delete"
          @click="emit('delete')"
        >
          <Trash2 />
        </Button>
      </div>
    </CardContent>
  </Card>
</template>

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
  <Card>
    <CardContent class="flex items-center justify-between gap-4">
      <div class="min-w-0 flex-1">
        <div class="flex items-center gap-2">
          <span class="font-medium">{{ displayName }}</span>
          <Badge :variant="statusVariant" class="text-xs">
            {{ statusLabel }}
          </Badge>
        </div>
        <p class="text-muted-foreground mt-1 truncate text-sm">
          {{ channelDisplay }}
        </p>
      </div>

      <div class="flex items-center gap-2">
        <Button
          variant="outline"
          size="icon"
          :disabled="loading || status === 'connected' || status === 'connecting'"
          title="Join"
          @click="emit('join')"
        >
          <Play />
        </Button>
        <Button
          variant="outline"
          size="icon"
          :disabled="loading"
          title="Rejoin"
          @click="emit('rejoin')"
        >
          <RotateCcw />
        </Button>
        <Button
          variant="outline"
          size="icon"
          :disabled="loading || status === 'disconnected'"
          title="Exit"
          @click="emit('exit')"
        >
          <Square />
        </Button>
        <Button variant="ghost" size="icon" title="Edit" @click="emit('edit')">
          <Pencil />
        </Button>
        <Button
          variant="ghost"
          size="icon"
          class="text-destructive hover:text-destructive"
          title="Delete"
          @click="emit('delete')"
        >
          <Trash2 />
        </Button>
      </div>
    </CardContent>
  </Card>
</template>

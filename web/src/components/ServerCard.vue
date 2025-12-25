<script setup lang="ts">
import { computed } from "vue";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Play, RotateCcw, Square, Pencil, Trash2 } from "lucide-vue-next";
import type { ServerEntry, ConnectionStatus } from "@/types";

const props = defineProps<{
  server: ServerEntry;
  status: ConnectionStatus;
  loading?: boolean;
}>();

const emit = defineEmits<{
  join: [];
  rejoin: [];
  exit: [];
  edit: [];
  delete: [];
}>();

const statusVariant = computed(() => {
  switch (props.status) {
    case "connected":
      return "default";
    case "connecting":
    case "backoff":
      return "secondary";
    case "error":
      return "destructive";
    default:
      return "outline";
  }
});

const statusLabel = computed(() => {
  switch (props.status) {
    case "connected":
      return "Connected";
    case "connecting":
      return "Connecting...";
    case "backoff":
      return "Reconnecting...";
    case "error":
      return "Error";
    default:
      return "Disconnected";
  }
});
</script>

<template>
  <Card>
    <CardContent class="flex items-center justify-between gap-4 p-4">
      <div class="min-w-0 flex-1">
        <div class="flex items-center gap-2">
          <span class="font-medium">Server Entry</span>
          <Badge :variant="statusVariant" class="text-xs">
            {{ statusLabel }}
          </Badge>
        </div>
        <p class="mt-1 truncate text-sm text-muted-foreground">
          Guild: {{ server.guild_id }} • Channel: {{ server.channel_id }}
        </p>
      </div>

      <div class="flex items-center gap-2">
        <Button
          variant="outline"
          size="icon"
          :disabled="loading || status === 'connected' || status === 'connecting'"
          @click="emit('join')"
          title="Join"
        >
          <Play class="h-4 w-4" />
        </Button>
        <Button
          variant="outline"
          size="icon"
          :disabled="loading"
          @click="emit('rejoin')"
          title="Rejoin"
        >
          <RotateCcw class="h-4 w-4" />
        </Button>
        <Button
          variant="outline"
          size="icon"
          :disabled="loading || status === 'disconnected'"
          @click="emit('exit')"
          title="Exit"
        >
          <Square class="h-4 w-4" />
        </Button>
        <Button
          variant="ghost"
          size="icon"
          @click="emit('edit')"
          title="Edit"
        >
          <Pencil class="h-4 w-4" />
        </Button>
        <Button
          variant="ghost"
          size="icon"
          class="text-destructive hover:text-destructive"
          @click="emit('delete')"
          title="Delete"
        >
          <Trash2 class="h-4 w-4" />
        </Button>
      </div>
    </CardContent>
  </Card>
</template>

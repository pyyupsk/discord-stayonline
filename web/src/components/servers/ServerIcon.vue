<script setup lang="ts">
import { Loader2 } from "lucide-vue-next";
import { computed } from "vue";

import type { ConnectionStatus, ServerEntry } from "@/types";

import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";

const props = defineProps<{
  isLoading: boolean;
  isSelected: boolean;
  server: ServerEntry;
  status: ConnectionStatus;
}>();

defineEmits<{
  click: [];
}>();

const initials = computed(() => {
  const name = props.server.guild_name || props.server.guild_id;
  if (!name) return "?";

  // If it's a guild ID (numeric), show last 2 chars
  if (/^\d+$/.test(name)) {
    return name.slice(-2);
  }

  // Get first letter of first two words
  const words = name.split(/\s+/);
  if (words.length >= 2 && words[0] && words[1]) {
    return ((words[0][0] || "") + (words[1][0] || "")).toUpperCase();
  }
  return name.slice(0, 2).toUpperCase();
});

const displayName = computed(() => {
  return props.server.guild_name || `Server ${props.server.guild_id.slice(-4)}`;
});

const statusColor = computed(() => {
  switch (props.status) {
    case "backoff":
    case "connecting":
      return "bg-warning";
    case "connected":
      return "bg-success";
    case "error":
      return "bg-destructive";
    default:
      return "bg-muted-foreground";
  }
});

const isConnecting = computed(() => {
  return props.status === "connecting" || props.status === "backoff" || props.isLoading;
});
</script>

<template>
  <Tooltip>
    <TooltipTrigger as-child>
      <button
        class="discord-server-icon group relative"
        :class="{
          selected: isSelected,
          connected: status === 'connected',
        }"
        @click="$emit('click')"
      >
        <!-- Selection Indicator -->
        <div
          class="bg-foreground absolute -left-3 h-2 w-1 rounded-r-full transition-all duration-200"
          :class="{
            'h-5': isSelected,
            'h-2 group-hover:h-4': !isSelected,
            'opacity-0': !isSelected && status !== 'connected',
          }"
        />

        <!-- Icon Content -->
        <span class="relative text-sm font-semibold">
          {{ initials }}
        </span>

        <!-- Status Indicator -->
        <div
          class="absolute -right-0.5 -bottom-0.5 h-4 w-4 rounded-full border-[3px] border-[#1e1f22]"
          :class="statusColor"
        >
          <Loader2 v-if="isConnecting" class="h-full w-full animate-spin p-0.5 text-[#1e1f22]" />
        </div>
      </button>
    </TooltipTrigger>
    <TooltipContent side="right" :side-offset="10" class="flex flex-col">
      <p class="font-medium">{{ displayName }}</p>
      <p class="text-muted-foreground text-xs">{{ server.channel_name || server.channel_id }}</p>
    </TooltipContent>
  </Tooltip>
</template>

<style scoped>
.discord-server-icon {
  display: flex;
  height: 48px;
  width: 48px;
  cursor: pointer;
  align-items: center;
  justify-content: center;
  border-radius: 24px;
  background: var(--muted);
  color: var(--muted-foreground);
  transition: all 0.15s ease;
}

.discord-server-icon:hover,
.discord-server-icon.selected {
  border-radius: 16px;
}

.discord-server-icon.connected {
  background: color-mix(in srgb, var(--success) 20%, transparent);
  color: var(--success);
}

.discord-server-icon.connected:hover,
.discord-server-icon.connected.selected {
  background: var(--success);
  color: var(--success-foreground);
}

.discord-server-icon:not(.connected):hover {
  background: var(--primary);
  color: var(--primary-foreground);
}

.discord-server-icon.selected:not(.connected) {
  background: var(--primary);
  color: var(--primary-foreground);
}
</style>

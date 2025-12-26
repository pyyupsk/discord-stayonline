<script setup lang="ts">
import { LogOut, Wifi, WifiOff } from "lucide-vue-next";

import type { Status } from "@/types";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

defineProps<{
  connectedCount: number;
  status: Status;
  totalCount: number;
  wsStatus: string;
}>();

const emit = defineEmits<{
  logout: [];
  updateStatus: [status: string];
}>();

function getStatusLabel(status: Status): string {
  switch (status) {
    case "dnd":
      return "Do Not Disturb";
    case "idle":
      return "Idle";
    case "online":
      return "Online";
    default:
      return status;
  }
}
</script>

<template>
  <header class="border-border/50 bg-card flex h-14 items-center justify-between border-b px-6">
    <div class="flex items-center gap-4">
      <!-- App Title -->
      <h1 class="text-lg font-semibold tracking-tight">Discord Stay Online</h1>

      <!-- Connection Badge -->
      <Badge :variant="wsStatus === 'connected' ? 'default' : 'secondary'" class="gap-1.5">
        <Wifi v-if="wsStatus === 'connected'" class="h-3 w-3" />
        <WifiOff v-else class="h-3 w-3" />
        {{ connectedCount }}/{{ totalCount }} Connected
      </Badge>
    </div>

    <div class="flex items-center gap-3">
      <!-- Status Selector -->
      <Select
        :model-value="status"
        @update:model-value="(val) => emit('updateStatus', String(val))"
      >
        <SelectTrigger class="w-[160px]">
          <SelectValue>
            <div class="flex items-center gap-2">
              <span
                class="h-2.5 w-2.5 rounded-full"
                :class="{
                  'bg-success': status === 'online',
                  'bg-warning': status === 'idle',
                  'bg-destructive': status === 'dnd',
                }"
              />
              {{ getStatusLabel(status) }}
            </div>
          </SelectValue>
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="online">
            <div class="flex items-center gap-2">
              <span class="bg-success h-2.5 w-2.5 rounded-full" />
              Online
            </div>
          </SelectItem>
          <SelectItem value="idle">
            <div class="flex items-center gap-2">
              <span class="bg-warning h-2.5 w-2.5 rounded-full" />
              Idle
            </div>
          </SelectItem>
          <SelectItem value="dnd">
            <div class="flex items-center gap-2">
              <span class="bg-destructive h-2.5 w-2.5 rounded-full" />
              Do Not Disturb
            </div>
          </SelectItem>
        </SelectContent>
      </Select>

      <!-- Logout Button -->
      <Button variant="ghost" size="icon" @click="emit('logout')">
        <LogOut />
      </Button>
    </div>
  </header>
</template>

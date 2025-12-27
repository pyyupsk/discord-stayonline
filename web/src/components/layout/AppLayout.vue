<script setup lang="ts">
import { LogOut, Wifi, WifiOff } from "lucide-vue-next";

import type { Status } from "@/types";

import ModeToggle from "@/components/ModeToggle.vue";
import ServerForm from "@/components/ServerForm.vue";
import TosModal from "@/components/TosModal.vue";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";
import { SidebarInset, SidebarProvider, SidebarTrigger } from "@/components/ui/sidebar";
import { Skeleton } from "@/components/ui/skeleton";
import { useDashboard } from "@/composables/useDashboard";

import AppSidebar from "./AppSidebar.vue";

const {
  actionLoading,
  config,
  connectedCount,
  editingServer,
  handleAcknowledgeTos,
  handleAddServer,
  handleExit,
  handleJoin,
  handleLogout,
  handleSaveServer,
  handleStatusChange,
  initialLoading,
  serverStatusMap,
  showServerForm,
  wsStatus,
} = useDashboard();

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
  <!-- Loading State -->
  <div v-if="initialLoading" class="flex min-h-screen items-center justify-center">
    <div class="flex flex-col items-center gap-4">
      <Skeleton class="h-12 w-12 rounded-full" />
      <div class="space-y-2">
        <Skeleton class="h-4 w-32" />
        <Skeleton class="h-3 w-24" />
      </div>
    </div>
  </div>

  <!-- Main App -->
  <template v-else>
    <!-- TOS Modal -->
    <TosModal :open="!config.tos_acknowledged" @acknowledge="handleAcknowledgeTos" />

    <!-- Main Layout -->
    <SidebarProvider v-if="config.tos_acknowledged">
      <AppSidebar
        :servers="config.servers"
        :server-statuses="serverStatusMap"
        :action-loading="actionLoading"
        @add-server="handleAddServer"
        @join-server="handleJoin"
        @exit-server="handleExit"
      />

      <SidebarInset>
        <!-- Header -->
        <header
          class="bg-background/95 supports-backdrop-filter:bg-background/60 sticky top-0 z-10 flex h-14 shrink-0 items-center gap-2 border-b px-4 backdrop-blur"
        >
          <SidebarTrigger class="-ml-1" />
          <Separator orientation="vertical" class="mr-2 h-4" />

          <!-- Connection Badge -->
          <Badge :variant="wsStatus === 'connected' ? 'default' : 'secondary'" class="gap-1.5">
            <Wifi v-if="wsStatus === 'connected'" class="h-3 w-3" />
            <WifiOff v-else class="h-3 w-3" />
            {{ connectedCount }}/{{ config.servers.length }} Connected
          </Badge>

          <div class="ml-auto flex items-center gap-3">
            <!-- Status Selector -->
            <Select
              :model-value="config.status"
              @update:model-value="(val) => handleStatusChange(String(val))"
            >
              <SelectTrigger class="w-[160px]">
                <SelectValue>
                  <div class="flex items-center gap-2">
                    <span
                      class="h-2.5 w-2.5 rounded-full"
                      :class="{
                        'bg-success': config.status === 'online',
                        'bg-warning': config.status === 'idle',
                        'bg-destructive': config.status === 'dnd',
                      }"
                    />
                    {{ getStatusLabel(config.status) }}
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

            <!-- Mode Toggle -->
            <ModeToggle />

            <!-- Logout Button -->
            <Button variant="ghost" size="icon" @click="handleLogout">
              <LogOut />
            </Button>
          </div>
        </header>

        <!-- Content Area -->
        <main class="p-6">
          <slot />
        </main>
      </SidebarInset>
    </SidebarProvider>

    <!-- Server Form Dialog -->
    <ServerForm v-model:open="showServerForm" :server="editingServer" @save="handleSaveServer" />
  </template>
</template>

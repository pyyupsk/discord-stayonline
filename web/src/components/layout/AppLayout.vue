<script setup lang="ts">
import { Wifi, WifiOff } from "lucide-vue-next";

import TosModal from "@/components/modals/TosModal.vue";
import ServerForm from "@/components/servers/ServerForm.vue";
import { Badge } from "@/components/ui/badge";
import { Separator } from "@/components/ui/separator";
import { SidebarInset, SidebarProvider, SidebarTrigger } from "@/components/ui/sidebar";
import { Skeleton } from "@/components/ui/skeleton";
import { useDashboard } from "@/composables/useDashboard";

import AppSidebar from "./AppSidebar.vue";
import UserDropdown from "./UserDropdown.vue";

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

          <!-- User Dropdown -->
          <UserDropdown
            :status="config.status"
            class="ml-auto"
            @status-change="handleStatusChange"
            @logout="handleLogout"
          />
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

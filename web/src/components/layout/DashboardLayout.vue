<script setup lang="ts">
import ServerForm from "@/components/ServerForm.vue";
import TosModal from "@/components/TosModal.vue";
import { useDashboard } from "@/composables/useDashboard";

import ContentHeader from "./ContentHeader.vue";
import Sidebar from "./Sidebar.vue";

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
  <div v-if="initialLoading" class="bg-background flex h-screen items-center justify-center">
    <div class="flex flex-col items-center gap-3">
      <div class="border-muted border-t-foreground h-8 w-8 animate-spin rounded-full border-2" />
      <p class="text-muted-foreground text-sm">Loading...</p>
    </div>
  </div>

  <!-- Main App -->
  <template v-else>
    <!-- TOS Modal -->
    <TosModal :open="!config.tos_acknowledged" @acknowledge="handleAcknowledgeTos" />

    <!-- Main Layout -->
    <div v-if="config.tos_acknowledged" class="bg-background flex h-screen w-full overflow-hidden">
      <!-- Sidebar -->
      <Sidebar
        :servers="config.servers"
        :server-statuses="serverStatusMap"
        :action-loading="actionLoading"
        @add-server="handleAddServer"
        @join-server="handleJoin"
        @exit-server="handleExit"
      />

      <!-- Main Content -->
      <main class="flex flex-1 flex-col overflow-hidden">
        <!-- Header -->
        <ContentHeader
          :status="config.status"
          :ws-status="wsStatus"
          :connected-count="connectedCount"
          :total-count="config.servers.length"
          @update-status="handleStatusChange"
          @logout="handleLogout"
        />

        <!-- Content Area -->
        <div class="flex-1 overflow-y-auto p-6">
          <slot />
        </div>
      </main>
    </div>

    <!-- Server Form Dialog -->
    <ServerForm v-model:open="showServerForm" :server="editingServer" @save="handleSaveServer" />
  </template>
</template>

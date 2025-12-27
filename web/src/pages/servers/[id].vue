<script setup lang="ts">
import { computed } from "vue";
import { useRoute, useRouter } from "vue-router";

import AppLayout from "@/components/layout/AppLayout.vue";
import ServerDetailView from "@/components/servers/ServerDetailView.vue";
import { useDashboard } from "@/composables/useDashboard";

const route = useRoute();
const router = useRouter();

const {
  actionLoading,
  config,
  filteredLogs,
  handleDeleteServer,
  handleEditServer,
  handleExit,
  handleJoin,
  handleRejoin,
  serverStatusMap,
} = useDashboard();

const serverId = computed(() => {
  if ("id" in route.params) {
    return route.params.id as string;
  }
  return "";
});

const server = computed(() => {
  return config.value.servers.find((s) => s.id === serverId.value) || null;
});

const serverStatus = computed(() => {
  return serverStatusMap.value.get(serverId.value) || "disconnected";
});

const serverLogs = computed(() => {
  return filteredLogs.value.filter((l) => l.serverId === serverId.value);
});

const isLoading = computed(() => {
  return actionLoading.value.get(serverId.value) ?? false;
});

async function onDelete() {
  await handleDeleteServer(serverId.value);
  router.push("/");
}
</script>

<template>
  <AppLayout>
    <ServerDetailView
      v-if="server"
      :server="server"
      :status="serverStatus"
      :is-loading="isLoading"
      :logs="serverLogs"
      @edit="handleEditServer(server!)"
      @delete="onDelete"
      @join="handleJoin(serverId)"
      @rejoin="handleRejoin(serverId)"
      @exit="handleExit(serverId)"
    />
    <div v-else class="flex min-h-[50vh] items-center justify-center">
      <p class="text-muted-foreground">Server not found</p>
    </div>
  </AppLayout>
</template>

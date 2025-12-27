<script setup lang="ts">
import DashboardView from "@/components/dashboard/DashboardView.vue";
import AppLayout from "@/components/layout/AppLayout.vue";
import { useDashboard } from "@/composables/useDashboard";

const { config, filteredLogs, serverStatusMap } = useDashboard();
</script>

<template>
  <AppLayout>
    <DashboardView
      :servers="config.servers"
      :server-statuses="serverStatusMap"
      :logs="filteredLogs"
    />
  </AppLayout>
</template>

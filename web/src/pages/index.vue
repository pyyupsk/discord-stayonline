<script setup lang="ts">
import DashboardView from "@/components/dashboard/DashboardView.vue";
import DashboardLayout from "@/components/layout/DashboardLayout.vue";
import { useDashboard } from "@/composables/useDashboard";

const { config, filteredLogs, serverStatusMap } = useDashboard();
</script>

<template>
  <DashboardLayout>
    <DashboardView
      :servers="config.servers"
      :server-statuses="serverStatusMap"
      :logs="filteredLogs"
    />
  </DashboardLayout>
</template>

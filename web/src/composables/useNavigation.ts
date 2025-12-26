import { computed, ref } from "vue";

import type { NavigationView } from "@/types";

const currentView = ref<NavigationView>("dashboard");
const selectedServerId = ref<null | string>(null);

export function useNavigation() {
  const isDashboard = computed(() => currentView.value === "dashboard");
  const isServerView = computed(() => currentView.value === "server");
  const isActivityView = computed(() => currentView.value === "activity");

  function navigateToDashboard() {
    currentView.value = "dashboard";
    selectedServerId.value = null;
  }

  function navigateToServer(serverId: string) {
    currentView.value = "server";
    selectedServerId.value = serverId;
  }

  function navigateToActivity() {
    currentView.value = "activity";
    selectedServerId.value = null;
  }

  function selectServer(serverId: null | string) {
    if (serverId) {
      navigateToServer(serverId);
    } else {
      navigateToDashboard();
    }
  }

  return {
    currentView,
    isActivityView,
    isDashboard,
    isServerView,
    navigateToActivity,
    navigateToDashboard,
    navigateToServer,
    selectedServerId,
    selectServer,
  };
}

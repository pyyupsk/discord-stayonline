import { computed } from "vue";
import { useRoute, useRouter } from "vue-router";

export function useNavigation() {
  const route = useRoute();
  const router = useRouter();

  const isDashboard = computed(() => route.path === "/");
  const isServerView = computed(() => route.path.startsWith("/servers/"));
  const isActivityView = computed(() => route.path === "/activity");

  const selectedServerId = computed(() => {
    if (isServerView.value && "id" in route.params) {
      return route.params.id as string;
    }
    return null;
  });

  function navigateToDashboard() {
    router.push("/");
  }

  function navigateToServer(serverId: string) {
    router.push(`/servers/${serverId}`);
  }

  function navigateToActivity() {
    router.push("/activity");
  }

  return {
    isActivityView,
    isDashboard,
    isServerView,
    navigateToActivity,
    navigateToDashboard,
    navigateToServer,
    selectedServerId,
  };
}

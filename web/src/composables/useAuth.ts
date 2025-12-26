import { storeToRefs } from "pinia";

import { useAuthStore } from "@/stores";

export function useAuth() {
  const store = useAuthStore();
  const { authenticated, authRequired, error, loading } = storeToRefs(store);

  return {
    authenticated,
    authRequired,
    checkAuth: store.checkAuth,
    error,
    loading,
    login: store.login,
    logout: store.logout,
  };
}

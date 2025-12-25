import { ref } from "vue";

const authenticated = ref(false);
const authRequired = ref(false);
const loading = ref(false);
const error = ref<null | string>(null);

export function useAuth() {
  async function checkAuth() {
    loading.value = true;
    error.value = null;

    try {
      const response = await fetch("/api/auth/check");
      if (!response.ok) {
        throw new Error("Failed to check authentication");
      }

      const data = await response.json();
      authenticated.value = data.authenticated;
      authRequired.value = data.auth_required;
    } catch (err) {
      error.value = err instanceof Error ? err.message : "Unknown error";
      authenticated.value = false;
    } finally {
      loading.value = false;
    }
  }

  async function login(apiKey: string) {
    loading.value = true;
    error.value = null;

    try {
      const response = await fetch("/api/auth/login", {
        body: JSON.stringify({ api_key: apiKey }),
        headers: { "Content-Type": "application/json" },
        method: "POST",
      });

      const data = await response.json();

      if (!response.ok) {
        throw new Error(data.message || "Login failed");
      }

      authenticated.value = true;
      return true;
    } catch (err) {
      error.value = err instanceof Error ? err.message : "Unknown error";
      return false;
    } finally {
      loading.value = false;
    }
  }

  async function logout() {
    loading.value = true;
    error.value = null;

    try {
      const response = await fetch("/api/auth/logout", {
        method: "POST",
      });

      if (!response.ok) {
        throw new Error("Logout failed");
      }

      authenticated.value = false;
      return true;
    } catch (err) {
      error.value = err instanceof Error ? err.message : "Unknown error";
      return false;
    } finally {
      loading.value = false;
    }
  }

  return {
    authenticated,
    authRequired,
    checkAuth,
    error,
    loading,
    login,
    logout,
  };
}

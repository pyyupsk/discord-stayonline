import { createRouter, createWebHistory } from "vue-router";
import { routes } from "vue-router/auto-routes";

import { useAuthStore } from "@/stores";

export const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes,
  scrollBehavior(_to, _from, savedPosition) {
    return savedPosition ?? { top: 0 };
  },
  stringifyQuery: (query) => new URLSearchParams(query as Record<string, string>).toString(),
});

router.beforeEach(async (to, _from, next) => {
  const authStore = useAuthStore();

  if (!authStore.authenticated && !authStore.authRequired) {
    await authStore.checkAuth();
  }

  const isLoginPage = to.path === "/login";
  const needsAuth = authStore.authRequired && !authStore.authenticated;

  if (isLoginPage && authStore.authenticated) {
    next("/");
  } else if (!isLoginPage && needsAuth) {
    next("/login");
  } else {
    next();
  }
});

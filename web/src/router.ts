import { createRouter, createWebHistory } from "vue-router";
import { routes } from "vue-router/auto-routes";

import { useAuth } from "@/composables/useAuth";

export const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes,
  scrollBehavior(_to, _from, savedPosition) {
    return savedPosition ?? { top: 0 };
  },
  stringifyQuery: (query) => new URLSearchParams(query as Record<string, string>).toString(),
});

router.beforeEach(async (to, _from, next) => {
  const { authenticated, authRequired, checkAuth } = useAuth();

  if (!authenticated.value && !authRequired.value) {
    await checkAuth();
  }

  const isLoginPage = to.path === "/login";
  const needsAuth = authRequired.value && !authenticated.value;

  if (isLoginPage && authenticated.value) {
    next("/");
  } else if (!isLoginPage && needsAuth) {
    next("/login");
  } else {
    next();
  }
});

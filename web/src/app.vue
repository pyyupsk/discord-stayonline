<script setup lang="ts">
import { useColorMode } from "@vueuse/core";
import { RouterView } from "vue-router";

import { Toaster } from "@/components/ui/sonner";

// Initialize color mode at app level so it applies to all pages
useColorMode();
</script>

<template>
  <RouterView />
  <Toaster />
</template>

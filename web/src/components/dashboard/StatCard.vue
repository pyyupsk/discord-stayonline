<script setup lang="ts">
import type { Component } from "vue";

defineProps<{
  icon: Component;
  subtitle?: string;
  title: string;
  value: string;
  variant?: "default" | "destructive" | "primary" | "success" | "warning";
}>();

const variantStyles = {
  default: {
    bg: "bg-muted/50",
    icon: "text-muted-foreground",
  },
  destructive: {
    bg: "bg-red-500/10",
    icon: "text-red-500",
  },
  primary: {
    bg: "bg-primary/10",
    icon: "text-primary",
  },
  success: {
    bg: "bg-green-500/10",
    icon: "text-green-500",
  },
  warning: {
    bg: "bg-yellow-500/10",
    icon: "text-yellow-500",
  },
};
</script>

<template>
  <div class="bg-card border-border/50 rounded-lg border p-4">
    <div class="flex items-center gap-3">
      <div class="rounded-lg p-2.5" :class="variantStyles[variant || 'default'].bg">
        <component :is="icon" class="size-5" :class="variantStyles[variant || 'default'].icon" />
      </div>
      <div class="min-w-0 flex-1">
        <p class="text-muted-foreground truncate text-xs font-medium tracking-wide uppercase">
          {{ title }}
        </p>
        <p class="truncate text-xl font-semibold tabular-nums">{{ value }}</p>
      </div>
    </div>
    <p v-if="subtitle" class="text-muted-foreground mt-2 text-xs">{{ subtitle }}</p>
  </div>
</template>

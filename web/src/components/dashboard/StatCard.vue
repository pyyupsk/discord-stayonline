<script setup lang="ts">
import type { Component } from "vue";

defineProps<{
  icon: Component;
  subtitle?: string;
  title: string;
  value: string;
  variant?: "default" | "destructive" | "primary" | "success" | "warning";
}>();
</script>

<template>
  <div class="border-border/50 bg-card rounded-xl border p-4">
    <div class="flex items-start justify-between">
      <div>
        <p class="text-muted-foreground text-sm">{{ title }}</p>
        <p class="mt-1 text-2xl font-bold">{{ value }}</p>
        <p v-if="subtitle" class="text-muted-foreground mt-0.5 text-xs">{{ subtitle }}</p>
      </div>
      <div
        class="rounded-lg p-2"
        :class="{
          'bg-muted': variant === 'default' || !variant,
          'bg-primary/20': variant === 'primary',
          'bg-green-500/20': variant === 'success',
          'bg-yellow-500/20': variant === 'warning',
          'bg-destructive/20': variant === 'destructive',
        }"
      >
        <component
          :is="icon"
          class="h-5 w-5"
          :class="{
            'text-muted-foreground': variant === 'default' || !variant,
            'text-primary': variant === 'primary',
            'text-green-500': variant === 'success',
            'text-yellow-500': variant === 'warning',
            'text-destructive': variant === 'destructive',
          }"
        />
      </div>
    </div>
  </div>
</template>

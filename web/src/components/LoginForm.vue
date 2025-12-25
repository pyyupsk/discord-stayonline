<script setup lang="ts">
import { KeyRound, LogIn } from "lucide-vue-next";
import { ref } from "vue";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useAuth } from "@/composables/useAuth";

const { error, loading, login } = useAuth();

const apiKey = ref("");

async function handleSubmit() {
  if (!apiKey.value.trim()) return;
  await login(apiKey.value.trim());
}
</script>

<template>
  <div class="flex min-h-screen items-center justify-center p-4">
    <Card class="w-full max-w-md">
      <CardHeader class="text-center">
        <div
          class="bg-primary/10 mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full"
        >
          <KeyRound class="text-primary h-6 w-6" />
        </div>
        <CardTitle class="text-xl">Discord Stay Online</CardTitle>
        <p class="text-muted-foreground text-sm">Enter your API key to access the dashboard</p>
      </CardHeader>
      <CardContent>
        <form class="space-y-4" @submit.prevent="handleSubmit">
          <div class="space-y-2">
            <Label for="api-key">API Key</Label>
            <Input
              id="api-key"
              v-model="apiKey"
              type="password"
              placeholder="Enter your API key"
              autocomplete="current-password"
              required
            />
          </div>

          <p v-if="error" class="text-destructive text-sm">
            {{ error }}
          </p>

          <Button type="submit" class="w-full" :disabled="loading || !apiKey.trim()">
            <LogIn />
            {{ loading ? "Logging in..." : "Login" }}
          </Button>
        </form>
      </CardContent>
    </Card>
  </div>
</template>

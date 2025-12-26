import { storeToRefs } from "pinia";

import { useConfigStore } from "@/stores";

export function useConfig() {
  const store = useConfigStore();
  const { config, error, loading } = storeToRefs(store);

  return {
    acknowledgeTos: store.acknowledgeTos,
    addServer: store.addServer,
    config,
    deleteServer: store.deleteServer,
    error,
    loadConfig: store.loadConfig,
    loading,
    saveConfig: store.saveConfig,
    setConfig: store.setConfig,
    updateServer: store.updateServer,
    updateStatus: store.updateStatus,
  };
}

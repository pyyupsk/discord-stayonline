import { storeToRefs } from "pinia";

import { useWebSocketStore } from "@/stores";

export function useWebSocket() {
  const store = useWebSocketStore();
  const { filteredLogs, logFilter, logs, serverStatuses, wsStatus } = storeToRefs(store);

  return {
    addLog: store.addLog,
    clearLogs: store.clearLogs,
    connect: store.connect,
    disconnect: store.disconnect,
    filteredLogs,
    getServerStatus: store.getServerStatus,
    loadLogs: store.loadLogs,
    loadStatuses: store.loadStatuses,
    logFilter,
    logs,
    serverStatuses,
    setLogFilter: store.setLogFilter,
    setOnConfigChanged: store.setOnConfigChanged,
    updateServerNamesFromConfig: store.updateServerNamesFromConfig,
    wsStatus,
  };
}

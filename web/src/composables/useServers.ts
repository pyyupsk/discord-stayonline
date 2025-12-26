import { storeToRefs } from "pinia";

import { useServersStore } from "@/stores";

export function useServers() {
  const store = useServersStore();
  const { actionLoading } = storeToRefs(store);

  return {
    actionLoading,
    exitServer: store.exitServer,
    isLoading: store.isLoading,
    joinServer: store.joinServer,
    rejoinServer: store.rejoinServer,
  };
}

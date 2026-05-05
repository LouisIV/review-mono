import { useState, useEffect, useCallback } from "react";
import { Store } from "@tauri-apps/plugin-store";
import type { SshHost } from "./useDaemonDiscovery";

export function useSettings() {
  const [sshHosts, setSshHosts] = useState<SshHost[]>([]);
  const [store, setStore] = useState<Store | null>(null);

  useEffect(() => {
    Store.load("settings.json").then(setStore);
  }, []);

  useEffect(() => {
    if (!store) return;
    store.get<SshHost[]>("ssh_hosts").then((hosts) => {
      if (hosts) setSshHosts(hosts);
    });
  }, [store]);

  const persist = useCallback(
    async (hosts: SshHost[]) => {
      if (!store) return;
      setSshHosts(hosts);
      await store.set("ssh_hosts", hosts);
      await store.save();
    },
    [store]
  );

  const addHost = useCallback(
    (host: SshHost) => persist([...sshHosts, host]),
    [sshHosts, persist]
  );

  const removeHost = useCallback(
    (label: string) => persist(sshHosts.filter((h) => h.label !== label)),
    [sshHosts, persist]
  );

  const updateHost = useCallback(
    (label: string, updated: SshHost) =>
      persist(sshHosts.map((h) => (h.label === label ? updated : h))),
    [sshHosts, persist]
  );

  return { sshHosts, addHost, removeHost, updateHost };
}

import { useState, useCallback } from "react";
import { invoke } from "@tauri-apps/api/core";

export interface SshHost {
  label: string;
  hostname: string;
  ssh_port: number;
  daemon_port: number;
}

export interface DaemonInfo {
  host: string;
  port: number;
  version: string;
  build_id: string;
  pid: number;
  source: { type: string; host?: string } | "Local";
}

export function useDaemonDiscovery() {
  const [daemons, setDaemons] = useState<DaemonInfo[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const discover = useCallback(async (sshHosts: SshHost[]) => {
    setLoading(true);
    setError(null);
    try {
      const result = await invoke<DaemonInfo[]>("discover_daemons", { sshHosts });
      setDaemons(result);
    } catch (e) {
      setError(String(e));
    } finally {
      setLoading(false);
    }
  }, []);

  return { daemons, loading, error, discover };
}

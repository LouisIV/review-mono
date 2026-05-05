import { useState } from "react";
import type { SshHost } from "../hooks/useDaemonDiscovery";

interface Props {
  onSubmit: (host: SshHost) => void;
  initial?: SshHost;
  onCancel?: () => void;
}

export default function SshHostForm({ onSubmit, initial, onCancel }: Props) {
  const [label, setLabel] = useState(initial?.label ?? "");
  const [hostname, setHostname] = useState(initial?.hostname ?? "");
  const [sshPort, setSshPort] = useState(initial?.ssh_port ?? 22);
  const [daemonPort, setDaemonPort] = useState(initial?.daemon_port ?? 7080);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onSubmit({ label, hostname, ssh_port: sshPort, daemon_port: daemonPort });
    if (!initial) { setLabel(""); setHostname(""); setSshPort(22); setDaemonPort(7080); }
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-3 p-4 border rounded-lg bg-white">
      <div>
        <label className="block text-sm font-medium text-gray-700">Label</label>
        <input value={label} onChange={(e) => setLabel(e.target.value)}
          className="mt-1 w-full rounded border px-3 py-1.5 text-sm" placeholder="e.g. work-server" required />
      </div>
      <div>
        <label className="block text-sm font-medium text-gray-700">Hostname</label>
        <input value={hostname} onChange={(e) => setHostname(e.target.value)}
          className="mt-1 w-full rounded border px-3 py-1.5 text-sm" placeholder="e.g. dev.example.com" required />
      </div>
      <div className="flex gap-4">
        <div className="flex-1">
          <label className="block text-sm font-medium text-gray-700">SSH Port</label>
          <input type="number" value={sshPort} onChange={(e) => setSshPort(Number(e.target.value))}
            className="mt-1 w-full rounded border px-3 py-1.5 text-sm" />
        </div>
        <div className="flex-1">
          <label className="block text-sm font-medium text-gray-700">Daemon Port</label>
          <input type="number" value={daemonPort} onChange={(e) => setDaemonPort(Number(e.target.value))}
            className="mt-1 w-full rounded border px-3 py-1.5 text-sm" />
        </div>
      </div>
      <div className="flex gap-2">
        <button type="submit" className="rounded bg-blue-600 px-4 py-1.5 text-sm text-white hover:bg-blue-700">
          {initial ? "Update" : "Add Host"}
        </button>
        {onCancel && <button type="button" onClick={onCancel}
          className="rounded border px-4 py-1.5 text-sm hover:bg-gray-50">Cancel</button>}
      </div>
    </form>
  );
}

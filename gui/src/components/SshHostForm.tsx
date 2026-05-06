import { useState } from "react";
import type { SshHost } from "@/hooks/useDaemonDiscovery";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent } from "@/components/ui/card";
import { Check, X } from "lucide-react";

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
    <Card>
      <CardContent className="p-4">
        <form onSubmit={handleSubmit} className="space-y-3">
          <div>
            <label className="block text-sm font-medium mb-1">Label</label>
            <Input value={label} onChange={(e) => setLabel(e.target.value)}
              placeholder="e.g. work-server" required />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">Hostname</label>
            <Input value={hostname} onChange={(e) => setHostname(e.target.value)}
              placeholder="e.g. dev.example.com" required />
          </div>
          <div className="flex gap-4">
            <div className="flex-1">
              <label className="block text-sm font-medium mb-1">SSH Port</label>
              <Input type="number" value={sshPort} onChange={(e) => setSshPort(Number(e.target.value))} />
            </div>
            <div className="flex-1">
              <label className="block text-sm font-medium mb-1">Daemon Port</label>
              <Input type="number" value={daemonPort} onChange={(e) => setDaemonPort(Number(e.target.value))} />
            </div>
          </div>
          <div className="flex gap-2 pt-1">
            <Button type="submit" size="sm">
              <Check className="h-4 w-4" />
              {initial ? "Update" : "Add Host"}
            </Button>
            {onCancel && (
              <Button type="button" variant="outline" size="sm" onClick={onCancel}>
                <X className="h-4 w-4" />
                Cancel
              </Button>
            )}
          </div>
        </form>
      </CardContent>
    </Card>
  );
}

import { useState } from "react";
import { Trash2, Pencil, Plus, Server } from "lucide-react";
import { useSettings } from "@/hooks/useSettings";
import SshHostForm from "@/components/SshHostForm";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from "@/components/ui/dialog";

export default function SettingsPage() {
  const { sshHosts, addHost, removeHost, updateHost } = useSettings();
  const [editing, setEditing] = useState<string | null>(null);
  const [showDialog, setShowDialog] = useState(false);

  return (
    <div className="p-6 max-w-2xl">
      <div className="flex items-center gap-2 mb-1">
        <Server className="h-5 w-5 text-muted-foreground" />
        <h2 className="text-xl font-bold">SSH Hosts</h2>
      </div>
      <CardDescription className="mb-6">
        Configure remote machines running the review daemon. The GUI will discover them via SSH tunneling.
      </CardDescription>

      {sshHosts.length === 0 && !showDialog && (
        <p className="text-sm text-muted-foreground italic mb-4">No SSH hosts configured.</p>
      )}

      <div className="space-y-3">
        {sshHosts.map((host) => (
          editing === host.label ? (
            <SshHostForm key={host.label} initial={host}
              onSubmit={(h) => { updateHost(host.label, h); setEditing(null); }}
              onCancel={() => setEditing(null)} />
          ) : (
            <Card key={host.label}>
              <CardContent className="p-4 flex items-center justify-between">
                <div>
                  <div className="font-medium">{host.label}</div>
                  <div className="text-sm text-muted-foreground">
                    {host.hostname}:{host.daemon_port}
                    <span className="text-xs ml-2">SSH:{host.ssh_port}</span>
                  </div>
                </div>
                <div className="flex gap-2">
                  <Button variant="ghost" size="sm" onClick={() => setEditing(host.label)}>
                    <Pencil className="h-4 w-4" />
                    Edit
                  </Button>
                  <Button variant="ghost" size="sm" onClick={() => removeHost(host.label)}>
                    <Trash2 className="h-4 w-4 text-destructive" />
                    <span className="text-destructive">Remove</span>
                  </Button>
                </div>
              </CardContent>
            </Card>
          )
        ))}
      </div>

      <Separator className="my-6" />

      {!showDialog ? (
        <Button onClick={() => setShowDialog(true)}>
          <Plus className="h-4 w-4" />
          Add SSH Host
        </Button>
      ) : (
        <Dialog open={showDialog} onOpenChange={setShowDialog}>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Add SSH Host</DialogTitle>
              <DialogDescription>
                Enter the connection details for a remote machine running the review daemon.
              </DialogDescription>
            </DialogHeader>
            <SshHostForm onSubmit={(h) => { addHost(h); setShowDialog(false); }} onCancel={() => setShowDialog(false)} />
          </DialogContent>
        </Dialog>
      )}
    </div>
  );
}

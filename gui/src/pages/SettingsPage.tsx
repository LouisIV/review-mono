import { useState } from "react";
import { useSettings } from "../hooks/useSettings";
import SshHostForm from "../components/SshHostForm";

export default function SettingsPage() {
  const { sshHosts, addHost, removeHost, updateHost } = useSettings();
  const [editing, setEditing] = useState<string | null>(null);
  const [showForm, setShowForm] = useState(false);

  return (
    <div className="p-6 max-w-2xl">
      <h2 className="text-xl font-bold mb-4">SSH Hosts</h2>
      <p className="text-sm text-gray-600 mb-4">
        Configure remote machines running the review daemon. The GUI will discover them via SSH tunneling.
      </p>

      {sshHosts.length === 0 && !showForm && (
        <p className="text-sm text-gray-400 italic mb-4">No SSH hosts configured.</p>
      )}

      <div className="space-y-3">
        {sshHosts.map((host) => (
          editing === host.label ? (
            <SshHostForm key={host.label} initial={host}
              onSubmit={(h) => { updateHost(host.label, h); setEditing(null); }}
              onCancel={() => setEditing(null)} />
          ) : (
            <div key={host.label} className="flex items-center justify-between p-3 border rounded-lg bg-white">
              <div>
                <span className="font-medium">{host.label}</span>
                <span className="text-sm text-gray-500 ml-3">{host.hostname}:{host.daemon_port}</span>
                <span className="text-xs text-gray-400 ml-2">SSH:{host.ssh_port}</span>
              </div>
              <div className="flex gap-2">
                <button onClick={() => setEditing(host.label)}
                  className="text-xs text-blue-600 hover:underline">Edit</button>
                <button onClick={() => removeHost(host.label)}
                  className="text-xs text-red-600 hover:underline">Remove</button>
              </div>
            </div>
          ))
        )}
      </div>

      {showForm ? (
        <div className="mt-4">
          <SshHostForm onSubmit={(h) => { addHost(h); setShowForm(false); }}
            onCancel={() => setShowForm(false)} />
        </div>
      ) : (
        <button onClick={() => setShowForm(true)}
          className="mt-4 rounded bg-blue-600 px-4 py-1.5 text-sm text-white hover:bg-blue-700">
          Add SSH Host
        </button>
      )}
    </div>
  );
}

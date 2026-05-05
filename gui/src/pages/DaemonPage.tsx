import { useState, useEffect } from "react";
import { useDaemonDiscovery } from "../hooks/useDaemonDiscovery";
import { useSettings } from "../hooks/useSettings";
import { useDaemonApi } from "../hooks/useDaemonApi";

export default function DaemonPage() {
  const { sshHosts } = useSettings();
  const { daemons, loading, error, discover } = useDaemonDiscovery();
  const [selectedDaemon, setSelectedDaemon] = useState<number | null>(null);
  const [repoPath, setRepoPath] = useState(".");
  const [repoInput, setRepoInput] = useState(".");
  const [selectedSession, setSelectedSession] = useState<string | null>(null);
  const [diffFile, setDiffFile] = useState<string | null>(null);
  const [newComment, setNewComment] = useState("");
  const [commentFile, setCommentFile] = useState<string | null>(null);
  const [actionMsg, setActionMsg] = useState("");

  const daemon = selectedDaemon !== null ? daemons[selectedDaemon] : null;
  const baseUrl = daemon ? `http://${daemon.host}:${daemon.port}` : "";
  const api = useDaemonApi(baseUrl, repoPath);

  useEffect(() => { discover(sshHosts); }, []);

  const handleRefresh = () => discover(sshHosts);

  const handleBrowseRepo = () => {
    setRepoPath(repoInput);
    setSelectedSession(null);
  };

  return (
    <div className="flex h-full">
      {/* Sidebar */}
      <div className="w-72 border-r bg-gray-50 p-4 flex flex-col gap-3 overflow-y-auto">
        <div className="flex items-center justify-between">
          <h2 className="font-bold text-lg">Daemons</h2>
          <button onClick={handleRefresh} disabled={loading}
            className="text-xs bg-blue-600 text-white px-2 py-1 rounded hover:bg-blue-700 disabled:opacity-50">
            {loading ? "Scanning..." : "Refresh"}
          </button>
        </div>

        {error && <p className="text-xs text-red-600">{error}</p>}

        {daemons.length === 0 && !loading && (
          <p className="text-sm text-gray-400">No daemons found.</p>
        )}

        {daemons.map((d, i) => (
          <button key={i} onClick={() => { setSelectedDaemon(i); setSelectedSession(null); }}
            className={`text-left p-2 rounded border text-sm ${selectedDaemon === i ? "border-blue-500 bg-blue-50" : "border-gray-200 hover:bg-white"}`}>
            <div className="font-medium">{d.host}:{d.port}</div>
            <div className="text-xs text-gray-500">v{d.version} — pid {d.pid}</div>
            <div className="text-xs text-gray-400">{typeof d.source === "string" ? d.source : d.source.type}</div>
          </button>
        ))}

        {daemon && (
          <div className="mt-4 pt-4 border-t">
            <label className="text-xs font-medium text-gray-600">Repo path</label>
            <div className="flex gap-1 mt-1">
              <input value={repoInput} onChange={(e) => setRepoInput(e.target.value)}
                className="flex-1 rounded border px-2 py-1 text-xs" placeholder="e.g. /home/user/project" />
              <button onClick={handleBrowseRepo}
                className="text-xs bg-gray-600 text-white px-2 py-1 rounded hover:bg-gray-700">Go</button>
            </div>
            <button onClick={() => api.fetchSessions()}
              className="mt-2 w-full text-xs bg-green-600 text-white px-2 py-1 rounded hover:bg-green-700">
              Browse Sessions
            </button>

            {api.sessions.map((s) => (
              <button key={s.id} onClick={() => { api.fetchSession(s.id); setSelectedSession(s.id); setDiffFile(null); }}
                className={`mt-1 w-full text-left p-2 rounded border text-xs ${selectedSession === s.id ? "border-blue-500 bg-blue-50" : "border-gray-200 hover:bg-white"}`}>
                <div className="font-medium">{s.branch} → {s.base}</div>
                <div className="text-gray-500">{s.status}</div>
              </button>
            ))}
          </div>
        )}
      </div>

      {/* Main content */}
      <div className="flex-1 p-4 overflow-y-auto">
        {api.loading && <p className="text-sm text-gray-400">Loading...</p>}
        {api.error && <p className="text-sm text-red-600">{api.error}</p>}

        {api.session && !diffFile && (
          <div>
            <div className="flex items-center justify-between mb-4">
              <div>
                <h3 className="text-lg font-bold">{api.session.branch} → {api.session.base}</h3>
                <p className="text-sm text-gray-500">Status: {api.session.status}</p>
              </div>
              <div className="flex gap-2">
                <button onClick={async () => {
                  try { await api.approve(); setActionMsg("Approved!"); } catch (e) { setActionMsg(`Error: ${e}`); }
                }} className="text-xs bg-green-600 text-white px-3 py-1 rounded hover:bg-green-700">Approve</button>
                <button onClick={async () => {
                  try { await api.requestChanges("Changes requested via GUI"); setActionMsg("Changes requested"); } catch (e) { setActionMsg(`Error: ${e}`); }
                }} className="text-xs bg-red-600 text-white px-3 py-1 rounded hover:bg-red-700">Request Changes</button>
              </div>
            </div>
            {actionMsg && <p className="text-sm mb-3 text-blue-600">{actionMsg}</p>}

            <h4 className="font-medium mb-2">Commits ({api.commits.length})</h4>
            <div className="space-y-1 mb-4">
              {api.commits.map((c, i) => (
                <div key={i} className="text-xs p-2 bg-gray-50 rounded">
                  <span className="font-mono text-blue-600">{c.hash}</span>
                  <span className="ml-2">{c.message}</span>
                  <span className="ml-2 text-gray-400">{c.author}</span>
                </div>
              ))}
            </div>

            <h4 className="font-medium mb-2">Files Changed ({api.diffFiles.length})</h4>
            <div className="space-y-1">
              {api.diffFiles.map((f) => (
                <button key={f.path} onClick={() => setDiffFile(f.path)}
                  className="block w-full text-left text-xs p-2 rounded border border-gray-200 hover:bg-gray-50">
                  <span className="font-mono">{f.path}</span>
                  <span className="ml-2 text-green-600">+{f.additions}</span>
                  <span className="ml-1 text-red-600">-{f.deletions}</span>
                </button>
              ))}
            </div>

            {/* Comments */}
            <h4 className="font-medium mt-4 mb-2">Comments ({api.comments.length})</h4>
            <div className="space-y-2">
              {api.comments.map((c) => (
                <div key={c.id} className={`p-2 rounded border text-xs ${c.resolved ? "bg-gray-100 border-gray-200" : "bg-yellow-50 border-yellow-200"}`}>
                  <div className="flex justify-between">
                    <span className="font-medium">{c.author}</span>
                    <span className="text-gray-400">{c.file}{c.line ? `:${c.line}` : ""}</span>
                  </div>
                  <p className="mt-1">{c.body}</p>
                  {!c.resolved && (
                    <button onClick={() => api.resolveComment(api.session!.id, c.id)}
                      className="mt-1 text-xs text-blue-600 hover:underline">Resolve</button>
                  )}
                  {c.resolved && <span className="text-xs text-green-600">✓ Resolved</span>}
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

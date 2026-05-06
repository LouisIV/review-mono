import { useState, useEffect } from "react";
import {
  RefreshCw, Plus, Check, X, MessageSquare, GitBranch, FileCode,
  Server, ArrowRight, ArrowLeft
} from "lucide-react";
import { useDaemonDiscovery } from "@/hooks/useDaemonDiscovery";
import { useSettings } from "@/hooks/useSettings";
import { useDaemonApi } from "@/hooks/useDaemonApi";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";
import { Textarea } from "@/components/ui/textarea";

function statusVariant(status: string): "default" | "destructive" | "outline" | "secondary" | "success" {
  switch (status) {
    case "approved": return "success";
    case "changes_requested": return "destructive";
    case "in_review": return "default";
    default: return "secondary";
  }
}

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

  const handleBrowseRepo = () => {
    setRepoPath(repoInput);
    setSelectedSession(null);
  };

  return (
    <div className="flex h-full">
      {/* Sidebar */}
      <div className="w-72 border-r bg-muted/50 p-4 flex flex-col gap-3 overflow-y-auto">
        <div className="flex items-center justify-between">
          <h2 className="font-bold text-lg">Daemons</h2>
          <Button variant="outline" size="sm" onClick={() => discover(sshHosts)} disabled={loading}>
            <RefreshCw className={`h-3.5 w-3.5 ${loading ? "animate-spin" : ""}`} />
            {loading ? "Scanning" : "Refresh"}
          </Button>
        </div>

        {error && <p className="text-xs text-destructive">{error}</p>}

        {daemons.length === 0 && !loading && (
          <p className="text-sm text-muted-foreground">No daemons found.</p>
        )}

        {daemons.map((d, i) => (
          <button
            key={i}
            onClick={() => { setSelectedDaemon(i); setSelectedSession(null); }}
            className={`text-left p-3 rounded-lg border text-sm transition-colors ${selectedDaemon === i
              ? "border-primary bg-primary/5 shadow-sm"
              : "border-border hover:bg-background hover:shadow-sm"
            }`}
          >
            <div className="flex items-center gap-2">
              <Server className="h-4 w-4 text-muted-foreground" />
              <span className="font-medium">{d.host}:{d.port}</span>
            </div>
            <div className="text-xs text-muted-foreground mt-1">
              v{d.version} — pid {d.pid}
            </div>
            <div className="text-xs text-muted-foreground">
              {typeof d.source === "string" ? d.source : d.source.type}
            </div>
          </button>
        ))}

        {daemon && (
          <>
            <Separator />
            <div className="space-y-2">
              <label className="text-xs font-medium">Repo path</label>
              <div className="flex gap-1.5">
                <Input
                  value={repoInput}
                  onChange={(e) => setRepoInput(e.target.value)}
                  placeholder="e.g. /home/user/project"
                  className="h-8 text-xs"
                />
                <Button size="sm" variant="secondary" className="h-8" onClick={handleBrowseRepo}>
                  <ArrowRight className="h-3.5 w-3.5" />
                </Button>
              </div>
              <Button
                size="sm"
                variant="default"
                className="w-full"
                onClick={() => api.fetchSessions()}
              >
                <GitBranch className="h-3.5 w-3.5" />
                Browse Sessions
              </Button>

              {api.sessions.map((s) => (
                <button
                  key={s.id}
                  onClick={() => { api.fetchSession(s.id); setSelectedSession(s.id); setDiffFile(null); }}
                  className={`w-full text-left p-2.5 rounded-lg border text-xs transition-colors ${selectedSession === s.id
                    ? "border-primary bg-primary/5"
                    : "border-border hover:bg-background"
                  }`}
                >
                  <div className="font-medium flex items-center gap-1.5">
                    <GitBranch className="h-3 w-3 text-muted-foreground" />
                    {s.branch} → {s.base}
                  </div>
                  <div className="mt-0.5">
                    <Badge variant={statusVariant(s.status)} className="text-[10px] px-1.5 py-0">
                      {s.status}
                    </Badge>
                  </div>
                </button>
              ))}
            </div>
          </>
        )}
      </div>

      {/* Main content */}
      <div className="flex-1 p-6 overflow-y-auto">
        {api.loading && (
          <div className="flex items-center gap-2 text-muted-foreground text-sm">
            <RefreshCw className="h-4 w-4 animate-spin" />
            Loading...
          </div>
        )}
        {api.error && <p className="text-sm text-destructive">{api.error}</p>}

        {!daemon && !api.loading && (
          <div className="flex flex-col items-center justify-center h-full text-muted-foreground gap-2">
            <Server className="h-12 w-12 opacity-20" />
            <p className="text-sm">Select a daemon to browse sessions</p>
          </div>
        )}

        {api.session && !diffFile && (
          <div>
            {/* Session header */}
            <Card className="mb-4">
              <CardHeader className="pb-3">
                <div className="flex items-center justify-between">
                  <div>
                    <CardTitle className="flex items-center gap-2 text-lg">
                      <GitBranch className="h-4 w-4" />
                      {api.session.branch} → {api.session.base}
                    </CardTitle>
                    <div className="flex items-center gap-2 mt-1">
                      <Badge variant={statusVariant(api.session.status)}>
                        {api.session.status}
                      </Badge>
                      <span className="text-xs text-muted-foreground">
                        Updated: {new Date(api.session.updated_at).toLocaleString()}
                      </span>
                    </div>
                  </div>
                  <div className="flex gap-2">
                    <Button
                      size="sm"
                      variant="default"
                      onClick={async () => {
                        try { await api.approve(); setActionMsg("Approved!"); } catch (e) { setActionMsg(`Error: ${e}`); }
                      }}
                    >
                      <Check className="h-4 w-4" />
                      Approve
                    </Button>
                    <Button
                      size="sm"
                      variant="destructive"
                      onClick={async () => {
                        try { await api.requestChanges("Changes requested via GUI"); setActionMsg("Changes requested"); } catch (e) { setActionMsg(`Error: ${e}`); }
                      }}
                    >
                      <X className="h-4 w-4" />
                      Request Changes
                    </Button>
                  </div>
                </div>
              </CardHeader>
              {actionMsg && (
                <CardContent className="pt-0 pb-3">
                  <p className="text-sm text-primary">{actionMsg}</p>
                </CardContent>
              )}
            </Card>

            {/* Commits */}
            <div className="mb-4">
              <h4 className="text-sm font-medium mb-2 flex items-center gap-1.5">
                <GitBranch className="h-3.5 w-3.5 text-muted-foreground" />
                Commits ({api.commits.length})
              </h4>
              <div className="space-y-1">
                {api.commits.map((c, i) => (
                  <Card key={i} className="border-muted">
                    <CardContent className="p-2.5 text-xs flex items-center gap-3">
                      <code className="font-mono text-primary bg-muted px-1.5 py-0.5 rounded text-[11px]">
                        {c.hash}
                      </code>
                      <span className="flex-1 truncate">{c.message}</span>
                      <span className="text-muted-foreground shrink-0">{c.author}</span>
                    </CardContent>
                  </Card>
                ))}
              </div>
            </div>

            {/* Files */}
            <div className="mb-4">
              <h4 className="text-sm font-medium mb-2 flex items-center gap-1.5">
                <FileCode className="h-3.5 w-3.5 text-muted-foreground" />
                Files Changed ({api.diffFiles.length})
              </h4>
              <div className="space-y-1">
                {api.diffFiles.map((f) => (
                  <button
                    key={f.path}
                    onClick={() => setDiffFile(f.path)}
                    className="block w-full text-left"
                  >
                    <Card className="border-muted hover:border-primary/50 hover:bg-accent/50 transition-colors cursor-pointer">
                      <CardContent className="p-2.5 text-xs flex items-center justify-between">
                        <code className="font-mono">{f.path}</code>
                        <div className="flex gap-2 shrink-0">
                          <span className="text-green-600 font-medium">+{f.additions}</span>
                          <span className="text-red-600 font-medium">-{f.deletions}</span>
                        </div>
                      </CardContent>
                    </Card>
                  </button>
                ))}
              </div>
            </div>

            {/* Comments */}
            <div>
              <h4 className="text-sm font-medium mb-2 flex items-center gap-1.5">
                <MessageSquare className="h-3.5 w-3.5 text-muted-foreground" />
                Comments ({api.comments.length})
              </h4>
              <div className="space-y-2">
                {api.comments.map((c) => (
                  <Card key={c.id} className={c.resolved ? "border-muted bg-muted/30" : "border-amber-200 bg-amber-50/50"}>
                    <CardContent className="p-3 text-xs">
                      <div className="flex items-center justify-between mb-1">
                        <div className="flex items-center gap-2">
                          <span className="font-medium">{c.author}</span>
                          <span className="text-muted-foreground">
                            {c.file}{c.line ? `:${c.line}` : ""}
                          </span>
                        </div>
                        {c.resolved ? (
                          <Badge variant="success" className="text-[10px] px-1.5 py-0">Resolved</Badge>
                        ) : (
                          <Button
                            variant="ghost"
                            size="sm"
                            className="h-6 text-xs"
                            onClick={() => api.resolveComment(api.session!.id, c.id)}
                          >
                            <Check className="h-3 w-3" />
                            Resolve
                          </Button>
                        )}
                      </div>
                      <p className="mt-1 text-foreground/80">{c.body}</p>
                    </CardContent>
                  </Card>
                ))}
              </div>
            </div>
          </div>
        )}

        {/* Diff viewer placeholder — shown when a file is selected */}
        {diffFile && api.diffFiles.find(f => f.path === diffFile) && (
          (() => {
            const file = api.diffFiles.find(f => f.path === diffFile)!;
            return (
              <div>
                <div className="flex items-center gap-2 mb-4">
                  <Button variant="ghost" size="sm" onClick={() => setDiffFile(null)}>
                    <ArrowLeft className="h-4 w-4" />
                    Back to files
                  </Button>
                  <code className="text-sm font-mono">{file.path}</code>
                  <div className="flex gap-2 ml-auto">
                    <span className="text-xs text-green-600 font-medium">+{file.additions}</span>
                    <span className="text-xs text-red-600 font-medium">-{file.deletions}</span>
                  </div>
                </div>

                <div className="space-y-0 font-mono text-xs">
                  {file.hunks.map((hunk, hi) => (
                    <div key={hi} className="mb-2">
                      <div className="bg-muted/50 text-muted-foreground px-3 py-1 rounded-t-md text-[11px]">
                        {hunk.header}
                      </div>
                      {hunk.lines.map((line, li) => (
                        <div
                          key={li}
                          className={`px-3 py-0.5 flex ${
                            line.type === "addition"
                              ? "bg-green-100 text-green-900 dark:bg-green-950 dark:text-green-200"
                              : line.type === "deletion"
                                ? "bg-red-100 text-red-900 dark:bg-red-950 dark:text-red-200"
                                : "bg-background text-foreground"
                          }`}
                        >
                          <span className="w-10 text-right text-muted-foreground shrink-0 select-none mr-3">
                            {line.number ?? ""}
                          </span>
                          <span className="whitespace-pre-wrap break-all">{line.content}</span>
                        </div>
                      ))}
                    </div>
                  ))}
                </div>
              </div>
            );
          })()
        )}

        {daemon && !api.session && !api.loading && (
          <div className="flex flex-col items-center justify-center h-full text-muted-foreground gap-2">
            <GitBranch className="h-12 w-12 opacity-20" />
            <p className="text-sm">Browse sessions to view review details</p>
          </div>
        )}
      </div>
    </div>
  );
}

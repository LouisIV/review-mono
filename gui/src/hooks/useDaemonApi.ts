import { useState, useCallback } from "react";

interface Session {
  id: string;
  repo: string;
  branch: string;
  base: string;
  status: string;
  created_at: string;
  updated_at: string;
}

interface Commit {
  hash: string;
  hash_full: string;
  author: string;
  message: string;
  timestamp: string;
}

interface DiffLine {
  type: string;
  number: number | null;
  content: string;
}

interface DiffHunk {
  header: string;
  lines: DiffLine[];
}

interface DiffFile {
  path: string;
  additions: number;
  deletions: number;
  hunks: DiffHunk[];
}

interface Comment {
  id: string;
  file: string;
  line: number | null;
  body: string;
  author: string;
  resolved: boolean;
  created_at: string;
  resolved_at: string | null;
}

export function useDaemonApi(baseUrl: string, repo: string) {
  const [sessions, setSessions] = useState<Session[]>([]);
  const [session, setSession] = useState<Session | null>(null);
  const [commits, setCommits] = useState<Commit[]>([]);
  const [diffFiles, setDiffFiles] = useState<DiffFile[]>([]);
  const [comments, setComments] = useState<Comment[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const api = useCallback(async (path: string, options?: RequestInit) => {
    const res = await fetch(`${baseUrl}${path}`, {
      headers: { "Content-Type": "application/json" },
      ...options,
    });
    if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
    return res.json();
  }, [baseUrl]);

  const fetchSessions = useCallback(async () => {
    setLoading(true);
    try {
      const data = await api(`/sessions?repo_uri=${encodeURIComponent(repo)}`);
      setSessions(data.sessions);
    } catch (e) { setError(String(e)); }
    finally { setLoading(false); }
  }, [api, repo]);

  const fetchSession = useCallback(async (id: string) => {
    setLoading(true);
    try {
      const [s, c, d, cm] = await Promise.all([
        api(`/session/${id}?repo=${encodeURIComponent(repo)}`),
        api(`/session/${id}/commits?repo=${encodeURIComponent(repo)}`),
        api(`/session/${id}/diff?repo=${encodeURIComponent(repo)}&skip_hunks=true`),
        api(`/session/${id}/comments?repo=${encodeURIComponent(repo)}`),
      ]);
      setSession(s);
      setCommits(c.commits);
      setDiffFiles(d.files);
      setComments(cm.comments);
    } catch (e) { setError(String(e)); }
    finally { setLoading(false); }
  }, [api, repo]);

  const addComment = useCallback(async (sessionId: string, file: string, line: number | null, body: string) => {
    const c = await api(`/session/${sessionId}/comments?repo=${encodeURIComponent(repo)}`, {
      method: "POST",
      body: JSON.stringify({ file, line, body, author: "human" }),
    });
    setComments(prev => [...prev, c]);
    return c;
  }, [api, repo]);

  const resolveComment = useCallback(async (sessionId: string, commentId: string) => {
    await api(`/session/${sessionId}/comments/${commentId}?repo=${encodeURIComponent(repo)}`, {
      method: "PATCH",
      body: JSON.stringify({ resolved: true }),
    });
    setComments(prev => prev.map(c => c.id === commentId ? { ...c, resolved: true } : c));
  }, [api, repo]);

  const approve = useCallback(async () => {
    return api(`/approve?repo=${encodeURIComponent(repo)}`, { method: "POST" });
  }, [api, repo]);

  const requestChanges = useCallback(async (message: string) => {
    return api(`/request-changes?repo=${encodeURIComponent(repo)}`, {
      method: "POST",
      body: JSON.stringify({ message }),
    });
  }, [api, repo]);

  return {
    sessions, session, commits, diffFiles, comments, loading, error,
    fetchSessions, fetchSession, addComment, resolveComment, approve, requestChanges,
  };
}

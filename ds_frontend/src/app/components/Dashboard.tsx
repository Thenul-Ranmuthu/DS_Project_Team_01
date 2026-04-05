"use client";

import { useState, useEffect, useCallback } from "react";
import {
  NODES, NodeStatus, UserFile,
  getNodeStatuses, getLamportClock, getUserFiles, uploadFiles, deleteFile
} from "../lib/api";

interface Props {
  email: string;
  name: string;
  onLogout: () => void;
}

export default function Dashboard({ email, name, onLogout }: Props) {
  const [nodeStatuses, setNodeStatuses] = useState<NodeStatus[]>([]);
  const [clocks, setClocks] = useState<Record<string, number | null>>({});
  const [files, setFiles] = useState<UserFile[]>([]);
  const [filesLoading, setFilesLoading] = useState(true);
  const [uploadFiles_, setUploadFiles_] = useState<File[]>([]);
  const [uploading, setUploading] = useState(false);
  const [uploadMsg, setUploadMsg] = useState("");
  const [dragging, setDragging] = useState(false);
  const [deletingId, setDeletingId] = useState<number | null>(null);
  const [leaderUrl, setLeaderUrl] = useState<string | null>(null);
  const [detectingLeader, setDetectingLeader] = useState(false);

  const refreshAll = useCallback(async () => {
    const statuses = await getNodeStatuses();
    setNodeStatuses(statuses);
    const leader = statuses.find(s => s.is_leader && s.online);
    setLeaderUrl(leader?.url ?? null);

    const clockResults: Record<string, number | null> = {};
    await Promise.all(statuses.map(async (s) => {
      if (s.online) {
        clockResults[s.id] = await getLamportClock(s.url);
      } else {
        clockResults[s.id] = null;
      }
    }));
    setClocks(clockResults);
  }, []);

  const refreshFiles = useCallback(async () => {
    setFilesLoading(true);
    const f = await getUserFiles(email);
    setFiles(f);
    setFilesLoading(false);
  }, [email]);

  useEffect(() => {
    refreshAll();
    refreshFiles();
    const i1 = setInterval(refreshAll, 6000);
    const i2 = setInterval(refreshFiles, 10000);
    return () => { clearInterval(i1); clearInterval(i2); };
  }, [refreshAll, refreshFiles]);

  const handleSelectLeader = async () => {
    setDetectingLeader(true);
    await refreshAll();
    setDetectingLeader(false);
  };

  const handleUpload = async (e: React.FormEvent) => {
    e.preventDefault();
    if (uploadFiles_.length === 0) return;
    setUploading(true);
    setUploadMsg("Locating leader node...");
    try {
      const result = await uploadFiles(email, uploadFiles_);
      setUploadMsg(`${result.message} — Lamport: ${result.lamport_clock}`);
      setUploadFiles_([]);
      await refreshFiles();
      await refreshAll();
    } catch (err: any) {
      setUploadMsg(`Error: ${err.message}`);
    } finally {
      setUploading(false);
    }
  };

  const handleDelete = async (fileId: number) => {
    if (!confirm("Delete this file from the cluster?")) return;
    setDeletingId(fileId);
    try {
      await deleteFile(fileId);
      await refreshFiles();
      await refreshAll();
    } catch (err: any) {
      alert(err.message);
    } finally {
      setDeletingId(null);
    }
  };

  const handleFileChange = (fl: FileList | null) => {
    if (!fl) return;
    setUploadFiles_(prev => [...prev, ...Array.from(fl)]);
  };

  const formatBytes = (b: number) => {
    if (!b) return "0 B";
    const k = 1024, s = ["B","KB","MB","GB"];
    const i = Math.floor(Math.log(b) / Math.log(k));
    return `${(b / Math.pow(k, i)).toFixed(1)} ${s[i]}`;
  };

  const formatDate = (d: string) => new Date(d).toLocaleString();

  const leaderNode = nodeStatuses.find(s => s.is_leader && s.online);
  const onlineCount = nodeStatuses.filter(s => s.online).length;

  return (
    <main className="min-h-screen bg-[#050505] text-white selection:bg-blue-500/30">
      <div className="fixed inset-0 -z-10 bg-[radial-gradient(ellipse_at_top_right,_var(--tw-gradient-stops))] from-blue-900/10 via-black to-black opacity-60"></div>

      <div className="max-w-[1600px] mx-auto p-4 md:p-8 lg:p-12">

        {/* Header */}
        <header className="flex flex-col md:flex-row md:items-center justify-between gap-6 mb-12 px-4">
          <div className="space-y-2">
            <h1 className="text-4xl md:text-5xl font-display font-black tracking-tight flex items-center gap-3 italic">
              <span className="bg-gradient-to-br from-white to-white/40 bg-clip-text text-transparent">DIST</span>
              <span className="text-blue-500">STORE</span>
            </h1>
            <div className="flex items-center gap-3">
              <span className="h-px w-8 bg-blue-500/50"></span>
              <p className="text-[10px] font-mono font-bold tracking-[0.3em] uppercase opacity-50">
                ZooKeeper Election / Lamport Clocks / File Replication
              </p>
            </div>
          </div>
          <div className="flex items-center gap-4">
            <div className="glass px-4 py-2 rounded-xl text-right">
              <p className="text-[9px] font-mono uppercase tracking-widest opacity-30">Logged in as</p>
              <p className="text-sm font-display font-bold">{name}</p>
              <p className="text-[10px] font-mono opacity-40">{email}</p>
            </div>
            <button onClick={onLogout}
              className="px-4 py-2 rounded-xl border border-white/10 text-[10px] font-mono uppercase tracking-widest hover:border-red-500/40 hover:text-red-400 transition-all">
              Logout
            </button>
          </div>
        </header>

        {/* Node Status Bar */}
        <section className="mb-10 animate-in fade-in duration-500">
          <div className="glass rounded-2xl p-6">
            <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 mb-6">
              <div>
                <h2 className="text-xs font-display font-black uppercase tracking-widest flex items-center gap-2">
                  <span className="w-1.5 h-1.5 rounded-full bg-blue-500 animate-pulse"></span>
                  Cluster Status
                </h2>
                <p className="text-[9px] font-mono opacity-30 uppercase tracking-widest mt-1">
                  {onlineCount}/{NODES.length} nodes online
                  {leaderNode ? ` · Leader: ${leaderNode.id}` : " · No leader detected"}
                </p>
              </div>
              <button onClick={handleSelectLeader} disabled={detectingLeader}
                className="px-5 py-2 rounded-xl bg-blue-500/10 border border-blue-500/30 text-blue-400 text-[10px] font-display font-black uppercase tracking-widest hover:bg-blue-500/20 transition-all disabled:opacity-40">
                {detectingLeader ? "Detecting..." : "⚡ Select Leader"}
              </button>
            </div>

            <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
              {NODES.map((node) => {
                const s = nodeStatuses.find(x => x.id === node.id);
                const online = s?.online;
                const isLeader = s?.is_leader;
                const clock = clocks[node.id];

                return (
                  <div key={node.id}
                    className={`relative rounded-2xl p-4 border transition-all ${
                      !online ? "border-red-500/10 bg-red-500/2 opacity-50" :
                      isLeader ? "border-blue-500/40 bg-blue-500/5 shadow-[0_0_30px_rgba(59,130,246,0.1)]" :
                      "border-white/5 bg-white/2"
                    }`}>
                    {isLeader && online && (
                      <div className="absolute top-3 right-3">
                        <span className="text-[8px] font-black font-mono uppercase tracking-widest px-2 py-0.5 rounded-full bg-blue-500 text-white shadow-[0_0_12px_rgba(59,130,246,0.6)]">LEADER</span>
                      </div>
                    )}
                    <div className="flex items-center gap-2 mb-3">
                      <div className={`w-2 h-2 rounded-full ${!online ? "bg-red-500/40" : isLeader ? "bg-blue-500 animate-pulse" : "bg-emerald-500/70"}`}></div>
                      <span className="text-[10px] font-mono font-black uppercase tracking-widest opacity-60">{node.id}</span>
                    </div>
                    <p className="text-[9px] font-mono opacity-30 mb-3">:{node.port}</p>
                    <div className="space-y-1.5">
                      <div className="flex justify-between">
                        <span className="text-[9px] font-mono opacity-30 uppercase">Status</span>
                        <span className={`text-[9px] font-mono font-black uppercase ${!online ? "text-red-400" : isLeader ? "text-blue-400" : "text-emerald-400"}`}>
                          {!online ? "Offline" : isLeader ? "Leader" : "Follower"}
                        </span>
                      </div>
                      <div className="flex justify-between">
                        <span className="text-[9px] font-mono opacity-30 uppercase">Lamport</span>
                        <span className="text-[9px] font-mono font-black text-white/70">
                          {clock !== null && clock !== undefined ? clock : "—"}
                        </span>
                      </div>
                      {s?.znode && (
                        <div className="pt-1 border-t border-white/5">
                          <p className="text-[8px] font-mono opacity-20 truncate" title={s.znode}>{s.znode.split("/").pop()}</p>
                        </div>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          </div>
        </section>

        <div className="grid grid-cols-1 lg:grid-cols-12 gap-8 lg:gap-12">

          {/* Upload Panel */}
          <section className="lg:col-span-3 space-y-6 animate-in fade-in slide-in-from-left-4 duration-700">
            <div className="glass p-8 rounded-3xl relative overflow-hidden group">
              <div className="absolute -top-12 -right-12 w-32 h-32 bg-blue-500/10 rounded-full blur-3xl group-hover:bg-blue-500/20 transition-all pointer-events-none"></div>

              <h2 className="text-xs font-display font-black uppercase tracking-widest mb-6 flex items-center gap-3">
                <span className="w-1.5 h-1.5 rounded-full bg-blue-500 animate-pulse"></span>
                Upload Files
              </h2>

              {/* Leader indicator */}
              <div className={`mb-6 p-3 rounded-xl border text-[9px] font-mono uppercase tracking-widest ${
                leaderNode ? "border-blue-500/20 bg-blue-500/5 text-blue-400" : "border-red-500/20 bg-red-500/5 text-red-400"
              }`}>
                {leaderNode ? `→ ${leaderNode.id} (port ${leaderNode.port})` : "No leader — uploads disabled"}
              </div>

              <form onSubmit={handleUpload} className="space-y-6">
                <div className="relative group/drop">
                  <input id="file-input" type="file" multiple className="hidden"
                    onChange={(e) => handleFileChange(e.target.files)} />
                  <label htmlFor="file-input"
                    className={`flex flex-col items-center justify-center w-full h-44 rounded-2xl border border-dashed cursor-pointer transition-all duration-300 ${
                      dragging ? "border-blue-500 bg-blue-500/10" : "border-white/10 bg-white/3 hover:border-white/20 hover:bg-white/5"
                    }`}
                    onDragOver={(e) => { e.preventDefault(); setDragging(true); }}
                    onDragLeave={() => setDragging(false)}
                    onDrop={(e) => { e.preventDefault(); setDragging(false); handleFileChange(e.dataTransfer.files); }}>
                    <div className="text-center space-y-3 p-6">
                      <div className={`w-12 h-12 rounded-xl mx-auto flex items-center justify-center transition-all ${dragging ? "bg-blue-500" : "bg-white/5"}`}>
                        <svg className={`w-6 h-6 transition-colors ${dragging ? "text-white" : "text-white/30"}`} fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" />
                        </svg>
                      </div>
                      <div>
                        <p className="text-xs font-display font-black uppercase tracking-widest text-white/70">
                          {uploadFiles_.length > 0 ? `${uploadFiles_.length} file(s) queued` : "Drop files here"}
                        </p>
                        <p className="text-[9px] font-mono opacity-20 mt-1">or click to browse</p>
                      </div>
                    </div>
                  </label>
                  {uploadFiles_.length > 0 && (
                    <div className="absolute top-2 right-2 w-5 h-5 rounded-full bg-emerald-500 flex items-center justify-center">
                      <span className="text-[9px] font-black text-black">{uploadFiles_.length}</span>
                    </div>
                  )}
                </div>

                {uploadFiles_.length > 0 && (
                  <div className="space-y-1 max-h-28 overflow-y-auto">
                    {uploadFiles_.map((f, i) => (
                      <div key={i} className="flex items-center justify-between gap-2 p-2 rounded-lg bg-white/3 border border-white/5">
                        <span className="text-[9px] font-mono truncate opacity-60">{f.name}</span>
                        <button type="button" onClick={() => setUploadFiles_(prev => prev.filter((_, j) => j !== i))}
                          className="text-white/30 hover:text-red-400 transition-colors text-xs shrink-0">✕</button>
                      </div>
                    ))}
                  </div>
                )}

                <button type="submit" disabled={uploadFiles_.length === 0 || uploading || !leaderNode}
                  className="w-full h-12 bg-gradient-to-br from-white to-white/70 hover:from-blue-500 hover:to-blue-600 text-black hover:text-white font-display font-black uppercase tracking-[0.2em] text-xs transition-all rounded-2xl active:scale-[0.98] disabled:opacity-20 shadow-xl shadow-black/20">
                  {uploading ? (
                    <span className="flex items-center justify-center gap-2">
                      {[0,150,300].map(d => <span key={d} className="w-1.5 h-1.5 rounded-full bg-current animate-bounce" style={{animationDelay:`${d}ms`}}></span>)}
                    </span>
                  ) : "Upload to Leader"}
                </button>

                {uploadMsg && (
                  <div className="flex items-start gap-2 p-3 rounded-xl bg-white/3 border border-white/5">
                    <div className={`w-1.5 h-1.5 rounded-full mt-0.5 shrink-0 ${uploadMsg.startsWith("Error") ? "bg-red-500" : "bg-blue-500"}`}></div>
                    <p className={`text-[9px] font-mono uppercase tracking-widest leading-relaxed ${uploadMsg.startsWith("Error") ? "text-red-400" : "text-blue-400 opacity-70"}`}>
                      {uploadMsg}
                    </p>
                  </div>
                )}
              </form>
            </div>
          </section>

          {/* Files Panel */}
          <section className="lg:col-span-9 animate-in fade-in slide-in-from-bottom-4 duration-700 delay-150">
            <div className="glass p-1 rounded-3xl min-h-[500px] flex flex-col bg-white/2 shadow-[0_32px_64px_-16px_rgba(0,0,0,0.6)]">
              <div className="p-6 border-b border-white/10 flex items-center justify-between">
                <div>
                  <h2 className="text-sm font-display font-black uppercase tracking-widest">Your Files</h2>
                  <p className="text-[9px] font-mono opacity-20 uppercase tracking-widest mt-1">{email}</p>
                </div>
                <button onClick={refreshFiles}
                  className="text-[9px] font-mono uppercase tracking-widest opacity-30 hover:opacity-70 transition-opacity flex items-center gap-2">
                  <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                  </svg>
                  Refresh
                </button>
              </div>

              <div className="flex-1 p-6">
                {filesLoading ? (
                  <div className="flex flex-col items-center justify-center h-64 gap-4">
                    <div className="flex gap-2">
                      {[0,200,400].map(d => <div key={d} className="w-1.5 h-1.5 rounded-full bg-blue-500 animate-bounce" style={{animationDelay:`${d}ms`}}></div>)}
                    </div>
                    <span className="text-[9px] font-mono uppercase tracking-widest opacity-30 animate-pulse">Fetching files...</span>
                  </div>
                ) : files.length === 0 ? (
                  <div className="flex flex-col items-center justify-center h-64 border border-dashed border-white/5 rounded-2xl">
                    <svg className="w-10 h-10 opacity-10 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                    </svg>
                    <p className="text-[9px] font-mono uppercase tracking-widest opacity-20">No files uploaded yet</p>
                  </div>
                ) : (
                  <div className="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-4">
                    {files.map((file, i) => {
                      const ext = file.original_name.split(".").pop()?.toUpperCase() ?? "FILE";
                      return (
                        <div key={file.ID}
                          className="group relative flex flex-col gap-3 p-4 rounded-2xl border border-white/5 bg-white/2 hover:border-blue-500/30 hover:bg-white/4 transition-all duration-300"
                          style={{ animationDelay: `${i * 40}ms` }}>
                          <div className="flex items-start justify-between">
                            <div className="w-10 h-10 rounded-xl bg-white/5 border border-white/5 flex items-center justify-center group-hover:bg-blue-500/10 group-hover:border-blue-500/20 transition-all">
                              <svg className="w-5 h-5 opacity-30 group-hover:opacity-70 group-hover:text-blue-400 transition-all" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                              </svg>
                            </div>
                            <span className="text-[8px] font-black font-mono uppercase tracking-widest px-2 py-0.5 rounded-full bg-white/5 text-white/40">{ext}</span>
                          </div>

                          <div>
                            <p className="text-sm font-display font-black text-white/90 truncate" title={file.original_name}>{file.original_name}</p>
                            <div className="flex items-center gap-3 mt-1.5">
                              <span className="text-[9px] font-mono opacity-30">{formatBytes(file.file_size)}</span>
                              <span className="w-1 h-1 rounded-full bg-white/10"></span>
                              <span className="text-[9px] font-mono opacity-30">{file.mime_type}</span>
                            </div>
                            <p className="text-[8px] font-mono opacity-20 mt-1">{formatDate(file.CreatedAt)}</p>
                          </div>

                          <div className="flex items-center gap-2 mt-auto">
                            <div className="flex-1 text-[8px] font-mono opacity-20 truncate">ID: {file.ID}</div>
                            <button
                              onClick={() => handleDelete(file.ID)}
                              disabled={deletingId === file.ID}
                              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-red-500/20 text-red-500/50 text-[9px] font-mono uppercase tracking-widest hover:border-red-500 hover:text-red-400 hover:bg-red-500/5 transition-all disabled:opacity-30">
                              {deletingId === file.ID ? "..." : (
                                <>
                                  <svg className="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                                  </svg>
                                  Delete
                                </>
                              )}
                            </button>
                          </div>
                        </div>
                      );
                    })}
                  </div>
                )}
              </div>
            </div>
          </section>
        </div>

        <footer className="mt-24 pt-10 border-t border-white/5 text-[9px] font-mono opacity-20 uppercase tracking-[0.4em] flex flex-col md:flex-row gap-6 justify-between items-center text-center italic">
          <span>PROJECT: DS_TEAM_01</span>
          <span>&copy; 2026 DISTRIBUTED SYSTEMS LABORATORY</span>
          <span>ZOOKEEPER ELECTION: ENABLED</span>
        </footer>
      </div>
    </main>
  );
}

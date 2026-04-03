"use client";

import { useEffect, useState } from "react";

const BACKEND_NODES = [
    "http://localhost:8000",
    "http://localhost:8001",
    "http://localhost:8002",
    "http://localhost:8003",
    "http://localhost:8004",
    "http://localhost:8005",
    "http://localhost:8006",
];

interface FileInfo {
    name: string;
    size: number;
    modTime: string;
    meta_only?: boolean;
}

interface ReadMode {
    writable: boolean;
    mode: "read-write" | "read-only" | "read-only-stale";
    sync_lag_s: number;
    message: string;
}

export default function FileList() {
    const [files, setFiles] = useState<FileInfo[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState("");
    const [activeNode, setActiveNode] = useState(BACKEND_NODES[0]);
    const [previewFile, setPreviewFile] = useState<FileInfo | null>(null);
    const [readMode, setReadMode] = useState<ReadMode | null>(null);

    const formatBytes = (bytes: number, decimals = 2) => {
        if (bytes === 0) return '0 Bytes';
        const k = 1024;
        const dm = decimals < 0 ? 0 : decimals;
        const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB', 'PB', 'EB', 'ZB', 'YB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i];
    };

    const fetchFiles = async () => {
        let success = false;
        for (const nodeUrl of BACKEND_NODES) {
            const controller = new AbortController();
            const timeoutId = setTimeout(() => controller.abort(), 1000);
            try {
                const res = await fetch(`${nodeUrl}/files`, { signal: controller.signal });
                clearTimeout(timeoutId);
                if (!res.ok) continue;
                const data = await res.json();

                const normalizedFiles = (data || []).map((f: any) => {
                    if (typeof f === 'string') {
                        return { name: f, size: 0, modTime: new Date().toISOString() };
                    }
                    return f;
                });

                setFiles(normalizedFiles);
                setActiveNode(nodeUrl);
                success = true;

                // Fetch read-mode from the same node that served the files
                try {
                    const rmRes = await fetch(`${nodeUrl}/read-mode`);
                    if (rmRes.ok) setReadMode(await rmRes.json());
                } catch (_) {}

                break;
            } catch (err) {
                clearTimeout(timeoutId);
                continue;
            }
        }

        if (!success) {
            setError("Network partition or cluster offline.");
        }
        setLoading(false);
    };

    useEffect(() => {
        fetchFiles();
        const interval = setInterval(fetchFiles, 8000);
        return () => clearInterval(interval);
    }, []);

    const DegradationBanner = () => {
        if (!readMode || readMode.mode === "read-write") return null;
        const isStale = readMode.mode === "read-only-stale";
        return (
            <div className={`flex items-start gap-4 p-4 rounded-2xl border mb-6 text-sm
                ${isStale
                    ? "bg-amber-500/5 border-amber-500/20 text-amber-400"
                    : "bg-blue-500/5 border-blue-500/20 text-blue-400"
                }`}>
                <div className={`mt-0.5 w-5 h-5 shrink-0 rounded-full flex items-center justify-center
                    ${isStale ? "bg-amber-500/20" : "bg-blue-500/20"}`}>
                    <svg className="w-3 h-3" fill="currentColor" viewBox="0 0 20 20">
                        <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clipRule="evenodd" />
                    </svg>
                </div>
                <div className="flex-1 min-w-0">
                    <p className="font-mono font-bold text-[11px] uppercase tracking-widest mb-1">
                        {isStale ? `Stale Replica — Sync lag ${readMode.sync_lag_s}s` : "Temporarily Read-Only"}
                    </p>
                    <p className="text-[11px] font-mono opacity-70 leading-relaxed">{readMode.message}</p>
                </div>
                <span className={`shrink-0 text-[9px] font-black font-mono uppercase tracking-widest px-2 py-0.5 rounded-full border
                    ${isStale ? "border-amber-500/30 text-amber-400" : "border-blue-500/30 text-blue-400"}`}>
                    {activeNode.replace("http://localhost:", "NODE:")}
                </span>
            </div>
        );
    };

    if (loading) return (
        <div className="flex flex-col items-center justify-center p-24 space-y-6">
            <div className="flex gap-2">
                {[...Array(3)].map((_, i) => (
                    <div key={i} className="w-1.5 h-1.5 rounded-full bg-blue-500 animate-bounce" style={{ animationDelay: `${i * 200}ms` }}></div>
                ))}
            </div>
            <span className="text-[10px] font-mono uppercase tracking-[0.5em] opacity-40 animate-pulse italic">Inquisiting Replicas</span>
        </div>
    );

    if (error) return (
        <div className="flex flex-col items-center justify-center p-16 space-y-4 border border-red-500/20 bg-red-500/5 rounded-3xl">
            <div className="w-12 h-12 rounded-full bg-red-500/10 flex items-center justify-center text-red-500">
                <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"></path></svg>
            </div>
            <p className="text-[10px] font-mono uppercase text-red-500 font-black tracking-widest">{error}</p>
        </div>
    );

    if (files.length === 0) {
        return (
            <>
                <DegradationBanner />
                <div className="flex flex-col items-center justify-center p-24 bg-white/1 rounded-3xl border border-dashed border-white/5 group transition-all hover:bg-white/2 cursor-default">
                    <div className="w-20 h-20 rounded-full bg-white/5 flex items-center justify-center mb-6 group-hover:scale-110 transition-transform">
                        <svg className="w-8 h-8 opacity-20 group-hover:opacity-40 transition-opacity" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4"></path></svg>
                    </div>
                    <span className="text-[10px] font-mono uppercase tracking-[0.5em] opacity-20 italic">No Distributed Assets Found</span>
                </div>
            </>
        );
    }

    return (
        <>
            <DegradationBanner />
            <ul className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 xl:grid-cols-4 gap-4 animate-in fade-in duration-500">
                {files.map((file, i) => {
                    const name = file.name || "Unknown File";
                    const isImage = /\.(jpg|jpeg|png|gif|webp)$/i.test(name);
                    const downloadUrl = `${activeNode}/download?file=${encodeURIComponent(name)}`;
                    const timestamp = file.modTime ? new Date(file.modTime).toLocaleString() : "Unknown Date";
                    const extension = name.split('.').pop()?.toUpperCase() || "FILE";
                    const isMetaOnly = !!file.meta_only;

                    return (
                        <li
                            key={`${file.name}-${i}`}
                            onClick={() => setPreviewFile(file)}
                            className="group relative flex flex-col justify-between p-4 overflow-visible cursor-pointer rounded-3xl border border-white/5 bg-white/2 hover:border-blue-500/50 hover:bg-white/5 transition-all duration-300"
                            style={{ animationDelay: `${i * 50}ms` }}
                        >
                            {/* Meta-only badge */}
                            {isMetaOnly && (
                                <div className="absolute top-3 left-3 px-2 py-0.5 bg-amber-500/10 border border-amber-500/30 rounded-full text-[8px] font-black font-mono uppercase tracking-widest text-amber-400 z-10">
                                    Metadata Only
                                </div>
                            )}

                            {/* Hover Metadata Popup */}
                            <div className="absolute left-1/2 -translate-x-1/2 bottom-full mb-2 w-56 bg-[#0a0a0a] border border-blue-500/30 rounded-xl p-4 shadow-2xl opacity-0 scale-95 group-hover:opacity-100 group-hover:scale-100 pointer-events-none transition-all duration-300 z-50 flex flex-col gap-2">
                                <div className="absolute -bottom-2 left-1/2 -translate-x-1/2 w-4 h-4 bg-[#0a0a0a] border-b border-r border-blue-500/30 transform rotate-45"></div>
                                <h4 className="text-white font-bold text-xs truncate border-b border-white/10 pb-2">{name}</h4>
                                <div className="text-[10px] text-gray-400 flex justify-between">
                                    <span>Type:</span> <span className="text-blue-400">{extension}</span>
                                </div>
                                <div className="text-[10px] text-gray-400 flex justify-between">
                                    <span>Size:</span> <span className="text-white">{formatBytes(file.size || 0)}</span>
                                </div>
                                <div className="text-[10px] text-gray-400 flex flex-col mt-1">
                                    <span>Last Synced:</span>
                                    <span className="text-white font-mono">{timestamp}</span>
                                </div>
                                <div className="text-[10px] text-gray-400 flex justify-between mt-1 pt-2 border-t border-white/10">
                                    <span>Storage:</span>
                                    <span className={isMetaOnly ? "text-amber-400" : "text-emerald-400"}>
                                        {isMetaOnly ? "Metadata Only" : "Available"}
                                    </span>
                                </div>
                            </div>

                            <div className="flex items-start justify-between mb-4 mt-4">
                                <div className="w-12 h-12 rounded-xl bg-white/5 border border-white/5 flex items-center justify-center overflow-hidden transition-all group-hover:scale-110 group-hover:bg-blue-500/10 group-hover:border-blue-500/20">
                                    {isImage && !isMetaOnly ? (
                                        <img src={downloadUrl} alt={name} className="w-full h-full object-cover" />
                                    ) : (
                                        <svg className="w-6 h-6 opacity-40 group-hover:opacity-100 group-hover:text-blue-500 transition-all" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"></path></svg>
                                    )}
                                </div>
                                <div className="flex flex-col items-end opacity-20 group-hover:opacity-40 transition-opacity text-[9px] font-mono font-bold uppercase tracking-widest leading-none text-right">
                                    <span>{activeNode.replace('http://localhost:', 'NODE:')}</span>
                                    <span className="mt-1">BLOCK-SYNC</span>
                                    <span className="mt-1 text-blue-500 font-bold">{extension}</span>
                                </div>
                            </div>

                            <div>
                                <div className="mb-4">
                                    <span className="block font-display font-black tracking-tight text-white/90 group-hover:text-white truncate pr-2 text-base">
                                        {name}
                                    </span>
                                    <div className="space-y-1 mt-2">
                                        <div className="flex gap-2 items-center opacity-40">
                                            <div className="w-1 h-1 rounded-full bg-blue-500"></div>
                                            <span className="text-[9px] font-mono tracking-widest font-bold uppercase">{formatBytes(file.size || 0)}</span>
                                        </div>
                                    </div>
                                </div>

                                {isMetaOnly ? (
                                    <div className="block text-center px-4 py-1.5 rounded-lg bg-amber-500/5 border border-amber-500/20 text-[9px] font-display font-black uppercase tracking-widest text-amber-500/60 cursor-not-allowed">
                                        Storage Unavailable
                                    </div>
                                ) : (
                                    <a
                                        href={downloadUrl}
                                        onClick={(e) => { e.stopPropagation(); }}
                                        className="block text-center px-4 py-1.5 overflow-hidden rounded-lg bg-white/5 border border-white/10 text-[9px] font-display font-black uppercase tracking-widest text-white/40 hover:text-white hover:bg-blue-500 hover:border-blue-500 transition-all shadow-xl shadow-black/20"
                                        download
                                    >
                                        Retrieve Asset
                                    </a>
                                )}
                            </div>

                            <div className="absolute top-0 right-0 w-24 h-24 bg-blue-500/5 blur-3xl opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none"></div>
                        </li>
                    );
                })}
            </ul>

            {/* Preview Dialog Modal */}
            {previewFile && (() => {
                const downloadUrl = `${activeNode}/download?file=${encodeURIComponent(previewFile.name)}`;
                const isImage = /\.(jpg|jpeg|png|gif|webp)$/i.test(previewFile.name);
                const isText = /\.(txt|json|md|csv|js|ts|tsx|css|html)$/i.test(previewFile.name);
                const isMetaOnly = !!previewFile.meta_only;

                return (
                    <div
                        className="fixed inset-0 z-[100] bg-black/80 backdrop-blur-md flex items-center justify-center p-4 sm:p-8 transition-opacity duration-300 animate-in fade-in"
                        onClick={() => setPreviewFile(null)}
                    >
                        <div
                            className="bg-[#050505] border border-white/10 rounded-2xl sm:rounded-3xl shadow-2xl relative max-w-5xl w-full max-h-[90vh] flex flex-col overflow-hidden animate-in zoom-in-95 duration-200"
                            onClick={e => e.stopPropagation()}
                        >
                            <div className="p-4 sm:p-6 border-b border-white/5 flex justify-between items-center bg-white/5">
                                <div className="flex items-center gap-3">
                                    <div className={`w-2 h-2 rounded-full animate-pulse ${isMetaOnly ? "bg-amber-500" : "bg-blue-500"}`}></div>
                                    <h3 className="text-sm sm:text-lg font-display font-bold truncate text-white">{previewFile.name}</h3>
                                    {isMetaOnly && (
                                        <span className="text-[9px] font-black font-mono uppercase tracking-widest px-2 py-0.5 rounded-full bg-amber-500/10 border border-amber-500/30 text-amber-400">
                                            Metadata Only
                                        </span>
                                    )}
                                </div>
                                <button
                                    onClick={() => setPreviewFile(null)}
                                    className="text-gray-400 hover:text-white hover:bg-white/10 p-2 rounded-xl transition-colors"
                                >
                                    <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                                    </svg>
                                </button>
                            </div>

                            <div className="flex-1 overflow-auto bg-black/20 flex flex-col justify-center items-center min-h-[300px] sm:min-h-[500px] p-4 sm:p-8">
                                {isMetaOnly ? (
                                    <div className="text-gray-500 flex flex-col items-center gap-6">
                                        <div className="w-24 h-24 rounded-full bg-amber-500/5 border border-amber-500/20 flex items-center justify-center">
                                            <svg className="w-10 h-10 text-amber-500/40" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" d="M3 15a4 4 0 004 4h9a5 5 0 10-.1-9.999 5.002 5.002 0 10-9.78 2.096A4.001 4.001 0 003 15z" />
                                            </svg>
                                        </div>
                                        <div className="text-center">
                                            <p className="font-bold text-amber-500/60 text-base uppercase tracking-widest">Storage Temporarily Unavailable</p>
                                            <p className="text-xs text-white/30 mt-2 max-w-xs leading-relaxed">
                                                MinIO is unreachable from this replica. Metadata is available but the file cannot be downloaded right now.
                                            </p>
                                        </div>
                                        <div className="grid grid-cols-2 gap-3 w-full max-w-xs text-[11px] font-mono">
                                            <div className="bg-white/3 border border-white/5 rounded-xl p-3">
                                                <p className="opacity-30 uppercase tracking-widest text-[9px] mb-1">Size</p>
                                                <p className="text-white font-bold">{formatBytes(previewFile.size || 0)}</p>
                                            </div>
                                            <div className="bg-white/3 border border-white/5 rounded-xl p-3">
                                                <p className="opacity-30 uppercase tracking-widest text-[9px] mb-1">Last Synced</p>
                                                <p className="text-white font-bold text-[9px]">{previewFile.modTime ? new Date(previewFile.modTime).toLocaleString() : "—"}</p>
                                            </div>
                                        </div>
                                    </div>
                                ) : isImage ? (
                                    <img src={downloadUrl} alt="Preview" className="max-w-full max-h-full object-contain drop-shadow-2xl rounded-lg" />
                                ) : isText ? (
                                    <iframe src={downloadUrl} className="w-full h-full bg-white/90 text-black rounded-lg" title="Text Preview" />
                                ) : (
                                    <div className="text-gray-500 flex flex-col items-center gap-6">
                                        <div className="w-24 h-24 rounded-full bg-white/5 flex items-center justify-center">
                                            <svg className="w-12 h-12 opacity-30" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"></path></svg>
                                        </div>
                                        <div className="text-center">
                                            <p className="font-bold text-white/50 text-lg uppercase tracking-widest">No Preview Available</p>
                                            <p className="text-xs text-white/30 mt-2">This file type must be downloaded to be viewed.</p>
                                        </div>
                                        <a href={downloadUrl} className="mt-4 px-8 py-3 rounded-xl bg-blue-600 hover:bg-blue-500 text-white font-bold text-xs uppercase tracking-widest transition-colors" download>
                                            Download File
                                        </a>
                                    </div>
                                )}
                            </div>

                            <div className="p-4 bg-[#050505] border-t border-white/5 flex flex-col sm:flex-row justify-between items-center gap-4">
                                <div className="text-[10px] font-mono text-white/40 flex items-center gap-4">
                                    <span>SIZE: <span className="text-white/80">{formatBytes(previewFile.size || 0)}</span></span>
                                    <span>DATA NODE: <span className="text-white/80">{activeNode}</span></span>
                                    {readMode && (
                                        <span>MODE: <span className={readMode.mode === "read-write" ? "text-emerald-400" : readMode.mode === "read-only-stale" ? "text-amber-400" : "text-blue-400"}>{readMode.mode.toUpperCase()}</span></span>
                                    )}
                                </div>
                                {!isMetaOnly && (
                                    <a
                                        href={downloadUrl}
                                        className="w-full sm:w-auto px-6 py-2 rounded-lg bg-white/10 hover:bg-white/20 text-white text-[10px] font-bold uppercase tracking-widest transition-colors text-center"
                                        download
                                    >
                                        Retrieve Asset directly
                                    </a>
                                )}
                            </div>
                        </div>
                    </div>
                );
            })()}
        </>
    );
}

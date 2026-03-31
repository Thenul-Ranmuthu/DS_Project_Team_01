"use client";

import { useEffect, useState } from "react";

const BACKEND_NODES = [
    "http://localhost:8000",
    "http://localhost:8001",
    "http://localhost:8002",
    "http://localhost:8003",
];

export default function FileList() {
    const [files, setFiles] = useState<string[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState("");
    const [activeNode, setActiveNode] = useState(BACKEND_NODES[0]);

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
                setFiles(data || []);
                setActiveNode(nodeUrl);
                success = true;
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
            <div className="flex flex-col items-center justify-center p-24 bg-white/1 rounded-3xl border border-dashed border-white/5 group transition-all hover:bg-white/2 cursor-default">
                <div className="w-20 h-20 rounded-full bg-white/5 flex items-center justify-center mb-6 group-hover:scale-110 transition-transform">
                    <svg className="w-8 h-8 opacity-20 group-hover:opacity-40 transition-opacity" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4"></path></svg>
                </div>
                <span className="text-[10px] font-mono uppercase tracking-[0.5em] opacity-20 italic">No Distributed Assets Found</span>
            </div>
        );
    }

    return (
        <ul className="grid grid-cols-1 md:grid-cols-2 gap-4 animate-in fade-in duration-500">
            {files.map((filename, i) => (
                <li
                    key={filename}
                    className="group relative h-40 flex flex-col justify-between p-6 overflow-hidden rounded-3xl border border-white/5 bg-white/2 hover:border-blue-500/50 hover:bg-white/5 transition-all duration-300"
                    style={{ animationDelay: `${i * 100}ms` }}
                >
                    <div className="flex items-start justify-between">
                        <div className="w-12 h-12 rounded-2xl bg-white/5 border border-white/5 flex items-center justify-center transition-all group-hover:scale-110 group-hover:bg-blue-500/10 group-hover:border-blue-500/20">
                            <svg className="w-6 h-6 opacity-40 group-hover:opacity-100 group-hover:text-blue-500 transition-all" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"></path></svg>
                        </div>
                        <div className="flex flex-col items-end opacity-20 group-hover:opacity-40 transition-opacity text-[10px] font-mono font-bold uppercase tracking-widest leading-none">
                            <span>{activeNode.replace('http://localhost:', 'NODE:')}</span>
                            <span className="mt-1">BLOCK-SYNC</span>
                        </div>
                    </div>

                    <div>
                        <div className="mb-4">
                            <span className="block font-display font-black tracking-tight text-white/90 group-hover:text-white truncate pr-4 text-lg">
                                {filename}
                            </span>
                            <div className="flex gap-2 items-center opacity-30 mt-1">
                                <div className="w-1 h-1 rounded-full bg-blue-500"></div>
                                <span className="text-[9px] font-mono tracking-widest font-black uppercase">Consensus Proven Asset</span>
                            </div>
                        </div>

                        <a
                            href={`${activeNode}/download?file=${encodeURIComponent(filename)}`}
                            className="absolute bottom-6 right-6 px-5 py-2 overflow-hidden rounded-xl bg-white/5 border border-white/10 text-[9px] font-display font-black uppercase tracking-widest text-white/40 hover:text-white hover:bg-blue-500 hover:border-blue-500 transition-all shadow-xl shadow-black/20"
                            download
                        >
                            Retrieve Asset
                        </a>
                    </div>

                    {/* Visual Hover Decoration */}
                    <div className="absolute top-0 right-0 w-24 h-24 bg-blue-500/5 blur-3xl opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none"></div>
                </li>
            ))}
        </ul>
    );
}

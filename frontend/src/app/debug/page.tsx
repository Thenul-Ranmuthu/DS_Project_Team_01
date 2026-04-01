"use client";

import { useEffect, useState } from "react";
import Link from "next/link";

const NODES = [
    { id: "NODE_01", url: "http://localhost:8000" },
    { id: "NODE_02", url: "http://localhost:8001" },
    { id: "NODE_03", url: "http://localhost:8002" },
    { id: "NODE_04", url: "http://localhost:8003" },
];

export default function DebugPage() {
    const [nodeStats, setNodeStats] = useState<any>({});
    const [loading, setLoading] = useState(true);

    const fetchStats = async () => {
        const stats: any = {};
        await Promise.all(
            NODES.map(async (node) => {
                const controller = new AbortController();
                const timeoutId = setTimeout(() => controller.abort(), 1000);
                try {
                    const res = await fetch(`${node.url}/status`, { signal: controller.signal });
                    clearTimeout(timeoutId);
                    if (res.ok) {
                        stats[node.id] = await res.json();
                    } else {
                        stats[node.id] = { state: "OFFLINE" };
                    }
                } catch (e) {
                    clearTimeout(timeoutId);
                    stats[node.id] = { state: "OFFLINE" };
                }
            })
        );
        setNodeStats(stats);
        setLoading(false);
    };

    useEffect(() => {
        fetchStats();
        const interval = setInterval(fetchStats, 2500);
        return () => clearInterval(interval);
    }, []);

    const shutdownNode = async (url: string) => {
        if (!confirm("Confirm remote shutdown and quorum reduction?")) return;
        try {
            await fetch(`${url}/shutdown`, { method: "POST" });
            fetchStats();
        } catch (e) {
            alert("Execution failed: Node already offline.");
        }
    };

    return (
        <main className="min-h-screen bg-[#050505] text-white overflow-hidden p-6 md:p-12 lg:p-16 selection:bg-blue-500/30">
            {/* Ambient background glow */}
            <div className="fixed -top-[20%] -left-[20%] w-[60%] h-[60%] bg-blue-900/10 blur-[120px] rounded-full opacity-50 pointer-events-none"></div>

            <div className="max-w-[1600px] mx-auto space-y-16 animate-in fade-in slide-in-from-bottom-2 duration-1000">
                <header className="flex flex-col lg:flex-row lg:items-end justify-between gap-12 border-b border-white/10 pb-12 relative">
                    <div className="space-y-4">
                        <Link href="/" className="group inline-flex items-center gap-2 text-[10px] font-mono font-black uppercase tracking-[0.3em] opacity-30 hover:opacity-100 hover:text-blue-500 transition-all">
                            <svg className="w-3 h-3 group-hover:-translate-x-1 transition-transform" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="3" d="M15 19l-7-7 7-7"></path></svg>
                            Back to Core
                        </Link>
                        <div className="space-y-1">
                            <h1 className="text-5xl lg:text-6xl font-display font-black tracking-tight flex items-end gap-2 leading-none uppercase italic">
                                CLUSTER<br /><span className="text-blue-500 italic">TELEMETRY</span>
                            </h1>
                        </div>
                    </div>

                    <div className="glass p-4 rounded-2xl flex items-center gap-6 border-white/5 shadow-2xl">
                        <div className="flex flex-col items-center">
                            <span className="text-[10px] font-mono opacity-20 uppercase font-black mb-1">Status</span>
                            <div className="flex gap-1.5 h-6 items-center">
                                {[...Array(6)].map((_, i) => (
                                    <div key={i} className="w-1 rounded-full bg-emerald-500/40 animate-pulse h-full" style={{ animationDelay: `${i * 100}ms` }}></div>
                                ))}
                            </div>
                        </div>
                        <div className="h-10 w-px bg-white/10"></div>
                        <div className="font-mono text-xs uppercase tracking-widest opacity-40">
                            SYS-INTEGRITY: NOMINAL<br />
                            ECC-CHECK-SUM: OK
                        </div>
                    </div>
                </header>

                <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-8">
                    {NODES.map((node, i) => {
                        const stats = nodeStats[node.id];
                        const isOffline = !stats || stats.state === "OFFLINE";
                        const isLeader = stats?.state === "Leader";

                        return (
                            <div
                                key={node.id}
                                className={`group relative glass rounded-3xl p-6 flex flex-col justify-between h-auto min-h-[550px] transition-all duration-300 ${isOffline
                                    ? 'opacity-30 grayscale border-red-500/10'
                                    : 'hover:border-blue-500/40 hover:-translate-y-2'
                                    }`}
                                style={{ animationDelay: `${i * 150}ms` }}
                            >
                                {/* Active State Background */}
                                {!isOffline && (
                                    <div className={`absolute top-0 right-0 w-32 h-32 blur-3xl rounded-full opacity-10 transition-opacity group-hover:opacity-30 pointer-events-none ${isLeader ? 'bg-blue-500' : 'bg-emerald-500'}`}></div>
                                )}

                                <div>
                                    <div className="flex justify-between items-start mb-10">
                                        <div className="space-y-1">
                                            <h3 className="text-2xl font-display font-black italic uppercase tracking-tighter opacity-80">{node.id}</h3>
                                            <p className="text-[10px] font-mono opacity-30 tracking-widest uppercase">Endpoint: {node.url.replace('http://', '').replace(':', ' : ')}</p>
                                        </div>
                                        {isLeader && (
                                            <div className="relative">
                                                <span className="absolute -inset-1 blur bg-blue-500/40 rounded-full animate-pulse"></span>
                                                <span className="relative bg-blue-500 text-white text-[10px] px-3 py-1 font-black uppercase rounded-full shadow-[0_0_20px_rgba(59,130,246,0.6)]">LEADER</span>
                                            </div>
                                        )}
                                        {isOffline && (
                                            <span className="bg-red-500/20 text-red-500 text-[10px] border border-red-500/30 px-3 py-1 font-black uppercase rounded-full">OFFLINE</span>
                                        )}
                                    </div>

                                    <div className="space-y-6">
                                        <div className="flex flex-col gap-1.5 font-mono text-[11px] uppercase font-black">
                                            <span className="opacity-20 flex items-center justify-between">State Transition <span className="h-px bg-white/5 flex-1 mx-3"></span></span>
                                            <span className={`text-base tracking-widest ${isOffline ? 'text-red-500' : isLeader ? 'text-blue-400' : 'text-emerald-400'}`}>
                                                {stats?.state || "DETECTING..."}
                                            </span>
                                        </div>

                                        {!isOffline && (
                                            <div className="grid grid-cols-2 gap-3">
                                                <div className="flex flex-col gap-1 bg-white/5 border border-white/5 rounded-2xl p-3 transition-all group-hover:bg-white/10">
                                                    <span className="text-[10px] font-mono opacity-30 uppercase font-black">App Indices</span>
                                                    <span className="text-lg font-display font-black tracking-tight">{stats.applied_index}</span>
                                                </div>
                                                <div className="flex flex-col gap-1 bg-white/5 border border-white/5 rounded-2xl p-3 transition-all group-hover:bg-white/10">
                                                    <span className="text-[10px] font-mono opacity-30 uppercase font-black">Commit Sync</span>
                                                    <span className="text-lg font-display font-black tracking-tight">{stats.commit_index}</span>
                                                </div>
                                                <div className="flex flex-col gap-1 bg-white/5 border border-white/5 rounded-2xl p-3 transition-all group-hover:bg-white/10">
                                                    <span className="text-[10px] font-mono opacity-30 uppercase font-black">Electoral Term</span>
                                                    <span className="text-lg font-display font-black tracking-tight">{stats.term}</span>
                                                </div>
                                                <div className="flex flex-col gap-1 bg-white/5 border border-white/5 rounded-2xl p-3 transition-all group-hover:bg-white/10">
                                                    <span className="text-[10px] font-mono opacity-30 uppercase font-black">Cluster Peers</span>
                                                    <span className="text-lg font-display font-black tracking-tight">{stats.num_peers}</span>
                                                </div>
                                            </div>
                                        )}

                                        {/* Activity Log */}
                                        {stats?.events && stats.events.length > 0 && (
                                            <div className="mt-6 pt-5 border-t border-white/5 space-y-3">
                                                <div className="flex items-center justify-between">
                                                    <span className="text-[11px] font-mono opacity-30 uppercase font-black">Activity Log</span>
                                                    <div className="flex gap-1">
                                                        <div className="w-1.5 h-1.5 rounded-full bg-blue-500 animate-pulse"></div>
                                                        <div className="w-1.5 h-1.5 rounded-full bg-blue-500 animate-pulse" style={{ animationDelay: '200ms' }}></div>
                                                    </div>
                                                </div>
                                                <div className="space-y-3 max-h-56 overflow-y-auto pr-1.5 custom-scrollbar">
                                                    {stats.events.slice().reverse().map((evt: string, idx: number) => (
                                                        <div key={idx} className="text-[12px] font-mono opacity-60 hover:opacity-100 transition-opacity border-l border-white/10 pl-3 py-1 leading-relaxed tracking-tight underline-offset-2 font-medium">
                                                            {evt}
                                                        </div>
                                                    ))}
                                                </div>
                                            </div>
                                        )}
                                    </div>
                                </div>

                                <div className="mt-8">
                                    {!isOffline ? (
                                        <button
                                            onClick={() => shutdownNode(node.url)}
                                            className="w-full h-11 rounded-2xl border border-red-500/20 hover:border-red-500 hover:bg-red-500 text-red-500 hover:text-white text-[11px] font-display font-black uppercase tracking-[0.3em] transition-all shadow-xl shadow-red-500/5 group/btn overflow-hidden relative"
                                        >
                                            <span className="relative z-10 transition-transform group-hover/btn:-translate-y-px">TERMINATE PROCESS</span>
                                            <div className="absolute inset-x-0 bottom-0 h-0.5 bg-black/20 origin-left scale-x-0 group-hover/btn:scale-x-100 transition-transform duration-700"></div>
                                        </button>
                                    ) : (
                                        <div className="text-center p-4 border border-dashed border-white/5 rounded-2xl flex flex-col gap-1 cursor-not-allowed">
                                            <span className="text-[8px] font-mono uppercase font-black opacity-10">Process Nullified</span>
                                            <span className="text-[7px] italic font-mono opacity-5">Awaiting physical restart...</span>
                                        </div>
                                    )}
                                </div>
                            </div>
                        );
                    })}
                </div>

                <section className="mt-32 max-w-4xl mx-auto glass p-10 rounded-[2.5rem] border-white/5 shadow-[0_64px_128px_-32px_rgba(0,0,0,1)] flex flex-col lg:flex-row gap-12 group overflow-hidden relative">
                    <div className="lg:w-12 h-12 shrink-0 rounded-full border border-blue-500/40 flex items-center justify-center text-blue-500 bg-blue-500/5 group-hover:scale-110 group-hover:rotate-12 transition-all">
                        <svg className="w-6 h-6" fill="currentColor" viewBox="0 0 20 20"><path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clipRule="evenodd"></path></svg>
                    </div>

                    <div className="space-y-6">
                        <div className="space-y-1">
                            <h2 className="text-2xl font-display font-black uppercase italic tracking-tight">Consensus Policy Control</h2>
                            <p className="text-[10px] font-mono opacity-20 uppercase tracking-[0.4em]">Rules for Manual State Overrides</p>
                        </div>
                        <div className="grid grid-cols-1 md:grid-cols-2 gap-8 text-[11px] font-mono leading-relaxed uppercase tracking-tighter opacity-40 italic">
                            <ul className="space-y-3 list-none">
                                <li className="flex items-start gap-4"><span className="text-blue-400 font-black">/01</span> Quorum (majority) is vital for availability. Terminating too many nodes locks consensus.</li>
                                <li className="flex items-start gap-4"><span className="text-blue-400 font-black">/02</span> A 4-node cluster requires {">"}2 nodes online to maintain commits.</li>
                            </ul>
                            <ul className="space-y-3 list-none">
                                <li className="flex items-start gap-4"><span className="text-blue-400 font-black">/03</span> Shutdown signals are absolute. Restarting requires local command execution.</li>
                                <li className="flex items-start gap-4"><span className="text-blue-400 font-black">/04</span> Telemetry reflects the state of the local State Machine and Raft log.</li>
                            </ul>
                        </div>
                    </div>

                    {/* Background decoration */}
                    <div className="absolute top-0 right-0 w-64 h-64 bg-blue-500/5 blur-[100px] rounded-full pointer-events-none group-hover:bg-blue-500/10 transition-colors"></div>
                </section>
            </div>
        </main>
    );
}

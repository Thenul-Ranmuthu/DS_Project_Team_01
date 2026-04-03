"use client";

import Link from "next/link";
import FileList from "./components/FileList";
import FileUpload from "./components/FileUpload";

export default function Home() {
  return (
    <main className="min-h-screen bg-[#050505] text-white selection:bg-blue-500/30">
      {/* Dynamic Background */}
      <div className="fixed inset-0 -z-10 bg-[radial-gradient(ellipse_at_top_right,_var(--tw-gradient-stops))] from-blue-900/10 via-black to-black opacity-60"></div>

      <div className="max-w-[1600px] mx-auto p-4 md:p-8 lg:p-12">
        {/* Header Unit */}
        <header className="flex flex-col md:flex-row md:items-center justify-between gap-6 mb-16 px-4">
          <div className="space-y-2">
            <h1 className="text-4xl md:text-5xl font-display font-black tracking-tight flex items-center gap-3 pr-4">
              <span className="bg-gradient-to-br from-white to-white/40 bg-clip-text text-transparent italic">DISTRIBUTED</span>
              <span className="text-blue-500 italic">SYSTEMS</span>
            </h1>
            <div className="flex items-center gap-3">
              <span className="h-px w-8 bg-blue-500/50"></span>
              <p className="text-[10px] md:text-xs font-mono font-bold tracking-[0.3em] uppercase opacity-50">
                RAFT CONSENSUS / METADATA SYNC / BLOCKET STORE
              </p>
            </div>
          </div>

          <nav className="flex items-center gap-4">
            <Link
              href="/debug"
              className="group relative px-6 py-2.5 overflow-hidden rounded-full font-bold uppercase tracking-widest text-[10px] bg-white/5 border border-white/10 glass-hover transition-all"
            >
              <span className="relative z-10 group-hover:text-blue-400">Debug Console</span>
              <div className="absolute inset-x-0 bottom-0 h-px bg-gradient-to-r from-transparent via-blue-500 to-transparent scale-x-0 group-hover:scale-x-100 transition-transform duration-500"></div>
            </Link>
          </nav>
        </header>

        {/* Dashboard Grid */}
        <div className="grid grid-cols-1 lg:grid-cols-12 gap-8 lg:gap-12">

          {/* Left Col: Actions */}
          <section className="lg:col-span-3 space-y-8 animate-in fade-in slide-in-from-left-4 duration-700">
            <div className="glass p-8 rounded-3xl relative overflow-hidden group">
              <div className="absolute -top-12 -right-12 w-32 h-32 bg-blue-500/10 rounded-full blur-3xl group-hover:bg-blue-500/20 transition-all"></div>

              <h2 className="text-xl font-display font-bold uppercase tracking-widest mb-8 flex items-center gap-3">
                <span className="w-1.5 h-1.5 rounded-full bg-blue-500 animate-pulse"></span>
                Input Terminal
              </h2>

              <FileUpload />
            </div>

            <div className="glass p-6 rounded-2xl border-white/5 bg-white/1 flex items-center justify-between group cursor-default">
              <div className="space-y-1">
                <p className="text-[10px] font-mono uppercase opacity-40 font-bold tracking-widest leading-none">Cluster Health</p>
                <p className="text-sm font-bold text-emerald-400 group-hover:text-emerald-300 transition-colors uppercase leading-none">Operational</p>
              </div>
              <div className="flex gap-1.5">
                {[...Array(4)].map((_, i) => (
                  <div key={i} className="w-1 h-3 rounded-full bg-emerald-500/30 animate-pulse" style={{ animationDelay: `${i * 150}ms` }}></div>
                ))}
              </div>
            </div>
          </section>

          {/* Right Col: Data */}
          <section className="lg:col-span-9 animate-in fade-in slide-in-from-bottom-4 duration-700 delay-150">
            <div className="glass p-1 rounded-3xl min-h-[600px] flex flex-col bg-white/2 shadow-[0_32px_64px_-16px_rgba(0,0,0,0.6)]">
              <div className="p-8 border-b border-white/10 flex items-center justify-between">
                <div>
                  <h2 className="text-xl font-display font-bold uppercase tracking-widest">Replicated Assets</h2>
                  <p className="text-[10px] font-mono opacity-20 uppercase tracking-[0.25em] mt-1">Virtual Path: /mnt/raft/storage</p>
                </div>
                <div className="h-8 w-8 rounded-full border border-white/10 bg-white/5 flex items-center justify-center">
                  <div className="w-1 h-1 rounded-full bg-white opacity-40"></div>
                </div>
              </div>

              <div className="flex-1 p-6 lg:p-10">
                <FileList />
              </div>
            </div>
          </section>

        </div>

        {/* System Footer */}
        <footer className="mt-32 pt-12 border-t border-white/5 text-[9px] font-mono opacity-20 uppercase tracking-[0.4em] flex flex-col md:flex-row gap-8 justify-between items-center text-center italic">
          <div className="flex items-center gap-4">
            <span>PROJECT: DS_TEAM_01</span>
            <span className="h-px w-4 bg-white/30"></span>
            <span>VER: 4.0.0-PRO</span>
          </div>
          <span>&copy; 2026 DISTRIBUTED SYSTEMS LABORATORY / END-TO-END REPLICATION</span>
          <div className="flex gap-12">
            <span>SECURE-BOOT: ENABLED</span>
            <span>ECC-CONSENSUS: STABLE</span>
          </div>
        </footer>
      </div>
    </main>
  );
}

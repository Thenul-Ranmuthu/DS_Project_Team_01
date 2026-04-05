"use client";

import { useState, useEffect } from "react";
import { createUser, getNodeStatuses, NodeStatus, NODES } from "./lib/api";
import Dashboard from "./components/Dashboard";

export default function Home() {
  const [step, setStep] = useState<"register" | "dashboard">("register");
  const [userEmail, setUserEmail] = useState("");
  const [userName, setUserName] = useState("");
  const [formEmail, setFormEmail] = useState("");
  const [formName, setFormName] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    const saved = localStorage.getItem("ds_user");
    if (saved) {
      const { email, name } = JSON.parse(saved);
      setUserEmail(email);
      setUserName(name);
      setStep("dashboard");
    }
  }, []);

  const handleRegister = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!formName.trim() || !formEmail.trim()) return;
    setLoading(true);
    setError("");
    try {
      await createUser(formName.trim(), formEmail.trim());
      localStorage.setItem("ds_user", JSON.stringify({ email: formEmail.trim(), name: formName.trim() }));
      setUserEmail(formEmail.trim());
      setUserName(formName.trim());
      setStep("dashboard");
    } catch (err: any) {
      setError(err.message || "Registration failed");
    } finally {
      setLoading(false);
    }
  };

  const handleLogout = () => {
    localStorage.removeItem("ds_user");
    setUserEmail(""); setUserName(""); setFormEmail(""); setFormName("");
    setStep("register");
  };

  if (step === "dashboard") {
    return <Dashboard email={userEmail} name={userName} onLogout={handleLogout} />;
  }

  return (
    <main className="min-h-screen bg-[#050505] text-white flex items-center justify-center relative overflow-hidden selection:bg-blue-500/30">
      <div className="fixed inset-0 -z-10">
        <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_center,_var(--tw-gradient-stops))] from-blue-950/20 via-transparent to-transparent"></div>
        <div className="absolute top-0 left-1/2 -translate-x-1/2 w-[800px] h-[800px] bg-blue-500/5 blur-[120px] rounded-full"></div>
        <div className="absolute inset-0 opacity-[0.03]" style={{ backgroundImage: "linear-gradient(rgba(255,255,255,.1) 1px, transparent 1px), linear-gradient(90deg, rgba(255,255,255,.1) 1px, transparent 1px)", backgroundSize: "60px 60px" }}></div>
      </div>

      <div className="w-full max-w-md mx-auto px-6 animate-in fade-in slide-in-from-bottom-4 duration-700">
        <div className="text-center mb-12">
          <div className="inline-flex items-center justify-center w-16 h-16 rounded-2xl bg-blue-500/10 border border-blue-500/20 mb-6 mx-auto">
            <svg className="w-8 h-8 text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" d="M5 12h14M5 12l4-4m-4 4l4 4M19 12l-4-4m4 4l-4 4" />
            </svg>
          </div>
          <h1 className="text-4xl font-display font-black tracking-tight uppercase italic mb-2">
            <span className="bg-gradient-to-br from-white to-white/40 bg-clip-text text-transparent">DIST</span>
            <span className="text-blue-500">STORE</span>
          </h1>
          <p className="text-[10px] font-mono tracking-[0.4em] uppercase opacity-30">ZooKeeper Leader Election / File Replication</p>
        </div>

        <div className="glass p-8 rounded-3xl relative overflow-hidden">
          <div className="absolute -top-16 -right-16 w-48 h-48 bg-blue-500/5 rounded-full blur-3xl pointer-events-none"></div>
          <h2 className="text-sm font-display font-black uppercase tracking-widest mb-2 flex items-center gap-3">
            <span className="w-1.5 h-1.5 rounded-full bg-blue-500 animate-pulse"></span>
            Register Node Access
          </h2>
          <p className="text-[10px] font-mono opacity-30 uppercase tracking-widest mb-8">Create an account to begin file operations</p>

          <form onSubmit={handleRegister} className="space-y-5">
            <div className="space-y-2">
              <label className="text-[10px] font-mono uppercase tracking-widest opacity-40 font-bold">Display Name</label>
              <input type="text" value={formName} onChange={(e) => setFormName(e.target.value)} placeholder="e.g. Thenul" required
                className="w-full bg-white/5 border border-white/10 rounded-xl px-4 py-3 text-sm font-mono focus:outline-none focus:border-blue-500/50 transition-all placeholder:opacity-20" />
            </div>
            <div className="space-y-2">
              <label className="text-[10px] font-mono uppercase tracking-widest opacity-40 font-bold">Email Address</label>
              <input type="email" value={formEmail} onChange={(e) => setFormEmail(e.target.value)} placeholder="e.g. user@example.com" required
                className="w-full bg-white/5 border border-white/10 rounded-xl px-4 py-3 text-sm font-mono focus:outline-none focus:border-blue-500/50 transition-all placeholder:opacity-20" />
            </div>
            {error && (
              <div className="flex items-center gap-3 p-3 rounded-xl bg-red-500/5 border border-red-500/20">
                <div className="w-1.5 h-1.5 rounded-full bg-red-500 shrink-0"></div>
                <p className="text-[10px] font-mono text-red-400 uppercase tracking-widest">{error}</p>
              </div>
            )}
            <button type="submit" disabled={loading}
              className="w-full h-12 mt-2 bg-gradient-to-br from-white to-white/70 hover:from-blue-500 hover:to-blue-600 text-black hover:text-white font-display font-black uppercase tracking-[0.2em] text-sm transition-all rounded-2xl active:scale-[0.98] disabled:opacity-30 shadow-xl shadow-black/20">
              {loading ? (
                <span className="flex items-center justify-center gap-2">
                  {[0,150,300].map(d => <span key={d} className="w-1.5 h-1.5 rounded-full bg-current animate-bounce" style={{animationDelay:`${d}ms`}}></span>)}
                </span>
              ) : "Initialize Session"}
            </button>
          </form>
          <div className="mt-6 pt-6 border-t border-white/5">
            <p className="text-[9px] font-mono uppercase tracking-widest opacity-20 text-center italic">Registration is routed to the elected leader node automatically</p>
          </div>
        </div>
        <NodePing />
      </div>
    </main>
  );
}

function NodePing() {
  const [statuses, setStatuses] = useState<NodeStatus[]>([]);
  useEffect(() => {
    const run = () => getNodeStatuses().then(setStatuses);
    run();
    const i = setInterval(run, 5000);
    return () => clearInterval(i);
  }, []);

  return (
    <div className="mt-6 glass p-4 rounded-2xl flex items-center justify-between">
      <span className="text-[9px] font-mono uppercase tracking-widest opacity-30">Cluster Nodes</span>
      <div className="flex gap-3 items-center">
        {NODES.map((node) => {
          const s = statuses.find((x) => x.id === node.id);
          return (
            <div key={node.id} className="flex flex-col items-center gap-1">
              <div className={`w-2 h-2 rounded-full ${!s?.online ? "bg-red-500/40" : s.is_leader ? "bg-blue-500 animate-pulse shadow-[0_0_8px_rgba(59,130,246,0.8)]" : "bg-emerald-500/60"}`}></div>
              <span className="text-[8px] font-mono opacity-30">{node.port}</span>
            </div>
          );
        })}
      </div>
    </div>
  );
}

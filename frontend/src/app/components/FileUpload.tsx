"use client";

import { useState } from "react";

const BACKEND_NODES = [
    "http://localhost:8000",
    "http://localhost:8001",
    "http://localhost:8002",
    "http://localhost:8003",
    "http://localhost:8004",
    "http://localhost:8005",
    "http://localhost:8006",
];

export default function FileUpload() {
    const [file, setFile] = useState<File | null>(null);
    const [uploading, setUploading] = useState(false);
    const [message, setMessage] = useState("");
    const [dragging, setDragging] = useState(false);

    const findLeaderAndUpload = async (fileToUpload: File) => {
        setUploading(true);
        setMessage("Detecting leader...");

        for (const nodeUrl of BACKEND_NODES) {
            const controller = new AbortController();
            const timeoutId = setTimeout(() => controller.abort(), 1000);
            try {
                const statusRes = await fetch(`${nodeUrl}/status`, { signal: controller.signal });
                clearTimeout(timeoutId);
                if (!statusRes.ok) continue;
                const status = await statusRes.json();
                if (status.state === "Leader") {
                    return await uploadToNode(nodeUrl, fileToUpload);
                }
            } catch (err) {
                clearTimeout(timeoutId);
                continue;
            }
        }
        return await uploadToNode(BACKEND_NODES[0], fileToUpload);
    };

    const uploadToNode = async (nodeUrl: string, fileToUpload: File) => {
        setMessage(`Broadcasting to ${nodeUrl.replace('http://localhost:', 'node:')}...`);
        const formData = new FormData();
        formData.append("file", fileToUpload);

        try {
            const res = await fetch(`${nodeUrl}/upload`, { method: "POST", body: formData });
            if (res.ok) {
                setMessage("Sync complete.");
                setFile(null);
                setTimeout(() => window.location.reload(), 1500);
            } else if (res.status === 423) {
                const data = await res.json();
                if (data.leader) {
                    const port = parseInt(data.leader.split(":")[1]) - 1000;
                    return await uploadToNode(`http://localhost:${port}`, fileToUpload);
                }
            } else {
                setMessage(`Upload aborted.`);
            }
        } catch (err: any) {
            setMessage(`Node error.`);
        } finally {
            setUploading(false);
        }
    };

    const handleUpload = async (e: React.FormEvent) => {
        e.preventDefault();
        if (!file) return;
        await findLeaderAndUpload(file);
    };

    return (
        <form onSubmit={handleUpload} className="space-y-8">
            <div className="relative group">
                <input id="dropzone-file" type="file" className="hidden" onChange={(e) => setFile(e.target.files?.[0] || null)} />
                <label
                    htmlFor="dropzone-file"
                    className={`flex flex-col items-center justify-center w-full h-56 rounded-3xl border border-dashed cursor-pointer transition-all duration-500 overflow-hidden relative ${dragging
                            ? "border-blue-500 bg-blue-500/10 shadow-[0_0_80px_-20px_rgba(59,130,246,0.3)]"
                            : "border-white/10 bg-white/5 hover:border-white/30 hover:bg-white/10"
                        }`}
                    onDragOver={(e) => { e.preventDefault(); setDragging(true); }}
                    onDragLeave={() => setDragging(false)}
                    onDrop={(e) => { e.preventDefault(); setDragging(false); if (e.dataTransfer.files?.[0]) setFile(e.dataTransfer.files[0]); }}
                >
                    <div className="flex flex-col items-center justify-center p-8 text-center space-y-4">
                        <div className={`w-16 h-16 rounded-full flex items-center justify-center transition-all duration-500 ${dragging ? 'bg-blue-500 scale-110' : 'bg-white/5 group-hover:bg-white/10'}`}>
                            <svg className={`w-8 h-8 transition-colors ${dragging ? 'text-white' : 'text-white/40 group-hover:text-white'}`} fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12"></path></svg>
                        </div>
                        <div className="space-y-1">
                            <p className="font-display font-black text-sm uppercase tracking-widest text-white/90">
                                {file ? file.name : "Inject Fragment"}
                            </p>
                            <p className="text-[10px] uppercase font-mono tracking-widest opacity-30 italic">Drag fragments onto surface</p>
                        </div>
                    </div>
                </label>
                {file && (
                    <div className="absolute top-2 right-2 px-3 py-1 bg-emerald-500 text-black text-[8px] font-black uppercase rounded-full animate-pulse shadow-[0_0_20px_rgba(16,185,129,0.3)]">
                        READY
                    </div>
                )}
            </div>

            <div className="space-y-4">
                <button
                    type="submit"
                    disabled={!file || uploading}
                    className="w-full group h-14 bg-gradient-to-br from-white to-white/70 hover:from-blue-500 hover:to-blue-600 text-black hover:text-white font-display font-black uppercase tracking-[0.3em] transition-all rounded-2xl active:scale-[0.98] disabled:opacity-5 disabled:grayscale shadow-xl shadow-black/20"
                >
                    {uploading ? (
                        <span className="flex items-center justify-center gap-3">
                            <span className="w-1.5 h-1.5 rounded-full bg-current animate-bounce" style={{ animationDelay: '0ms' }}></span>
                            <span className="w-1.5 h-1.5 rounded-full bg-current animate-bounce" style={{ animationDelay: '200ms' }}></span>
                            <span className="w-1.5 h-1.5 rounded-full bg-current animate-bounce" style={{ animationDelay: '400ms' }}></span>
                        </span>
                    ) : "Commit Fragment"}
                </button>
                <div className="flex items-center gap-3 px-2">
                    <div className={`h-1 w-1 rounded-full ${message.includes('❌') ? 'bg-red-500' : 'bg-blue-500'} animate-pulse`}></div>
                    <p className={`text-[9px] font-mono uppercase font-bold tracking-[0.15em] ${message.includes('❌') ? 'text-red-500' : 'text-blue-400 opacity-60'}`}>
                        {message || "Terminal standby."}
                    </p>
                </div>
            </div>
        </form>
    );
}

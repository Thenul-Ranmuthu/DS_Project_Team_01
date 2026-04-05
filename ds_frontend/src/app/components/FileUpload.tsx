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

const ALLOWED_TYPES = ".png,.jpg,.jpeg,.gif,.svg,.mp3,.mp4";

export default function FileUpload() {
    const [files, setFiles] = useState<File[]>([]);
    const [uploading, setUploading] = useState(false);
    const [message, setMessage] = useState("");
    const [dragging, setDragging] = useState(false);

    const findLeaderAndUpload = async (fileToUpload: File, idempotencyKey: string) => {
        for (const nodeUrl of BACKEND_NODES) {
            const controller = new AbortController();
            const timeoutId = setTimeout(() => controller.abort(), 1000);
            try {
                const statusRes = await fetch(`${nodeUrl}/status`, { signal: controller.signal });
                clearTimeout(timeoutId);
                if (!statusRes.ok) continue;
                const status = await statusRes.json();
                if (status.state === "Leader") {
                    return await uploadToNode(nodeUrl, fileToUpload, idempotencyKey);
                }
            } catch (err) {
                clearTimeout(timeoutId);
                continue;
            }
        }
        return await uploadToNode(BACKEND_NODES[0], fileToUpload, idempotencyKey);
    };

    const uploadToNode = async (nodeUrl: string, fileToUpload: File, idempotencyKey: string, attempt = 1): Promise<boolean> => {
        setMessage(`Broadcasting ${fileToUpload.name} (Attempt ${attempt})...`);
        const formData = new FormData();
        formData.append("file", fileToUpload);

        try {
            const res = await fetch(`${nodeUrl}/upload`, { 
                method: "POST", 
                body: formData,
                headers: {
                    "Idempotency-Key": idempotencyKey
                }
            });
            if (res.ok) {
                return true;
            } else if (res.status === 423) {
                const data = await res.json();
                if (data.leader) {
                    const port = parseInt(data.leader.split(":")[1]);
                    return await uploadToNode(`http://localhost:${port}`, fileToUpload, idempotencyKey, attempt);
                }
            } else if (res.status >= 500) {
                 throw new Error(`Server Error ${res.status}`);
            }
            return false;
        } catch (err: any) {
            if (attempt <= 5) {
                const backoffMs = Math.pow(2, attempt) * 500 + Math.random() * 200;
                setMessage(`Upload failed. Retrying in ${Math.round(backoffMs)}ms...`);
                await new Promise(resolve => setTimeout(resolve, backoffMs));
                return await uploadToNode(nodeUrl, fileToUpload, idempotencyKey, attempt + 1);
            }
            return false;
        }
    };

    const handleUpload = async (e: React.FormEvent) => {
        e.preventDefault();
        if (files.length === 0) return;
        
        setUploading(true);
        let successCount = 0;

        for (let i = 0; i < files.length; i++) {
            setMessage(`Uploading fragment ${i + 1}/${files.length}...`);
            const idempotencyKey = crypto.randomUUID ? crypto.randomUUID() : Math.random().toString(36).substring(2) + Date.now().toString(36);
            const success = await findLeaderAndUpload(files[i], idempotencyKey);
            if (success) successCount++;
        }

        if (successCount === files.length) {
            setMessage(`Sync complete: ${successCount} fragments committed.`);
            setFiles([]);
            setTimeout(() => window.location.reload(), 1500);
        } else {
            setMessage(`Partial sync: ${successCount}/${files.length} fragments committed.`);
        }
        setUploading(false);
    };

    const handleFileChange = (newFiles: FileList | null) => {
        if (!newFiles) return;
        const fileArray = Array.from(newFiles);
        setFiles(prev => [...prev, ...fileArray]);
    };

    return (
        <form onSubmit={handleUpload} className="space-y-8">
            <div className="relative group">
                <input 
                    id="dropzone-file" 
                    type="file" 
                    multiple 
                    accept={ALLOWED_TYPES}
                    className="hidden" 
                    onChange={(e) => handleFileChange(e.target.files)} 
                />
                <label
                    htmlFor="dropzone-file"
                    className={`flex flex-col items-center justify-center w-full h-56 rounded-3xl border border-dashed cursor-pointer transition-all duration-500 overflow-hidden relative ${dragging
                            ? "border-blue-500 bg-blue-500/10 shadow-[0_0_80px_-20px_rgba(59,130,246,0.3)]"
                            : "border-white/10 bg-white/5 hover:border-white/30 hover:bg-white/10"
                        }`}
                    onDragOver={(e) => { e.preventDefault(); setDragging(true); }}
                    onDragLeave={() => setDragging(false)}
                    onDrop={(e) => { e.preventDefault(); setDragging(false); handleFileChange(e.dataTransfer.files); }}
                >
                    <div className="flex flex-col items-center justify-center p-8 text-center space-y-4">
                        <div className={`w-16 h-16 rounded-full flex items-center justify-center transition-all duration-500 ${dragging ? 'bg-blue-500 scale-110' : 'bg-white/5 group-hover:bg-white/10'}`}>
                            <svg className={`w-8 h-8 transition-colors ${dragging ? 'text-white' : 'text-white/40 group-hover:text-white'}`} fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12"></path></svg>
                        </div>
                        <div className="space-y-1">
                            <p className="font-display font-black text-sm uppercase tracking-widest text-white/90">
                                {files.length > 0 ? `${files.length} fragments selected` : "Inject Fragments"}
                            </p>
                            <p className="text-[10px] uppercase font-mono tracking-widest opacity-30 italic">
                                {ALLOWED_TYPES.replace(/\./g, '').toUpperCase().split(',').join(' / ')}
                            </p>
                        </div>
                    </div>
                </label>
                {files.length > 0 && (
                    <div className="absolute top-2 right-2 px-3 py-1 bg-emerald-500 text-black text-[8px] font-black uppercase rounded-full animate-pulse shadow-[0_0_20px_rgba(16,185,129,0.3)]">
                        {files.length} READY
                    </div>
                )}
            </div>

            <div className="space-y-4">
                <button
                    type="submit"
                    disabled={files.length === 0 || uploading}
                    className="w-full group h-14 bg-gradient-to-br from-white to-white/70 hover:from-blue-500 hover:to-blue-600 text-black hover:text-white font-display font-black uppercase tracking-[0.3em] transition-all rounded-2xl active:scale-[0.98] disabled:opacity-5 disabled:grayscale shadow-xl shadow-black/20"
                >
                    {uploading ? (
                        <span className="flex items-center justify-center gap-3">
                            <span className="w-1.5 h-1.5 rounded-full bg-current animate-bounce" style={{ animationDelay: '0ms' }}></span>
                            <span className="w-1.5 h-1.5 rounded-full bg-current animate-bounce" style={{ animationDelay: '200ms' }}></span>
                            <span className="w-1.5 h-1.5 rounded-full bg-current animate-bounce" style={{ animationDelay: '400ms' }}></span>
                        </span>
                    ) : "Commit Fragments"}
                </button>
                <div className="flex items-center gap-3 px-2">
                    <div className={`h-1 w-1 rounded-full ${message.includes('❌') || message.includes('Partial') ? 'bg-red-500' : 'bg-blue-500'} animate-pulse`}></div>
                    <p className={`text-[9px] font-mono uppercase font-bold tracking-[0.15em] ${message.includes('❌') || message.includes('Partial') ? 'text-red-500' : 'text-blue-400 opacity-60'}`}>
                        {message || "Terminal standby."}
                    </p>
                </div>
            </div>
        </form>
    );
}

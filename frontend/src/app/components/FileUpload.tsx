"use client";

import { useState } from "react";

const BACKEND_NODES = [
    "http://localhost:8000",
    "http://localhost:8001",
    "http://localhost:8002",
    "http://localhost:8003",
];

export default function FileUpload() {
    const [file, setFile] = useState<File | null>(null);
    const [uploading, setUploading] = useState(false);
    const [message, setMessage] = useState("");
    const [dragging, setDragging] = useState(false);

    const findLeaderAndUpload = async (fileToUpload: File) => {
        setUploading(true);
        setMessage("🔍 Finding cluster leader...");

        for (const nodeUrl of BACKEND_NODES) {
            try {
                console.log(`Checking leader status at ${nodeUrl}...`);
                const statusRes = await fetch(`${nodeUrl}/status`, { signal: AbortSignal.timeout(1000) });
                if (!statusRes.ok) continue;

                const status = await statusRes.json();
                const targetUrl = status.state === "Leader" ? nodeUrl : null;

                if (targetUrl) {
                    console.log(`Leader found at ${targetUrl}. Starting upload...`);
                    return await uploadToNode(targetUrl, fileToUpload);
                }
            } catch (err) {
                console.warn(`Node ${nodeUrl} is unreachable.`);
            }
        }

        // fallback: if no node reports being leader, try 423 redirect trick on node 0
        return await uploadToNode(BACKEND_NODES[0], fileToUpload);
    };

    const uploadToNode = async (nodeUrl: string, fileToUpload: File) => {
        setMessage(`📤 Uploading to ${nodeUrl}...`);
        const formData = new FormData();
        formData.append("file", fileToUpload);

        try {
            const res = await fetch(`${nodeUrl}/upload`, {
                method: "POST",
                body: formData,
            });

            if (res.ok) {
                setMessage("✅ File uploaded successfully!");
                setFile(null);
                setTimeout(() => window.location.reload(), 1500);
            } else if (res.status === 423) {
                // Our custom leader hint
                const data = await res.json();
                if (data.leader) {
                    // The leader address from Raft is 127.0.0.1:900x. We need http port 800x.
                    const port = parseInt(data.leader.split(":")[1]) - 1000;
                    const leaderHttp = `http://localhost:${port}`;
                    setMessage(`↪️ Redirecting to leader at ${leaderHttp}...`);
                    return await uploadToNode(leaderHttp, fileToUpload);
                }
                throw new Error("Cluster is still electing a leader.");
            } else {
                const text = await res.text();
                setMessage(`❌ Upload failed: ${text}`);
            }
        } catch (err: any) {
            setMessage(`❌ Error: ${err.message}`);
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
        <form onSubmit={handleUpload} className="space-y-6">
            <div
                className="flex flex-col items-center justify-center w-full"
                onDragOver={(e) => { e.preventDefault(); setDragging(true); }}
                onDragLeave={() => setDragging(false)}
                onDrop={(e) => { e.preventDefault(); setDragging(false); if (e.dataTransfer.files?.[0]) setFile(e.dataTransfer.files[0]); }}
            >
                <label
                    htmlFor="dropzone-file"
                    className={`flex flex-col items-center justify-center w-full h-56 border-2 border-dashed rounded-xl cursor-pointer transition-all duration-300 ${dragging ? "border-blue-400 bg-blue-900/20" : "border-slate-600 bg-slate-900/50 hover:bg-slate-800"
                        }`}
                >
                    <div className="flex flex-col items-center justify-center pt-5 pb-6 text-slate-400 relative">
                        <div className={`p-4 rounded-full mb-4 transition-all duration-500 ${dragging ? 'bg-blue-500/20 scale-110' : 'bg-slate-800'}`}>
                            <svg className={`w-10 h-10 ${dragging ? 'text-blue-400' : 'text-slate-400'}`} fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12"></path></svg>
                        </div>
                        <p className="mb-2 text-base">
                            <span className="font-semibold text-blue-400">Click to upload</span> or drag and drop
                        </p>
                        {file && <span className="text-emerald-400 font-medium">Selected: {file.name}</span>}
                    </div>
                    <input id="dropzone-file" type="file" className="hidden" onChange={(e) => setFile(e.target.files?.[0] || null)} />
                </label>
            </div>

            <div className="flex items-center justify-between pt-8">
                <span className={`text-sm font-medium ${message.includes('❌') ? 'text-red-400' : 'text-emerald-400'}`}>
                    {message}
                </span>
                <button
                    type="submit"
                    disabled={!file || uploading}
                    className="px-8 py-3 bg-gradient-to-r from-blue-600 to-indigo-600 hover:from-blue-500 hover:to-indigo-500 text-white font-semibold rounded-lg disabled:opacity-50 transition-all shadow-lg shadow-blue-500/25 flex items-center gap-2"
                >
                    {uploading ? "Processing..." : "Upload File"}
                </button>
            </div>
        </form>
    );
}

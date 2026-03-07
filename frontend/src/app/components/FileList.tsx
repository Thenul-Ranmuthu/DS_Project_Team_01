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
            try {
                const res = await fetch(`${nodeUrl}/files`, { signal: AbortSignal.timeout(1000) });
                if (!res.ok) continue;
                const data = await res.json();
                setFiles(data || []);
                setActiveNode(nodeUrl);
                success = true;
                break;
            } catch (err) {
                continue;
            }
        }

        if (!success) {
            setError("Could not connect to any nodes in the Raft cluster.");
        }
        setLoading(false);
    };

    useEffect(() => {
        fetchFiles();
    }, []);

    if (loading) return <div className="text-center py-8">Loading cluster files...</div>;
    if (error) return <div className="text-red-400 p-4 bg-red-900/10 rounded-xl border border-red-500/20">{error}</div>;

    return (
        <ul className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {files.map((filename) => (
                <li
                    key={filename}
                    className="flex items-center justify-between p-4 bg-slate-800/80 rounded-xl border border-slate-600 hover:border-blue-500/60 transition-all group"
                >
                    <span className="truncate font-medium text-slate-200">{filename}</span>
                    <a
                        href={`${activeNode}/download?file=${encodeURIComponent(filename)}`}
                        className="px-4 py-2 bg-slate-600 hover:bg-emerald-600 text-white text-sm font-medium rounded-lg"
                        download
                    >
                        Download
                    </a>
                </li>
            ))}
        </ul>
    );
}

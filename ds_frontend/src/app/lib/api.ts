export const NODES = [
  { id: "node-5051", url: "http://localhost:5051", port: 5051 },
  { id: "node-5052", url: "http://localhost:5052", port: 5052 },
  { id: "node-5053", url: "http://localhost:5053", port: 5053 },
  { id: "node-5054", url: "http://localhost:5054", port: 5054 },
];

export interface ElectionStatus {
  is_leader: boolean;
  leader_id: string;
  node_id: string;
  znode: string;
}

export interface NodeStatus extends ElectionStatus {
  url: string;
  id: string;
  port: number;
  online: boolean;
  lamport_clock?: number;
}

export interface UserFile {
  ID: number;
  CreatedAt: string;
  UpdatedAt: string;
  DeletedAt: null | string;
  original_name: string;
  file_path: string;
  mime_type: string;
  file_size: number;
  user_id: number;
  user: object;
}

export interface User {
  ID: number;
  name: string;
  email: string;
  files: UserFile[] | null;
}

async function fetchWithTimeout(url: string, options?: RequestInit, timeoutMs = 2000): Promise<Response> {
  const controller = new AbortController();
  const id = setTimeout(() => controller.abort(), timeoutMs);
  try {
    const res = await fetch(url, { ...options, signal: controller.signal });
    clearTimeout(id);
    return res;
  } catch (e) {
    clearTimeout(id);
    throw e;
  }
}

export async function getNodeStatuses(): Promise<NodeStatus[]> {
  return Promise.all(
    NODES.map(async (node) => {
      try {
        const res = await fetchWithTimeout(`${node.url}/election/status`);
        if (!res.ok) return { ...node, online: false, is_leader: false, leader_id: "", node_id: node.id, znode: "" };
        const data: ElectionStatus = await res.json();
        return { ...node, ...data, online: true };
      } catch {
        return { ...node, online: false, is_leader: false, leader_id: "", node_id: node.id, znode: "" };
      }
    })
  );
}

export async function findLeaderUrl(): Promise<string | null> {
  const statuses = await getNodeStatuses();
  const leader = statuses.find((s) => s.is_leader && s.online);
  return leader ? leader.url : null;
}

export async function getLamportClock(nodeUrl: string): Promise<number | null> {
  try {
    const res = await fetchWithTimeout(`${nodeUrl}/clock`);
    if (!res.ok) return null;
    const data = await res.json();
    return data.lamport_clock ?? data.clock ?? null;
  } catch {
    return null;
  }
}

export async function createUser(name: string, email: string): Promise<User> {
  const leaderUrl = await findLeaderUrl();
  const url = leaderUrl ?? NODES[0].url;
  const res = await fetch(`${url}/createUser`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ name, email }),
  });
  if (!res.ok) {
    const err = await res.text();
    throw new Error(err || "Failed to create user");
  }
  return res.json();
}

export async function getUserFiles(email: string): Promise<UserFile[]> {
  for (const node of NODES) {
    try {
      const res = await fetchWithTimeout(`${node.url}/users/files/${encodeURIComponent(email)}`);
      if (!res.ok) continue;
      const data = await res.json();
      return data.files ?? data ?? [];
    } catch {
      continue;
    }
  }
  return [];
}

export async function uploadFiles(email: string, files: File[]): Promise<{ lamport_clock: number; message: string }> {
  const leaderUrl = await findLeaderUrl();
  if (!leaderUrl) throw new Error("No leader found in cluster");
  const formData = new FormData();
  files.forEach((f) => formData.append("files", f));
  const res = await fetch(`${leaderUrl}/upload/${encodeURIComponent(email)}`, {
    method: "POST",
    body: formData,
  });
  if (!res.ok) {
    const err = await res.text();
    throw new Error(err || "Upload failed");
  }
  return res.json();
}

export async function deleteFile(fileId: number): Promise<void> {
  const leaderUrl = await findLeaderUrl();
  if (!leaderUrl) throw new Error("No leader found in cluster");
  const res = await fetch(`${leaderUrl}/files/${fileId}`, { method: "DELETE" });
  if (!res.ok) throw new Error("Delete failed");
}

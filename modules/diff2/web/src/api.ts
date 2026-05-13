// Tiny fetch wrapper for the diff2 JSON API.
// Throws APIError on non-2xx responses so callers can unwrap.
import type {
  BeadRequest,
  BeadResponse,
  File,
  FileDiff,
  Health,
} from "./types";

export class APIError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
    this.name = "APIError";
  }
}

async function j<T>(res: Response): Promise<T> {
  const body = await res.text();
  let parsed: unknown = null;
  try {
    parsed = body ? JSON.parse(body) : null;
  } catch {
    throw new APIError(res.status, body || res.statusText);
  }
  if (!res.ok) {
    const msg =
      (parsed as { error?: string })?.error ??
      res.statusText ??
      `HTTP ${res.status}`;
    throw new APIError(res.status, msg);
  }
  return parsed as T;
}

export async function getHealth(): Promise<Health> {
  return j<Health>(await fetch("/api/health"));
}

export async function listDiff(): Promise<File[]> {
  const res = await j<{ files: File[] }>(await fetch("/api/diff"));
  return res.files;
}

export async function getFileDiff(path: string): Promise<FileDiff> {
  const url = `/api/diff/file?path=${encodeURIComponent(path)}`;
  return j<FileDiff>(await fetch(url));
}

export async function listLabels(): Promise<string[]> {
  const res = await j<{ labels: string[] }>(await fetch("/api/labels"));
  return res.labels;
}

export async function createBead(req: BeadRequest): Promise<BeadResponse> {
  const res = await fetch("/api/bead", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(req),
  });
  return j<BeadResponse>(res);
}

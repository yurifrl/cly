// Mirror of the Go JSON shapes in modules/diff2/*.go.
// Keep in sync by hand for now — no codegen yet.

export type FileStatus =
  | "added"
  | "modified"
  | "deleted"
  | "renamed"
  | "copied"
  | "untracked"
  | "unknown";

export interface File {
  path: string;
  oldPath?: string;
  status: FileStatus;
  additions: number;
  deletions: number;
  binary: boolean;
}

export type LineKind = "context" | "add" | "del" | "meta";

export interface Line {
  kind: LineKind;
  old: number;
  new: number;
  text: string;
}

export interface Hunk {
  header: string;
  lines: Line[];
}

export interface FileDiff {
  path: string;
  binary: boolean;
  hunks: Hunk[];
}

export interface Health {
  git: boolean;
  bd: boolean;
  beadsDb: boolean;
}

export type BeadType = "bug" | "feature" | "task" | "chore" | "decision";
export type BeadPriority = "P0" | "P1" | "P2" | "P3" | "P4";

export interface BeadRequest {
  title: string;
  description?: string;
  type: BeadType;
  priority: BeadPriority;
  context?: string;
  labels?: string[];
}

export interface BeadResponse {
  id: string;
}

export interface APIError {
  error: string;
}

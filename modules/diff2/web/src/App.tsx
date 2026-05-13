import { useCallback, useEffect, useState } from "react";
import * as api from "./api";
import type { File, FileDiff, Health } from "./types";
import { BeadModal } from "./components/BeadModal";

export function App() {
  const [health, setHealth] = useState<Health | null>(null);
  const [files, setFiles] = useState<File[]>([]);
  const [idx, setIdx] = useState(0);
  const [diff, setDiff] = useState<FileDiff | null>(null);
  const [modalOpen, setModalOpen] = useState(false);
  const [toast, setToast] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    void (async () => {
      try {
        const [h, f] = await Promise.all([api.getHealth(), api.listDiff()]);
        setHealth(h);
        setFiles(f);
      } catch (e) {
        setError((e as Error).message);
      }
    })();
  }, []);

  const current = files[idx];

  useEffect(() => {
    if (!current) {
      setDiff(null);
      return;
    }
    void (async () => {
      try {
        setDiff(await api.getFileDiff(current.path));
      } catch (e) {
        setError((e as Error).message);
      }
    })();
  }, [current?.path]);

  const showToast = useCallback((msg: string) => {
    setToast(msg);
    window.setTimeout(() => setToast(null), 2200);
  }, []);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (modalOpen) return;
      const t = e.target as HTMLElement;
      if (t.tagName === "INPUT" || t.tagName === "TEXTAREA") return;
      if (e.key === "n" || e.key === "N") {
        e.preventDefault();
        setModalOpen(true);
      } else if (e.key === "ArrowLeft") {
        setIdx((i) => Math.max(0, i - 1));
      } else if (e.key === "ArrowRight") {
        setIdx((i) => Math.min(files.length - 1, i + 1));
      } else if (e.key === "Home") {
        setIdx(0);
      } else if (e.key === "End") {
        setIdx(Math.max(0, files.length - 1));
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [modalOpen, files.length]);

  const beadsDisabled = !!health && !health.beadsDb;
  const total = files.length;
  const pos = total > 0 ? idx + 1 : 0;

  return (
    <>
      <header>
        <div className="brand">
          <span className="live-dot"></span>cly diff
        </div>
        <div className="progress-wrap">
          <div className="progress-track">
            <span className="progress-label l">old</span>
            <div className="progress-bg">
              <div
                className="progress-fill"
                style={{
                  width: total > 0 ? `${(pos / total) * 100}%` : "0%",
                }}
              />
            </div>
            <span className="progress-label r">now</span>
          </div>
          <div className="progress-meta">
            <span>
              <b>{pos || "—"}</b> of <b>{total || "—"}</b>
            </span>
            <span className="muted">
              {current ? current.path : "no files"}
            </span>
          </div>
        </div>
        <button
          className="bead-btn"
          onClick={() => setModalOpen(true)}
          disabled={beadsDisabled}
          title={beadsDisabled ? "bd not available" : "New bead (n)"}
        >
          ◆ <span className="bead-btn-label">bead</span>
          <span className="kbd-inline">n</span>
        </button>
      </header>

      <main>
        {beadsDisabled && (
          <div className="banner">
            bd database not found. Run <code>bd init</code> to enable bead
            creation.
          </div>
        )}
        {error && <div className="banner err">{error}</div>}

        {files.length === 0 ? (
          <div className="empty-deck">
            <p>nothing to review — working tree is clean</p>
            <button
              className="bead-btn"
              onClick={() => setModalOpen(true)}
              disabled={beadsDisabled}
            >
              create global bead
            </button>
          </div>
        ) : (
          <div className="deck">
            {current && diff && <DiffCard file={current} diff={diff} />}
          </div>
        )}
      </main>

      <footer>
        <button
          className="nav-btn"
          onClick={() => setIdx(0)}
          title="Oldest (Home)"
        >
          ⇤
        </button>
        <button
          className="nav-btn"
          onClick={() => setIdx((i) => Math.max(0, i - 1))}
          title="Previous file (←)"
        >
          ◀
        </button>
        <button
          className="nav-btn"
          onClick={() =>
            setIdx((i) => Math.min(files.length - 1, i + 1))
          }
          title="Next file (→)"
        >
          ▶
        </button>
        <button
          className="nav-btn"
          onClick={() => setIdx(Math.max(0, files.length - 1))}
          title="Latest (End)"
        >
          ⇥
        </button>
      </footer>

      <div className="kbd-hint" aria-hidden>
        <kbd>←</kbd>/<kbd>→</kbd> nav · <kbd>n</kbd> bead
      </div>

      {modalOpen && (
        <BeadModal
          contextPath={current?.path ?? ""}
          availableFiles={files.map((f) => f.path)}
          onClose={() => setModalOpen(false)}
          onCreated={(id) => {
            setModalOpen(false);
            showToast(`bead ${id} created`);
          }}
        />
      )}

      <div className={`toast ${toast ? "show" : ""}`}>{toast ?? ""}</div>
    </>
  );
}

function DiffCard({ file, diff }: { file: File; diff: FileDiff }) {
  const slash = file.path.lastIndexOf("/");
  const dir = slash >= 0 ? file.path.slice(0, slash + 1) : "";
  const base = slash >= 0 ? file.path.slice(slash + 1) : file.path;
  return (
    <div className="diff-card">
      <div className="diff-card-head">
        <span className="path">
          <span className="dir">{dir}</span>
          {base}
        </span>
        <span className="stats">
          {file.additions > 0 && (
            <span className="add">+{file.additions}</span>
          )}{" "}
          {file.deletions > 0 && (
            <span className="del">-{file.deletions}</span>
          )}
        </span>
      </div>
      {(diff.hunks ?? []).map((h, i) => (
        <div className="hunk" key={i}>
          <div className="hunk-header">{h.header}</div>
          {h.lines.map((l, j) => (
            <div className={`line ${l.kind}`} key={j}>
              <span className="ln">{l.old || ""}</span>
              <span className="ln">{l.new || ""}</span>
              <span className="txt">{l.text}</span>
            </div>
          ))}
        </div>
      ))}
    </div>
  );
}

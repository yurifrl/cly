import { useEffect, useRef, useState } from "react";
import * as api from "../api";
import type { BeadPriority, BeadType } from "../types";
import { ChipsInput } from "./ChipsInput";
import { ContextPicker } from "./ContextPicker";

interface Props {
  contextPath: string;
  availableFiles: string[];
  onClose: () => void;
  onCreated: (id: string) => void;
}

const TYPES: BeadType[] = ["bug", "feature", "task", "chore", "decision"];
const PRIORITIES: BeadPriority[] = ["P0", "P1", "P2", "P3", "P4"];

export function BeadModal({
  contextPath,
  availableFiles,
  onClose,
  onCreated,
}: Props) {
  const [title, setTitle] = useState("");
  const [desc, setDesc] = useState("");
  const [type, setType] = useState<BeadType>("task");
  const [priority, setPriority] = useState<BeadPriority>("P2");
  const [ctx, setCtx] = useState(contextPath);
  const [labels, setLabels] = useState<string[]>([]);
  const [known, setKnown] = useState<string[]>([]);
  const [err, setErr] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const titleRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    titleRef.current?.focus();
    void api.listLabels().then(setKnown).catch(() => setKnown([]));
  }, []);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        onClose();
      } else if ((e.metaKey || e.ctrlKey) && e.key === "Enter") {
        void submit();
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [title, desc, type, priority, ctx, labels]);

  async function submit() {
    if (!title.trim() || busy) return;
    setBusy(true);
    setErr(null);
    try {
      const res = await api.createBead({
        title: title.trim(),
        description: desc,
        type,
        priority,
        context: ctx || undefined,
        labels: labels.length > 0 ? labels : undefined,
      });
      onCreated(res.id);
    } catch (e) {
      setErr((e as Error).message);
    } finally {
      setBusy(false);
    }
  }

  return (
    <div
      className="modal-backdrop show"
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
    >
      <div className="modal" role="dialog" aria-modal>
        <div className="modal-head">
          <h2>◆ New bead</h2>
          <button
            type="button"
            className="modal-close"
            onClick={onClose}
            aria-label="Close"
          >
            ✕
          </button>
        </div>

        <div className="modal-body">
          <div className="field">
            <label className="field-label" htmlFor="bf-title">
              title <span className="req">*</span>
            </label>
            <input
              id="bf-title"
              ref={titleRef}
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="Short, actionable summary"
              autoComplete="off"
            />
          </div>

          <div className="field">
            <label className="field-label" htmlFor="bf-desc">
              description
            </label>
            <textarea
              id="bf-desc"
              rows={3}
              value={desc}
              onChange={(e) => setDesc(e.target.value)}
              placeholder="What needs to happen and why"
            />
          </div>

          <div className="field-row">
            <div className="field">
              <span className="field-label">type</span>
              <div className="seg">
                {TYPES.map((t) => (
                  <button
                    key={t}
                    type="button"
                    className={t === type ? "on" : ""}
                    onClick={() => setType(t)}
                  >
                    {t}
                  </button>
                ))}
              </div>
            </div>
            <div className="field" style={{ maxWidth: 160 }}>
              <span className="field-label">priority</span>
              <div className="seg">
                {PRIORITIES.map((p) => (
                  <button
                    key={p}
                    type="button"
                    className={p === priority ? "on" : ""}
                    onClick={() => setPriority(p)}
                  >
                    {p}
                  </button>
                ))}
              </div>
            </div>
          </div>

          <div className="field" style={{ position: "relative" }}>
            <span className="field-label">
              context
              <span
                style={{
                  textTransform: "none",
                  letterSpacing: 0,
                  fontWeight: "normal",
                  color: "var(--muted)",
                  marginLeft: 6,
                }}
              >
                {ctx ? "from current diff" : "global bead"}
              </span>
            </span>
            <ContextPicker
              value={ctx}
              onChange={setCtx}
              files={availableFiles}
            />
          </div>

          <div className="field" style={{ position: "relative" }}>
            <span className="field-label">labels</span>
            <ChipsInput
              value={labels}
              onChange={setLabels}
              suggestions={known}
            />
          </div>

          {err && (
            <div style={{ color: "var(--red)", marginTop: 8 }}>{err}</div>
          )}
        </div>

        <div className="modal-foot">
          <span className="cmd-hint">
            <kbd>Esc</kbd> cancel · <kbd>⌘</kbd>+<kbd>Enter</kbd> create
          </span>
          <button type="button" className="btn-ghost" onClick={onClose}>
            cancel
          </button>
          <button
            type="button"
            className="btn-primary"
            onClick={submit}
            disabled={!title.trim() || busy}
          >
            {busy ? "creating…" : "create"}
          </button>
        </div>
      </div>
    </div>
  );
}

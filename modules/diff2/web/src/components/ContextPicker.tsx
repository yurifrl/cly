import { useEffect, useMemo, useRef, useState } from "react";

interface Props {
  value: string;
  onChange: (v: string) => void;
  files: string[];
}

// Mockup-style context picker: button that opens a filterable dropdown
// of files. Empty value = "global — no context".
export function ContextPicker({ value, onChange, files }: Props) {
  const [open, setOpen] = useState(false);
  const [q, setQ] = useState("");
  const [hl, setHl] = useState(0);
  const searchRef = useRef<HTMLInputElement>(null);
  const popRef = useRef<HTMLDivElement>(null);

  const items = useMemo(() => {
    const filtered = q
      ? files.filter((p) => p.toLowerCase().includes(q.toLowerCase()))
      : files;
    return [
      { kind: "global" as const, path: "" },
      ...filtered.map((p) => ({ kind: "file" as const, path: p })),
    ];
  }, [q, files]);

  useEffect(() => {
    if (open) {
      setHl(0);
      searchRef.current?.focus();
    }
  }, [open]);

  useEffect(() => {
    const onClick = (e: MouseEvent) => {
      if (!popRef.current) return;
      if (popRef.current.contains(e.target as Node)) return;
      setOpen(false);
    };
    document.addEventListener("mousedown", onClick);
    return () => document.removeEventListener("mousedown", onClick);
  }, []);

  function pick(path: string) {
    onChange(path);
    setOpen(false);
    setQ("");
  }

  function onKey(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === "ArrowDown") {
      setHl((h) => (h + 1) % Math.max(1, items.length));
      e.preventDefault();
    } else if (e.key === "ArrowUp") {
      setHl((h) =>
        (h - 1 + Math.max(1, items.length)) % Math.max(1, items.length),
      );
      e.preventDefault();
    } else if (e.key === "Enter") {
      const it = items[hl];
      if (it) pick(it.path);
      e.preventDefault();
    } else if (e.key === "Escape") {
      setOpen(false);
      e.preventDefault();
    }
  }

  const label = value || "global — no context";
  const isGlobal = !value;

  return (
    <div ref={popRef} style={{ position: "relative" }}>
      <button
        type="button"
        className="picker-btn"
        onClick={() => setOpen((o) => !o)}
        aria-haspopup="listbox"
        aria-expanded={open}
      >
        <span className="pbtn-icon">{isGlobal ? "🌐" : "📄"}</span>
        <span className={`pbtn-text ${isGlobal ? "global" : ""}`}>
          {label}
        </span>
        <span className="pbtn-chev">▾</span>
      </button>

      {open && (
        <div className="picker-pop show" role="listbox">
          <input
            ref={searchRef}
            type="text"
            placeholder="search context…"
            value={q}
            onChange={(e) => {
              setQ(e.target.value);
              setHl(0);
            }}
            onKeyDown={onKey}
            autoComplete="off"
          />
          <div className="picker-list">
            {items.length === 0 && (
              <div className="picker-empty">no match</div>
            )}
            {items.map((it, i) => {
              const sel =
                (it.kind === "global" && isGlobal) ||
                (it.kind === "file" && it.path === value);
              const cls = [
                "picker-opt",
                it.kind === "global" ? "global" : "",
                sel ? "selected" : "",
                i === hl ? "hl" : "",
              ]
                .filter(Boolean)
                .join(" ");
              if (it.kind === "global") {
                return (
                  <div
                    key="_global"
                    className={cls}
                    onMouseDown={(e) => e.preventDefault()}
                    onClick={() => pick("")}
                  >
                    <span>🌐</span>
                    <span className="po-path">global — no context</span>
                  </div>
                );
              }
              const slash = it.path.lastIndexOf("/");
              const dir = slash >= 0 ? it.path.slice(0, slash + 1) : "";
              const base = slash >= 0 ? it.path.slice(slash + 1) : it.path;
              return (
                <div
                  key={it.path}
                  className={cls}
                  onMouseDown={(e) => e.preventDefault()}
                  onClick={() => pick(it.path)}
                >
                  <span>📄</span>
                  <span className="po-path">
                    <span className="dir">{dir}</span>
                    {base}
                  </span>
                </div>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}

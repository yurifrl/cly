import { useMemo, useRef, useState } from "react";

interface Props {
  value: string[];
  onChange: (v: string[]) => void;
  suggestions: string[];
}

// Labels chip input + autosuggest dropdown.
// Class names mirror the mockup so its CSS applies verbatim.
export function ChipsInput({ value, onChange, suggestions }: Props) {
  const [input, setInput] = useState("");
  const [open, setOpen] = useState(false);
  const [hl, setHl] = useState(0);
  const wrapRef = useRef<HTMLDivElement>(null);

  const matches = useMemo(() => {
    const q = input.trim().toLowerCase();
    const base = suggestions.filter((s) => !value.includes(s));
    const filtered = q
      ? base.filter((s) => s.toLowerCase().includes(q))
      : base;
    const limited = filtered.slice(0, 10);
    const exact =
      q && (suggestions.includes(q) || value.includes(q));
    const items: { value: string; kind: "existing" | "create" }[] =
      limited.map((s) => ({ value: s, kind: "existing" }));
    if (q && !exact) items.push({ value: q, kind: "create" });
    return items;
  }, [input, value, suggestions]);

  function add(v: string) {
    const t = v.trim().toLowerCase().replace(/\s+/g, "-");
    if (!t) return;
    if (value.includes(t)) return;
    onChange([...value, t]);
    setInput("");
    setOpen(false);
    setHl(0);
  }

  function remove(idx: number) {
    onChange(value.filter((_, i) => i !== idx));
  }

  function onKey(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === "ArrowDown") {
      if (!open) setOpen(true);
      else setHl((h) => (h + 1) % Math.max(1, matches.length));
      e.preventDefault();
    } else if (e.key === "ArrowUp") {
      if (!open) setOpen(true);
      else
        setHl((h) =>
          (h - 1 + Math.max(1, matches.length)) %
          Math.max(1, matches.length),
        );
      e.preventDefault();
    } else if (e.key === "Enter") {
      if (open && matches[hl]) add(matches[hl].value);
      else if (input.trim()) add(input);
      e.preventDefault();
    } else if (e.key === "," || e.key === "Tab") {
      if (input.trim()) {
        add(input);
        e.preventDefault();
      }
    } else if (e.key === "Backspace" && !input && value.length > 0) {
      remove(value.length - 1);
      e.preventDefault();
    } else if (e.key === "Escape") {
      setOpen(false);
    }
  }

  return (
    <>
      <div
        ref={wrapRef}
        className="chips-input"
        onClick={() => wrapRef.current?.querySelector("input")?.focus()}
      >
        <span>
          {value.map((l, i) => (
            <span key={l} className="chip known">
              {l}
              <button
                type="button"
                className="chip-remove"
                onClick={(e) => {
                  e.stopPropagation();
                  remove(i);
                }}
                aria-label={`remove ${l}`}
              >
                ✕
              </button>
            </span>
          ))}
        </span>
        <input
          value={input}
          onFocus={() => setOpen(true)}
          onBlur={() => window.setTimeout(() => setOpen(false), 150)}
          onChange={(e) => {
            setInput(e.target.value);
            setOpen(true);
            setHl(0);
          }}
          onKeyDown={onKey}
          placeholder={value.length === 0 ? "add label…" : ""}
          autoComplete="off"
        />
      </div>
      {open && matches.length > 0 && (
        <div className="chips-suggest show" role="listbox">
          {matches.map((m, i) => (
            <div
              key={m.value + m.kind}
              className={`sg ${m.kind === "create" ? "create" : ""} ${
                i === hl ? "hl" : ""
              }`}
              onMouseDown={(e) => {
                e.preventDefault();
                add(m.value);
              }}
              onMouseEnter={() => setHl(i)}
            >
              <span>{m.kind === "create" ? "➕" : "🏷"}</span>
              <span>
                {m.kind === "create" ? `create "${m.value}"` : m.value}
              </span>
              <span className="sg-tag">
                {m.kind === "create" ? "new" : "existing"}
              </span>
            </div>
          ))}
        </div>
      )}
    </>
  );
}

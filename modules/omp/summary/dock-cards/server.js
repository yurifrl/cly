// omp cards → Dock browser pane (saved web fallback)
// Serves the live workspace cards (from `cmux workspace list --json`) as HTML.
// The Dock hosts this as a browser pane; the page self-refreshes via polling.
//
// Kept as an alternative to the native omp-cards right sidebar
// (modules/omp/summary/omp-cards.swift), which is the shipped default.
// Not wired into the binary; run standalone with `bun <this file>` (port 8791).

const PORT = 8791;
const REFRESH_MS = 2000;
const CLI_CACHE_MS = 1000;

let cachedAt = 0;
let cachedWorkspaces = null;

async function workspaces() {
	const now = Date.now();
	if (cachedWorkspaces && now - cachedAt < CLI_CACHE_MS) return cachedWorkspaces;
	const proc = Bun.spawnSync(["cmux", "workspace", "list", "--json"], {
		stdout: "pipe",
		stderr: "ignore",
	});
	try {
		const parsed = JSON.parse(new TextDecoder().decode(proc.stdout));
		cachedWorkspaces = parsed.workspaces ?? [];
	} catch {
		// keep last good cache on CLI hiccup
		if (cachedWorkspaces === null) cachedWorkspaces = [];
	}
	cachedAt = now;
	return cachedWorkspaces;
}

const esc = (s) =>
	String(s ?? "").replace(/[&<>"']/g, (c) => ({
		"&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;",
	}[c]));

function cardsHtml(list) {
	if (!list.length) return `<div class="empty">no sessions</div>`;
	return list
		.map((w) => {
			const segs = String(w.description ?? "")
				.split("|")
				.map((s) => s.trim())
				.filter(Boolean)
				.slice(0, 14);
			const rows = segs.length
				? segs.map((s) => `<div class="seg">${esc(s)}</div>`).join("")
				: `<div class="seg dim">no session data yet</div>`;
			const sel = w.selected ? `<span class="dot sel"></span>` : `<span class="dot"></span>`;
			return `<div class="card${w.selected ? " selected" : ""}">
				<div class="title">${sel}${esc(w.title ?? w.custom_title ?? w.ref)}</div>
				${rows}
			</div>`;
		})
		.join("");
}

const page = (list) => `<!doctype html>
<html><head><meta charset="utf-8">
<style>
	* { box-sizing: border-box; }
	body { margin: 0; background: #16181d; color: #c9cdd4;
		font: 10px/1.45 ui-monospace, SFMono-Regular, Menlo, monospace; }
	#hdr { display: flex; align-items: center; gap: 6px;
		padding: 8px 10px 4px; color: #8a919c; font-size: 10px; }
	#hdr b { color: #e6e9ef; font-weight: 600; }
	#cards { padding: 0 8px 12px; }
	.card { border: 1px solid #262a33; border-radius: 6px; padding: 6px 8px; margin-bottom: 6px;
		background: #1b1e24; }
	.card.selected { border-color: #4c8dff; }
	.title { color: #e6e9ef; font-weight: 600; font-size: 11px; margin-bottom: 3px;
		display: flex; align-items: center; gap: 5px; }
	.card.selected .title { color: #7fa9ff; }
	.dot { width: 6px; height: 6px; border-radius: 50%; background: #3a3f49; flex: none; }
	.dot.sel { background: #4c8dff; }
	.seg { white-space: nowrap; overflow: hidden; text-overflow: ellipsis; color: #9aa2ad; }
	.seg.dim { color: #5b626d; font-style: italic; }
	.empty { padding: 12px; color: #5b626d; font-style: italic; }
</style></head>
<body>
	<div id="hdr">✨ omp cards · <b>${list.length}</b> sessions</div>
	<div id="cards">${cardsHtml(list)}</div>
	<script>
		const el = document.getElementById("cards");
		const hdr = document.getElementById("hdr");
		setInterval(async () => {
			try {
				const r = await fetch("/cards");
				const d = await r.json();
				el.innerHTML = d.cards;
				hdr.innerHTML = '✨ omp cards · <b>' + d.count + '</b> sessions';
			} catch {}
		}, ${REFRESH_MS});
	</script>
</body></html>`;

Bun.serve({
	port: PORT,
	hostname: "127.0.0.1",
	async fetch(req) {
		const url = new URL(req.url);
		const list = await workspaces();
		if (url.pathname === "/cards") {
			return Response.json({ count: list.length, cards: cardsHtml(list) });
		}
		return new Response(page(list), {
			headers: { "content-type": "text/html; charset=utf-8" },
		});
	},
});
console.log(`omp-cards dock server on http://127.0.0.1:${PORT}`);

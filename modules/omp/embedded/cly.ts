// ⚠️  DO NOT EDIT — generated/shipped by cly.
// Source: cly/modules/omp/embedded/cly.ts. Installed via `cly omp extensions install`.
// Edits here are overwritten on the next install. Change the source in the cly repo.
import type { ExtensionAPI } from "@oh-my-pi/pi-coding-agent";
import { spawn } from "node:child_process";
import fs from "node:fs";
import path from "node:path";

/**
 * omp-cly — omp extension bundled by cly.
 *
 * Commands:
 *   /save [name] [description="..."]
 *     Invokes `cly as save <id> <name> <description>` with deterministic
 *     prefilled values and surfaces the result via ctx.ui.notify. The
 *     command is fully handled by this extension — it does NOT feed any
 *     message back to the agent, so the AI will not react to /save.
 *     Overrides:
 *       - positional text (after kv extraction) → name
 *       - description="..."                     → description
 */

interface SaveArgs {
	id: string;
	name: string;
	description: string;
}

interface SaveOverrides {
	name?: string;
	description?: string;
}

type NotifyLevel = "info" | "success" | "error";

const KV_RE = /(\w+)=(?:"([^"]*)"|(\S+))/g;

export function parseSaveArgs(raw: string): SaveOverrides {
	const overrides: Record<string, string> = {};
	const rest = raw.replace(KV_RE, (_m, key: string, quoted?: string, bare?: string) => {
		overrides[key] = quoted !== undefined ? quoted : bare !== undefined ? bare : "";
		return " ";
	});
	const name = rest.trim();
	const out: SaveOverrides = {};
	if (name.length > 0) out.name = name;
	if (typeof overrides.description === "string") out.description = overrides.description;
	return out;
}

function slugify(input: string): string {
	const s = input.toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-+|-+$/g, "");
	return s || "session";
}

/** Reads fields off the extension context without trusting its shape. */
function readContextFields(ctx: unknown): { cwd: string; sessionFile: string | undefined; hasUI: boolean; ui: unknown } {
	const obj = (ctx && typeof ctx === "object" ? ctx : {}) as Record<string, unknown>;
	const cwd = typeof obj.cwd === "string" ? obj.cwd : process.cwd();
	const sm = obj.sessionManager as Record<string, unknown> | undefined;
	const getSessionFile = sm?.getSessionFile;
	const sessionFile = typeof getSessionFile === "function" ? (getSessionFile.call(sm) as unknown) : undefined;
	return {
		cwd,
		sessionFile: typeof sessionFile === "string" ? sessionFile : undefined,
		hasUI: obj.hasUI === true,
		ui: obj.ui,
	};
}

function getSessionId(sessionFile: string | undefined, cwd: string): string {
	try {
		const direct = readIdFromSessionFile(sessionFile);
		if (direct) return direct;

		// Fallback: scan session dir for latest file
		const home = process.env.HOME || "";
		const trimmed = cwd.replace(/^\/+|\/+$/g, "");
		const encoded = "-" + trimmed.replace(/\//g, "-");
		const sessionDir = path.join(home, ".omp", "agent", "sessions", encoded);
		if (fs.existsSync(sessionDir)) {
			const candidates = fs.readdirSync(sessionDir)
				.filter((n) => n.endsWith(".jsonl"))
				.map((n) => path.join(sessionDir, n))
				.filter((f) => { try { return fs.statSync(f).isFile(); } catch { return false; } })
				.sort((a, b) => fs.statSync(b).mtimeMs - fs.statSync(a).mtimeMs);
			if (candidates.length > 0) {
				const fallbackId = readIdFromSessionFile(candidates[0]);
				if (fallbackId) return fallbackId;
			}
		}
	} catch {
		// ignore
	}
	return "";
}

function readIdFromSessionFile(file: string | undefined): string {
	try {
		if (typeof file !== "string" || file.length === 0) return "";
		const lines = fs.readFileSync(file, "utf-8").split("\n");
		const first = lines[0] ? lines[0].trim() : "";
		if (first) {
			const obj = JSON.parse(first) as { id?: unknown };
			if (typeof obj.id === "string" && obj.id.length > 0) return obj.id;
		}
	} catch {
		// ignore
	}
	return "";
}

function getSummaryName(pi: ExtensionAPI): string {
	try {
		const getter = (pi as unknown as Record<string, unknown>).getSessionName;
		if (typeof getter === "function") {
			const n = (getter as () => unknown).call(pi);
			if (typeof n === "string") return n.trim();
		}
	} catch {
		// ignore
	}
	return "";
}

function defaultPrefills(pi: ExtensionAPI, sessionFile: string | undefined, cwd: string): SaveArgs {
	const slug = slugify(path.basename(cwd));
	let id = getSessionId(sessionFile, cwd);
	if (id.length === 0) {
		const ts = new Date().toISOString().slice(0, 16).replace(/[:T]/g, "-");
		id = slug + "-" + ts;
	}
	const summary = getSummaryName(pi);
	return {
		id,
		name: summary.length > 0 ? summary : slug,
		description: "omp session in " + cwd,
	};
}

function runCly(args: string[]): Promise<{ code: number; stdout: string; stderr: string }> {
	const { promise, resolve } = Promise.withResolvers<{ code: number; stdout: string; stderr: string }>();
	const child = spawn("cly", args, { stdio: ["ignore", "pipe", "pipe"] });
	let stdout = "";
	let stderr = "";
	child.stdout.on("data", (d) => { stdout += d.toString(); });
	child.stderr.on("data", (d) => { stderr += d.toString(); });
	child.on("error", (err) => resolve({ code: -1, stdout, stderr: stderr + String(err) }));
	child.on("close", (code) => resolve({ code: code == null ? -1 : code, stdout, stderr }));
	return promise;
}

function notify(ctx: unknown, msg: string, level: NotifyLevel): void {
	try {
		const obj = (ctx && typeof ctx === "object" ? ctx : {}) as Record<string, unknown>;
		if (obj.hasUI !== true) return;
		const ui = obj.ui as Record<string, unknown> | undefined;
		const notifyFn = ui?.notify;
		if (typeof notifyFn === "function") {
			(notifyFn as (msg: string, level: NotifyLevel) => void).call(ui, msg, level);
		}
	} catch {
		// ignore
	}
}

export default function piClyExtension(pi: ExtensionAPI): void {
	// Auto-apply session name from $CLY_SESSION_NAME on session start.
	// Set by `cly omp -n NAME` (or any caller of pkg/envs.SetSessionName).
	// Skips silently when the env var is empty or already matches.
	pi.on("session_start", async (_event, ctx) => {
		try {
			const desired = (process.env.CLY_SESSION_NAME ?? process.env.CLAUDE_SESSION_NAME ?? "").trim();
			if (desired.length === 0) return;
			const current = getSummaryName(pi);
			if (current === desired) return;
			const setter = (pi as unknown as Record<string, unknown>).setSessionName;
			if (typeof setter === "function") {
				(setter as (name: string) => void).call(pi, desired);
			}
		} catch {
			// best-effort; never block session startup
		}
	});

	pi.registerCommand("save", {
		description: 'Save the current agent session. Usage: /save [name] [description="..."]. Invokes `cly as save`. Does not forward to the agent.',
		handler: async (args: string, ctx: unknown) => {
			const { cwd, sessionFile, hasUI: _hasUI, ui: _ui } = readContextFields(ctx);
			const prefills = defaultPrefills(pi, sessionFile, cwd);
			const overrides = parseSaveArgs(args || "");
			const resolved: SaveArgs = {
				id: prefills.id,
				name: overrides.name || prefills.name,
				description: overrides.description || prefills.description,
			};

			const cliArgs = ["as", "save", resolved.id, "--name", resolved.name, "--description", resolved.description, "--override"];
			const result = await runCly(cliArgs);

			if (result.code === 0) {
				notify(ctx, "/save → " + resolved.name + " (" + resolved.id + ")", "success");
			} else {
				const err = result.stderr.trim() || result.stdout.trim() || ("exit " + result.code);
				notify(ctx, "/save failed: " + err, "error");
			}
			// Deliberately do NOT forward a user message — /save must not trigger the agent.
		},
	});
}

import type { ExtensionAPI } from "@mariozechner/pi-coding-agent";
import { spawn } from "node:child_process";
import fs from "node:fs";
import path from "node:path";

/**
 * pi-cly — pi extension bundled by cly.
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

const KV_RE = /(\w+)=(?:"([^"]*)"|(\S+))/g;

export function parseSaveArgs(raw: string): { name?: string; description?: string } {
	const overrides: Record<string, string> = {};
	const rest = raw.replace(KV_RE, (_m, key: string, quoted?: string, bare?: string) => {
		overrides[key] = quoted !== undefined ? quoted : bare !== undefined ? bare : "";
		return " ";
	});
	const name = rest.trim();
	const out: { name?: string; description?: string } = {};
	if (name.length > 0) out.name = name;
	if (typeof overrides.description === "string") out.description = overrides.description;
	return out;
}

function slugify(input: string): string {
	const s = input.toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-+|-+$/g, "");
	return s || "session";
}

function getSessionId(ctx: any, cwd: string): string {
	try {
		// Try ctx.sessionManager.getSessionFile() first
		const sm = ctx && ctx.sessionManager;
		const file = sm && typeof sm.getSessionFile === "function" ? sm.getSessionFile() : undefined;
		const id = readIdFromSessionFile(file);
		if (id) return id;

		// Fallback: scan session dir for latest file (same as checkpoint.ts)
		const home = process.env.HOME || "";
		const trimmed = cwd.replace(/^\/+|\/+$/g, "");
		const encoded = "--" + trimmed.replace(/\//g, "-") + "--";
		const sessionDir = path.join(home, ".pi", "agent", "sessions", encoded);
		if (fs.existsSync(sessionDir)) {
			const candidates = fs.readdirSync(sessionDir)
				.filter(function (n) { return n.endsWith(".jsonl"); })
				.map(function (n) { return path.join(sessionDir, n); })
				.filter(function (f) { try { return fs.statSync(f).isFile(); } catch (e) { return false; } })
				.sort(function (a, b) { return fs.statSync(b).mtimeMs - fs.statSync(a).mtimeMs; });
			if (candidates.length > 0) {
				const fallbackId = readIdFromSessionFile(candidates[0]);
				if (fallbackId) return fallbackId;
			}
		}
	} catch (e) {
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
			const obj = JSON.parse(first);
			if (obj.id) return obj.id;
		}
	} catch (e) {
		// ignore
	}
	return "";
}

function getSummaryName(pi: any): string {
	try {
		if (typeof pi.getSessionName === "function") {
			const n = pi.getSessionName();
			if (typeof n === "string") return n.trim();
		}
	} catch {
		// ignore
	}
	return "";
}

function defaultPrefills(pi: ExtensionAPI, ctx: any, cwd: string): SaveArgs {
	const slug = slugify(path.basename(cwd));
	let id = getSessionId(ctx, cwd);
	if (!id) {
		const ts = new Date().toISOString().slice(0, 16).replace(/[:T]/g, "-");
		id = slug + "-" + ts;
	}
	const summary = getSummaryName(pi);
	return {
		id,
		name: summary || slug,
		description: "pi session in " + cwd,
	};
}

function runCly(args: string[]): Promise<{ code: number; stdout: string; stderr: string }> {
	return new Promise((resolve) => {
		const child = spawn("cly", args, { stdio: ["ignore", "pipe", "pipe"] });
		let stdout = "";
		let stderr = "";
		child.stdout.on("data", (d) => { stdout += d.toString(); });
		child.stderr.on("data", (d) => { stderr += d.toString(); });
		child.on("error", (err) => resolve({ code: -1, stdout, stderr: stderr + String(err) }));
		child.on("close", (code) => resolve({ code: code == null ? -1 : code, stdout, stderr }));
	});
}

function notify(ctx: any, msg: string, level: "info" | "success" | "error"): void {
	try {
		if (ctx && ctx.hasUI && ctx.ui && typeof ctx.ui.notify === "function") {
			ctx.ui.notify(msg, level);
		}
	} catch {
		// ignore
	}
}

export default function piClyExtension(pi: ExtensionAPI): void {
	pi.registerCommand("save", {
		description: 'Save the current agent session. Usage: /save [name] [description="..."]. Invokes `cly as save`. Does not forward to the agent.',
		handler: async (args: string, ctx: any) => {
			const cwd = (ctx && typeof ctx.cwd === "string") ? ctx.cwd : process.cwd();
			const prefills = defaultPrefills(pi, ctx, cwd);
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
			// Deliberately do NOT call pi.sendUserMessage — /save must not trigger the agent.
		},
	});
}

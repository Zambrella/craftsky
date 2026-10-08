import { promises as fs } from "node:fs";
import path from "node:path";
import { NextResponse } from "next/server";
import { rejectCrossSiteWrite } from "@/lib/request-guard";
import { readJsonBody } from "@/lib/request-body";

export async function POST(req: Request) {
  const blocked = rejectCrossSiteWrite(req);
  if (blocked) return NextResponse.json({ error: blocked.error }, { status: blocked.status });
  const parsed = await readJsonBody(req, 64 * 1024 * 1024);
  if (parsed.response) return parsed.response;
  const payload = parsed.value as { zip?: unknown; device?: unknown };
  if (typeof payload?.zip !== "string" || !/^[A-Za-z0-9+/]*={0,2}$/.test(payload.zip)) {
    return NextResponse.json({ error: "Invalid ZIP" }, { status: 400 });
  }
  if (typeof payload.device !== "string" || !["iphone", "ipad", "android", "feature-graphic"].includes(payload.device)) return NextResponse.json({ error: "Unsupported device" }, { status: 400 });
  const bytes = Buffer.from(payload.zip, "base64");
  if (bytes.length < 4 || bytes.readUInt32LE(0) !== 0x04034b50) return NextResponse.json({ error: "Invalid ZIP" }, { status: 400 });
  const directory = path.join(process.cwd(), "exports");
  await fs.mkdir(directory, { recursive: true });
  await fs.writeFile(path.join(directory, `craftsky-${payload.device}-en.zip`), bytes);
  return NextResponse.json({ ok: true });
}

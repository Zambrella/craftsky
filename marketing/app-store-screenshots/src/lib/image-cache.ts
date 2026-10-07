"use client";
// Pre-loads images as base64 data URIs so html-to-image exports without
// non-deterministic image fetch races. Always use img(path) in render.

const cache = new Map<string, string>();
const failed = new Set<string>();
const pending = new Map<string, Promise<void>>();

async function fetchAsDataUrl(path: string): Promise<string | null> {
  try {
    const resp = await fetch(path, { signal: AbortSignal.timeout(10000) });
    if (!resp.ok) return null;
    const blob = await resp.blob();
    const data = await new Promise<string>((resolve, reject) => {
      const reader = new FileReader();
      reader.onloadend = () => resolve(reader.result as string);
      reader.onerror = reject;
      reader.readAsDataURL(blob);
    });
    // A successful HTTP response may still be an HTML error page or corrupt
    // image. Catch that before export silently substitutes an empty device.
    const image = new Image();
    image.src = data;
    await image.decode();
    return data;
  } catch {
    return null;
  }
}

export async function preloadImages(
  paths: string[],
  options: { retryFailed?: boolean } = {},
): Promise<void> {
  await Promise.all(
    paths
      .filter(Boolean)
      .filter((p) => !cache.has(p) && (options.retryFailed || !failed.has(p)))
      .map((p) => {
        const existing = pending.get(p);
        if (existing) return existing;
        const task = fetchAsDataUrl(p).then((data) => {
          // A newer upload may have populated the same content-addressed URL.
          if (cache.has(p)) return;
          if (data) { cache.set(p, data); failed.delete(p); }
          else failed.add(p);
        }).finally(() => pending.delete(p));
        pending.set(p, task);
        return task;
      }),
  );
}

export function img(path: string | undefined): string {
  if (!path) return "";
  if (path.startsWith("data:")) return path;
  if (failed.has(path)) return "";
  return cache.get(path) || path;
}

export function setImage(path: string, dataUrl: string) {
  cache.set(path, dataUrl);
  failed.delete(path);
}

export function didFail(path: string | undefined): boolean {
  if (!path) return false;
  return failed.has(path);
}

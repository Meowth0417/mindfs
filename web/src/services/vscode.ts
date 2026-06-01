import { protectedJSON } from "./api";
import { appURL } from "./base";
import { openExternalURL } from "./platformNavigation";
import { isCapacitorRuntime } from "./runtime";

export function canLaunchVSCode(): boolean {
  if (typeof window === "undefined" || isCapacitorRuntime()) {
    return false;
  }
  const hostname = window.location.hostname.toLowerCase();
  return hostname === "localhost" || hostname === "127.0.0.1" || hostname === "[::1]";
}

export type ExternalLaunchAction = "vscode" | "explorer" | "powershell";

export async function launchVSCode(input: {
  rootId: string;
  path?: string;
  line?: number;
  column?: number;
  action?: ExternalLaunchAction;
}): Promise<{ mode: string; target: string; action: ExternalLaunchAction }> {
  return protectedJSON<{ mode: string; target: string; action: ExternalLaunchAction }>(appURL("/api/vscode/open"), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      root: input.rootId,
      path: input.path || "",
      line: input.line || 0,
      column: input.column || 0,
      action: input.action || "vscode",
    }),
  });
}

export function openVSCodeProtocol(targetPath: string): void {
  const target = String(targetPath || "").trim();
  if (!target) {
    return;
  }
  const normalized = target.replace(/\\/g, "/");
  const slashPrefixed =
    /^[A-Za-z]:\//.test(normalized) && !normalized.startsWith("/")
      ? `/${normalized}`
      : normalized;
  openExternalURL(`vscode://file${encodeURI(slashPrefixed)}`);
}

import { protectedJSON } from "./api";
import { appURL } from "./base";

export type EnvironmentCommandIcon = "run" | "tool" | "test" | "other";

export type EnvironmentCommand = {
  id: string;
  name: string;
  icon: EnvironmentCommandIcon;
  command: string;
  is_default: boolean;
};

export type EnvironmentCommandRunStatus = "running" | "completed" | "failed";

export type EnvironmentCommandRunResult = {
  id: string;
  name: string;
  command: string;
  target: string;
  action: string;
  run_id: string;
  root_id: string;
  status: EnvironmentCommandRunStatus;
  started_at?: string;
};

export async function fetchEnvironmentCommands(rootId: string): Promise<EnvironmentCommand[]> {
  const response = await protectedJSON<{ items: EnvironmentCommand[] }>(
    appURL(`/api/environment/commands?root=${encodeURIComponent(rootId)}`),
  );
  return response.items || [];
}

export async function saveEnvironmentCommand(input: {
  rootId: string;
  name: string;
  icon: EnvironmentCommandIcon;
  command: string;
  setDefault?: boolean;
}): Promise<EnvironmentCommand[]> {
  const response = await protectedJSON<{ items: EnvironmentCommand[] }>(
    appURL("/api/environment/commands"),
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        root: input.rootId,
        name: input.name,
        icon: input.icon,
        command: input.command,
        set_default: input.setDefault === true,
      }),
    },
  );
  return response.items || [];
}

export async function runEnvironmentCommand(input: {
  rootId: string;
  commandId: string;
  makeDefault?: boolean;
}): Promise<EnvironmentCommandRunResult> {
  return protectedJSON<EnvironmentCommandRunResult>(
    appURL("/api/environment/commands/run"),
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        root: input.rootId,
        command_id: input.commandId,
        make_default: input.makeDefault === true,
      }),
    },
  );
}

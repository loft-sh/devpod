import { TProviderID, TWorkspaceID } from "./types"

export const QueryKeys = {
  PLATFORM: ["platform"],
  ARCHITECTURE: ["architecture"],
  WORKSPACES: ["workspaces"],
  PROVIDERS: ["providers"],
  IDES: ["ides"],
  COMMUNITY_CONTRIBUTIONS: ["communityContributions"],
  CONTEXT_OPTIONS: ["contextOptions"],
  workspace(id: TWorkspaceID): string[] {
    return [...QueryKeys.WORKSPACES, id]
  },
  workspaceStatus(id: TWorkspaceID): string[] {
    return [...QueryKeys.WORKSPACES, id, "status"]
  },
  provider(id: TProviderID): string[] {
    return [...QueryKeys.PROVIDERS, id]
  },
  IS_CLI_INSTALLED: ["isCliInstalled"],
  providerOptions(id: TProviderID): string[] {
    return [...QueryKeys.provider(id), "options"]
  },
  providerUpdate(id: TProviderID): string[] {
    return [...QueryKeys.provider(id), "update"]
  },
}

export const MutationKeys = {
  CREATE_WORKSPACE: ["createWorkspace"],
  START_WORKSPACE: ["startWorkspace"],
  STOP_WORKSPACE: ["stopWorkspace"],
  REBUILD_WORKSPACE: ["rebuildWorkspace"],
  REMOVE_WORKSPACE: ["removeWorkspace"],
} as const

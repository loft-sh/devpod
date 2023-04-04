import { TProviderID, TWorkspaceID } from "./types"

export const QueryKeys = {
  PLATFORM: ["platform"],
  ARCHITECTURE: ["architecture"],
  WORKSPACES: ["workspaces"],
  PROVIDERS: ["providers"],
  IDES: ["ides"],
  workspace(id: TWorkspaceID): string[] {
    return [...QueryKeys.WORKSPACES, id]
  },
  workspaceStatus(id: TWorkspaceID): string[] {
    return [...QueryKeys.WORKSPACES, id, "status"]
  },
  provider(id: TProviderID): string[] {
    return [...QueryKeys.PROVIDERS, id]
  },
}

export const MutationKeys = {
  CREATE_WORKSPACE: ["createWorkspace"],
  START_WORKSPACE: ["startWorkspace"],
  STOP_WORKSPACE: ["stopWorkspace"],
  REBUILD_WORKSPACE: ["rebuildWorkspace"],
  REMOVE_WORKSPACE: ["removeWorkspace"],
} as const

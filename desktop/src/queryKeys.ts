import { TProviderID, TWorkspaceID } from "./types"

export const QueryKeys = {
  PLATFORM: ["platform"],
  ARCHITECTURE: ["architecture"],
  WORKSPACES: ["workspaces"],
  PROVIDERS: ["providers"],
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
  CREATE: ["create"],
  START: ["start"],
  STOP: ["stop"],
  REBUILD: ["rebuild"],
  REMOVE: ["remove"],
}

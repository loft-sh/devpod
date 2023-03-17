import { TWorkspaceID } from "./types"

export const QueryKeys = {
  PLATFORM: "platform",
  ARCHITECTURE: "architecture",
  WORKSPACES: "workspaces",
  PROVIDERS: "providers",
  workspace(id: TWorkspaceID): string[] {
    return [QueryKeys.WORKSPACES, id]
  },
  workspaceStatus(id: TWorkspaceID): string[] {
    return [QueryKeys.WORKSPACES, id, "status"]
  },
}

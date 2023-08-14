import { TProviderID, TWorkspaceID } from "./types"

export const QueryKeys = {
  PLATFORM: ["platform"],
  ARCHITECTURE: ["architecture"],
  WORKSPACES: ["workspaces"],
  PROVIDERS: ["providers"],
  IDES: ["ides"],
  COMMUNITY_CONTRIBUTIONS: ["communityContributions"],
  CONTEXT_OPTIONS: ["contextOptions"],
  RELEASES: ["releases"],
  APP_VERSION: ["appVersion"],
  UPDATE_RELEASE: ["updateRelease"],
  PENDING_UPDATE: ["pendingUpdate"],
  INSTALL_UPDATE: ["installUpdate"],
  PRO_INSTANCES: ["proInstances"],
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
  providerSetOptions(id: TProviderID): string[] {
    return [...QueryKeys.provider(id), "set-options"]
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

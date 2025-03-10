import { TProInstances, TProviderID, TWorkspaceID } from "./types"

export const QueryKeys = {
  PLATFORM: ["platform"],
  ARCHITECTURE: ["architecture"],
  SYSTEM_THEME: ["systemTheme"],
  WORKSPACES: ["workspaces"],
  PROVIDERS: ["providers"],
  PROVIDERS_CHECK_UPDATE_ALL: ["providers", "update", "all"],
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
  proWorkspaceTemplates(host: string, project: string): string[] {
    return ["workspaceTemplates", host, project]
  },
  proClusters(host: string, project: string): string[] {
    return ["clusters", host, project]
  },
  connectionStatus(host: string): string[] {
    return ["connectionStatus", host]
  },
  versionInfo(host: string): string[] {
    return ["versionInfo", host]
  },
  proProviderUpdates(proInstances: TProInstances | undefined) {
    return ["check-pro-provider-updates", proInstances]
  },
  userProfile(name: string | undefined) {
    return ["user-profile", name]
  },
}

export const MutationKeys = {
  CREATE_WORKSPACE: ["createWorkspace"],
  START_WORKSPACE: ["startWorkspace"],
  STOP_WORKSPACE: ["stopWorkspace"],
  REBUILD_WORKSPACE: ["rebuildWorkspace"],
  REMOVE_WORKSPACE: ["removeWorkspace"],
} as const

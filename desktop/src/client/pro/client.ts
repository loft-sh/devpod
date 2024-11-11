import { ProWorkspaceInstance } from "@/contexts"
import { ManagementV1DevPodWorkspaceInstance } from "@loft-enterprise/client/gen/models/managementV1DevPodWorkspaceInstance"
import { ManagementV1Project } from "@loft-enterprise/client/gen/models/managementV1Project"
import { ManagementV1ProjectClusters } from "@loft-enterprise/client/gen/models/managementV1ProjectClusters"
import { ManagementV1ProjectTemplates } from "@loft-enterprise/client/gen/models/managementV1ProjectTemplates"
import { ManagementV1Self } from "@loft-enterprise/client/gen/models/managementV1Self"
import { ErrorTypeCancelled, Result, ResultError } from "../../lib"
import { TImportWorkspaceConfig, TListProInstancesConfig, TProID, TProInstance } from "../../types"
import { TDebuggable, TStreamEventListenerFn } from "../types"
import { ProCommands } from "./proCommands"
import { Failed } from "@loft-enterprise/client"

export class ProClient implements TDebuggable {
  constructor(private readonly id: string) {}

  public setDebug(isEnabled: boolean): void {
    ProCommands.DEBUG = isEnabled
  }

  public async login(
    host: string,
    accessKey?: string,
    listener?: TStreamEventListenerFn
  ): Promise<ResultError> {
    return ProCommands.Login(host, accessKey, listener)
  }

  public async checkHealth() {
    return ProCommands.CheckHealth(this.id)
  }

  public async getVersion() {
    return ProCommands.GetVersion(this.id)
  }

  public async checkUpdate() {
    return ProCommands.CheckUpdate(this.id)
  }

  public async update(version: string) {
    return ProCommands.Update(this.id, version)
  }

  public async listProInstances(
    config?: TListProInstancesConfig
  ): Promise<Result<readonly TProInstance[]>> {
    return ProCommands.ListProInstances(config)
  }

  public async removeProInstance(id: TProID) {
    return ProCommands.RemoveProInstance(id)
  }

  public async importWorkspace(config: TImportWorkspaceConfig): Promise<ResultError> {
    return ProCommands.ImportWorkspace(config)
  }

  public watchWorkspaces(
    projectName: string,
    listener: (newWorkspaces: readonly ProWorkspaceInstance[]) => void,
    errorListener?: (failed: Failed) => void
  ) {
    const cmd = ProCommands.WatchWorkspaces(this.id, projectName)

    // kick off stream in the background
    cmd
      .stream(
        (event) => {
          if (event.type === "data") {
            const rawInstances =
              event.data as unknown as readonly ManagementV1DevPodWorkspaceInstance[]
            const workspaceInstances = rawInstances.map(
              (instance) => new ProWorkspaceInstance(instance)
            )
            listener(workspaceInstances)

            return
          }
        },
        { ignoreStderrError: true }
      )
      .then((res) => {
        if (res.err && res.val.type !== ErrorTypeCancelled) {
          errorListener?.(res.val)
        }
      })

    // Don't await here, we want to return the unsubscribe function
    return () => {
      cmd.cancel()
    }
  }

  public async listProjects(): Promise<Result<readonly ManagementV1Project[]>> {
    return ProCommands.ListProjects(this.id)
  }

  public async getSelf(): Promise<Result<ManagementV1Self>> {
    return ProCommands.GetSelf(this.id)
  }

  public async getProjectTemplates(
    projectName: string
  ): Promise<Result<ManagementV1ProjectTemplates>> {
    return ProCommands.ListTemplates(this.id, projectName)
  }

  public async getProjectClusters(
    projectName: string
  ): Promise<Result<ManagementV1ProjectClusters>> {
    return ProCommands.ListClusters(this.id, projectName)
  }

  public async createWorkspace(
    instance: ManagementV1DevPodWorkspaceInstance
  ): Promise<Result<ManagementV1DevPodWorkspaceInstance>> {
    return ProCommands.CreateWorkspace(this.id, instance)
  }

  public async updateWorkspace(
    instance: ManagementV1DevPodWorkspaceInstance
  ): Promise<Result<ManagementV1DevPodWorkspaceInstance>> {
    return ProCommands.UpdateWorkspace(this.id, instance)
  }
}

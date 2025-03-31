import { TWorkspaceOwnerFilterState } from "@/components"
import { ProWorkspaceInstance } from "@/contexts"
import { DaemonStatus } from "@/gen"
import { ManagementV1DevPodWorkspaceInstance } from "@loft-enterprise/client/gen/models/managementV1DevPodWorkspaceInstance"
import { ManagementV1Project } from "@loft-enterprise/client/gen/models/managementV1Project"
import { ManagementV1ProjectClusters } from "@loft-enterprise/client/gen/models/managementV1ProjectClusters"
import { ManagementV1ProjectTemplates } from "@loft-enterprise/client/gen/models/managementV1ProjectTemplates"
import { ManagementV1Self } from "@loft-enterprise/client/gen/models/managementV1Self"
import { ManagementV1UserProfile } from "@loft-enterprise/client/gen/models/managementV1UserProfile"
import { Result, ResultError, Return, isError, sleep } from "../../lib"
import {
  TGitCredentialHelperData,
  TImportWorkspaceConfig,
  TListProInstancesConfig,
  TPlatformHealthCheck,
  TPlatformVersionInfo,
  TProID,
  TProInstance,
} from "../../types"
import { TAURI_SERVER_URL } from "../tauriClient"
import { TDebuggable, TStreamEventListenerFn } from "../types"
import { ProCommands } from "./proCommands"
import { client as globalClient } from "@/client"

export class ProClient implements TDebuggable {
  constructor(protected readonly id: string) {}

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

  public async checkHealth(): Promise<Result<TPlatformHealthCheck>> {
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

  public watchWorkspacesProxy(
    projectName: string,
    _ownerFilter: TWorkspaceOwnerFilterState,
    listener: (newWorkspaces: readonly ProWorkspaceInstance[]) => void
  ) {
    const cmd = ProCommands.WatchWorkspaces(this.id, projectName)

    // kick off stream in the background
    cmd.stream(
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

    // Don't await here, we want to return the unsubscribe function
    return () => {
      // Still, return the promise so someone can choose to await if necessary.
      return cmd.cancel()
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

export class DaemonClient extends ProClient {
  constructor(id: string) {
    super(id)
  }

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

  private handleError<T>(err: unknown, fallbackMsg: string): Result<T> {
    if (isError(err)) {
      return Return.Failed(err.message)
    }

    if (typeof err === "string") {
      return Return.Failed(`${fallbackMsg}: ${err}`)
    }

    return Return.Failed(fallbackMsg)
  }

  private async getProxy<T>(path: string): Promise<Result<T>> {
    try {
      const res = await fetch(`${TAURI_SERVER_URL}/daemon-proxy/${this.id}${path}`, {
        method: "GET",
        headers: {
          "content-type": "application/json",
        },
      })
      if (!res.ok) {
        const maybeText = await res.text()

        let errMessage = `Get resource: ${res.statusText}.`
        if (maybeText) {
          errMessage += maybeText
        }

        return Return.Failed(errMessage)
      }
      const json: T = await res.json()

      return Return.Value(json)
    } catch (e) {
      return this.handleError(e, "unable to get resource")
    }
  }

  private async get<T>(path: string): Promise<Result<T>> {
    try {
      const res = await fetch(`${TAURI_SERVER_URL}${path}`, {
        method: "GET",
        headers: {
          "content-type": "application/json",
        },
      })
      if (!res.ok) {
        return Return.Failed(`Get resource: ${res.statusText}`)
      }

      const json: T = await res.json().catch(() => "")
      return Return.Value(json)
    } catch (e) {
      return this.handleError(e, "unable to get resource")
    }
  }

  private async post<T>(path: string, body: BodyInit): Promise<Result<T>> {
    try {
      const res = await fetch(`${TAURI_SERVER_URL}/daemon-proxy/${this.id}${path}`, {
        method: "POST",
        headers: {
          "content-type": "application/json",
        },
        body,
      })
      if (!res.ok) {
        return Return.Failed(`Error getting resource ${path} : ${res.statusText}`)
      }
      const json: T = await res.json()

      return Return.Value(json)
    } catch (e) {
      return this.handleError(e, "unable to get resource")
    }
  }

  public async restartDaemon() {
    return this.get(`/daemon/${this.id}/restart`)
  }

  public async checkHealth(): Promise<Result<TPlatformHealthCheck>> {
    // NOTE: We don't access this through the proxy because there might be issues during daemon startup
    // that we couldn't surface otherwise
    const res = await this.get<DaemonStatus>(`/daemon/${this.id}/status`)
    if (!res.ok) {
      return res
    }
    const status = res.val
    let healthy = status.state === "running"

    const details = []
    if (status.loginRequired) {
      healthy = false
      details.push("Login required to connect to platform")
    }
    if (status.state === "pending") {
      details.push("Daemon is starting up")
    }

    // if the backend state is running but we are not online, something is wrong with networking
    if (!status.online) {
      healthy = false
      details.push("Platform is offline")
    }

    return Return.Value({ healthy, details, loginRequired: status.loginRequired, online: status.online })
  }

  public watchWorkspaces(
    projectName: string,
    ownerFilter: TWorkspaceOwnerFilterState,
    listener: TWorksaceListener
  ): () => void {
    const watcher = new WorkspaceWatcher(this.id, projectName, ownerFilter, listener)

    return watcher.watch()
  }

  public async getSelf(): Promise<Result<ManagementV1Self>> {
    return this.getProxy("/self")
  }

  public async getUserProfile(): Promise<Result<ManagementV1UserProfile>> {
    return this.getProxy("/user-profile")
  }

  public async updateUserProfile(
    userProfile: ManagementV1UserProfile
  ): Promise<Result<ManagementV1UserProfile>> {
    try {
      const body = JSON.stringify(userProfile)
      const res = (await this.post("/update-user-profile", body)) as Result<ManagementV1UserProfile>

      return res
    } catch (e) {
      return this.handleError(e, "failed to update workspace")
    }
  }

  public async listProjects(): Promise<Result<readonly ManagementV1Project[]>> {
    return this.getProxy("/projects")
  }

  public async getVersion() {
    return this.getProxy<TPlatformVersionInfo>("/version")
  }

  public async getProjectTemplates(
    projectName: string
  ): Promise<Result<ManagementV1ProjectTemplates>> {
    return this.getProxy(`/projects/${projectName}/templates`)
  }

  public async getProjectClusters(
    projectName: string
  ): Promise<Result<ManagementV1ProjectClusters>> {
    return this.getProxy(`/projects/${projectName}/clusters`)
  }

  public async createWorkspace(
    instance: ManagementV1DevPodWorkspaceInstance
  ): Promise<Result<ManagementV1DevPodWorkspaceInstance>> {
    try {
      const body = JSON.stringify(instance)

      return this.post("/create-workspace", body)
    } catch (e) {
      return this.handleError(e, "failed to create workspace")
    }
  }

  public async updateWorkspace(
    instance: ManagementV1DevPodWorkspaceInstance
  ): Promise<Result<ManagementV1DevPodWorkspaceInstance>> {
    try {
      const body = JSON.stringify(instance)

      return this.post("/update-workspace", body)
    } catch (e) {
      return this.handleError(e, "failed to update workspace")
    }
  }

  public async queryGitCredentialsHelper(
    host: string
  ): Promise<Result<TGitCredentialHelperData | undefined>> {
    const searchParams = new URLSearchParams([["host", host]])

    return this.getProxy("/git-credentials?" + searchParams.toString())
  }

  public async checkUpdate() {
    return Return.Failed("provider is built-in, update is not supported")
  }

  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  public async update(_version: string) {
    return Return.Failed("provider is built-in, update is not supported")
  }
}
type TWorksaceListener = (newWorkspaces: readonly ProWorkspaceInstance[]) => void

class WorkspaceWatcher {
  private abortController = new AbortController()
  private reader: ReadableStreamDefaultReader | undefined
  private buffer: string = ""

  constructor(
    private readonly hostID: string,
    private readonly projectName: string,
    private readonly ownerFilter: TWorkspaceOwnerFilterState,
    private readonly listener: TWorksaceListener
  ) {}

  public cancel() {
    try {
      this.abortController.abort("watcher cancelled")
      this.reader?.cancel().catch((err) => {
        console.debug("cancel failed", err)
      })
    } catch(err) {
      console.error(err)
    }
    this.reader = undefined
    this.buffer = ""
  }

  public watch(): () => void {
    try {
      const url = new URL(`${TAURI_SERVER_URL}/daemon-proxy/${this.hostID}/watch-workspaces`)
      url.searchParams.set("project", this.projectName)
      url.searchParams.set("owner", this.ownerFilter)

      // start long-lived request. This should never stop unless cancelled through abortController
      fetch(url, {
        method: "GET",
        headers: { "content-type": "application/json" },
        keepalive: true,
        signal: this.abortController.signal,
      })
        .then((res) => {
          this.reader = res.body?.getReader()

          return this.read()
        })
        .catch((err) => {
          globalClient.log("info", `[${this.hostID}] watch workspaces error: ${err}`)
        })
        .finally(async () => {
          if (!this.abortController.signal.aborted && !(await this.reader?.closed)) {
            // Either the webview or the daemon terminated the watcher, try to reconnect
            console.info("reconnect")
            this.reader = undefined
            this.buffer = ""
            this.watch()
          }
          
          // Otherwise caller is responsible for reestablishing connection
        })
      return this.cancel.bind(this)
    } catch {
      return this.cancel.bind(this)
    }
  }

  private async read(): Promise<unknown> {
    const decoder = new TextDecoder()

    try {
      if (!this.reader) {
        return
      }

      const { done, value } = await this.reader.read()
      if (done) {
        return
      }
      this.buffer += decoder.decode(value, { stream: true })
      // NOTE: This relies on sender to end every message with a newline character. Make sure you also update the daemon server if you change this!
      const lines = this.buffer.split("\n")
      // Keep the last partial line in the buffer
      const maybeLine = lines.pop()
      if (maybeLine !== undefined) {
        this.buffer = maybeLine
      }

      lines.forEach((line) => {
        if (line.trim()) {
          try {
            const rawInstances: readonly ManagementV1DevPodWorkspaceInstance[] = JSON.parse(line)
            const workspaceInstances = rawInstances.map(
              (instance) => new ProWorkspaceInstance(instance)
            )
            this.listener(workspaceInstances)
          } catch (err) {
            const res = this.handleError(err, "failed to parse workspaces")
            if (res.err) {
              return err
            }
          }
        }
      })

      // Continue reading
      this.read()
    } catch (err) {
      return err
    }
  }

  private handleError<T>(err: unknown, fallbackMsg: string): Result<T> {
    if (isError(err)) {
      return Return.Failed(err.message)
    }

    if (typeof err === "string") {
      return Return.Failed(`${fallbackMsg}: ${err}`)
    }

    return Return.Failed(fallbackMsg)
  }
}

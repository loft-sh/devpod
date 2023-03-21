import { ChildProcess, Command, EventEmitter } from "@tauri-apps/api/shell"
import { Debug, exists } from "../lib"
import {
  TLogOutput,
  TWorkspace,
  TWorkspaceID,
  TWorkspaceStartConfig,
  TWorkspaceStatusResult,
  TWorkspaceWithoutStatus,
} from "../types"
import {
  DEFAULT_STATIC_COMMAND_CONFIG,
  DEVPOD_BINARY,
  DEVPOD_COMMAND_BUILD,
  DEVPOD_COMMAND_DELETE,
  DEVPOD_COMMAND_LIST,
  DEVPOD_COMMAND_STATUS,
  DEVPOD_COMMAND_STOP,
  DEVPOD_COMMAND_UP,
  DEVPOD_FLAG_DEBUG,
  DEVPOD_FLAG_FORCE_BUILD,
  DEVPOD_FLAG_ID,
  DEVPOD_FLAG_IDE,
  DEVPOD_FLAG_JSON_LOG_OUTPUT,
  DEVPOD_FLAG_JSON_OUTPUT,
  DEVPOD_FLAG_RECREATE,
} from "./constants"

type TGetResultFromConfig<T> = T extends TCommandConfig<infer U, TCommandStaticConfig> ? U : never
export type TCommand<
  TStaticConfig extends TCommandStaticConfig,
  TConfig extends TCommandConfig<unknown, TStaticConfig>
> = Readonly<
  TStaticConfig["streamResponse"] extends true
    ? {
        run(): Promise<TGetResultFromConfig<TConfig>>
        stream: TStreamCommandFn
      }
    : {
        run(): Promise<TGetResultFromConfig<TConfig>>
        stream?: never
      }
>
export type TStreamEvent = Readonly<
  { type: "data"; data: TLogOutput } | { type: "error"; error: TLogOutput }
>
export type TStreamEventListenerFn = (event: TStreamEvent) => void
export type TStreamCommandFn = (eventListener: TStreamEventListenerFn) => Promise<void>

type TEventListener<TEventName extends string> = Parameters<
  EventEmitter<TEventName>["addListener"]
>[1]

type TCommandRequiredConfig<TResult> = Readonly<{
  args(): string[]
  process(rawResult: ChildProcess): TResult
}>
type TDoStreamResponse = { streamResponse: true }
type TNoStreamResponse = { streamResponse: false }
type TCommandStaticConfig = Readonly<{ debug: boolean } & (TDoStreamResponse | TNoStreamResponse)>
type TDefaultStaticConfig = typeof DEFAULT_STATIC_COMMAND_CONFIG
type TCommandConfig<
  TResult,
  TStaticConfig extends TCommandStaticConfig = TDefaultStaticConfig
> = TCommandRequiredConfig<TResult> & TStaticConfig

export async function createCommand<
  TResult extends unknown,
  TStaticConfig extends TCommandStaticConfig
>(config: TCommandConfig<TResult, TStaticConfig>): Promise<TCommand<TStaticConfig, typeof config>> {
  const args = [...config.args(), ...(config.debug ? [DEVPOD_FLAG_DEBUG] : [])]

  if (await Debug.get?.("logs")) {
    console.info("Creating Devpod command with args: ", args)
  }

  const sidecarCommand = Command.sidecar(DEVPOD_BINARY, args)

  const run = async function run(): Promise<TResult> {
    const rawResult = await sidecarCommand.execute()

    if (await Debug.get?.("logs")) {
      console.info(`Result for command with args ${args}:`, rawResult)
    }

    return config.process(rawResult)
  }

  const stream: TCommand<TStaticConfig, typeof config>["stream"] = async function stream(listener) {
    if (!exists(listener)) {
      await sidecarCommand.execute()

      return Promise.resolve()
    }

    await sidecarCommand.spawn()

    return new Promise((res, rej) => {
      const stdoutListener: TEventListener<"data"> = (message) => {
        try {
          // TODO: TYPECHECK
          listener({ type: "data", data: JSON.parse(message) })
        } catch (error) {
          console.error("Failed to parse stdout message ", message, error)
        }
      }
      const stderrListener: TEventListener<"data"> = (message) => {
        try {
          // TODO: TYPECHECK
          listener({ type: "error", error: JSON.parse(message) })
        } catch (error) {
          console.error("Failed to parse stderr message ", message, error)
        }
      }

      sidecarCommand.stderr.addListener("data", stderrListener)
      sidecarCommand.stdout.addListener("data", stdoutListener)

      const cleanup = () => {
        sidecarCommand.stderr.removeListener("data", stderrListener)
        sidecarCommand.stdout.removeListener("data", stdoutListener)
      }

      sidecarCommand.on("close", () => {
        cleanup()
        res()
      })

      sidecarCommand.on("error", (arg) => {
        cleanup()
        rej(arg)
      })
    })
  }

  return {
    run,
    ...(config.streamResponse ? { stream } : {}),
  } as TCommand<TStaticConfig, typeof config> // TS has problems inferring the runtime check based `stream` method, so we need to cast here,
}

function createBaseConfig<TResult, TStaticConfig extends TCommandStaticConfig>(
  configImpl: TCommandRequiredConfig<TResult>,
  staticConfig?: TStaticConfig
): TCommandConfig<TResult, TStaticConfig> {
  const resolvedStaticConfig = {
    ...DEFAULT_STATIC_COMMAND_CONFIG,
    ...staticConfig,
  } as TStaticConfig

  return {
    ...resolvedStaticConfig,
    ...configImpl,
  }
}

export function createWithDebug(
  getShouldDebug: () => boolean
): <TResult, TConfig extends TCommandConfig<TResult, TCommandStaticConfig>>(
  commandConfig: TConfig
) => TConfig {
  return (commandConfig) => ({ ...commandConfig, debug: getShouldDebug() })
}

function toFlagArg(flag: string, arg: string) {
  return [flag, arg].join("=")
}

//#region commands
type TRawWorkspaces = readonly (Omit<TWorkspace, "status" | "id"> &
  Readonly<{ id: string | null }>)[]
export function listWorkspacesCommandConfig(): TCommandConfig<readonly TWorkspaceWithoutStatus[]> {
  return createBaseConfig({
    args() {
      return [DEVPOD_COMMAND_LIST, DEVPOD_FLAG_JSON_OUTPUT]
    },
    process(rawResult) {
      const rawWorkspaces = JSON.parse(rawResult.stdout) as TRawWorkspaces

      return rawWorkspaces.filter((workspace): workspace is TWorkspaceWithoutStatus =>
        exists(workspace.id)
      )
    },
  })
}

export function workspaceStatusCommandConfig(
  id: TWorkspaceID
): TCommandConfig<Pick<TWorkspace, "id" | "status">> {
  return createBaseConfig({
    args() {
      return [DEVPOD_COMMAND_STATUS, id, DEVPOD_FLAG_JSON_OUTPUT]
    },
    process(rawResult) {
      if (rawResult.code !== 0) {
        throw Error(`Failed to get status for workspace ${id}`)
      }

      const { state } = JSON.parse(rawResult.stdout) as TWorkspaceStatusResult

      return { id, status: state }
    },
  })
}

export function startWorkspaceCommandConfig(id: TWorkspaceID, config: TWorkspaceStartConfig) {
  return createBaseConfig(
    {
      args() {
        // TODO: Implement other options

        const maybeSource = config.sourceConfig?.source
        const maybeIDFlag = exists(maybeSource) ? [toFlagArg(DEVPOD_FLAG_ID, id)] : []

        const maybeIdeName = config.ideConfig?.ide
        const maybeIDEFlag = exists(maybeIdeName) ? [toFlagArg(DEVPOD_FLAG_IDE, maybeIdeName)] : []

        const identifier = exists(maybeSource) && exists(maybeIDFlag) ? maybeSource : id

        return [
          DEVPOD_COMMAND_UP,
          identifier,
          ...maybeIDFlag,
          ...maybeIDEFlag,
          DEVPOD_FLAG_JSON_LOG_OUTPUT,
        ]
      },
      process() {
        // noop
      },
    },
    { streamResponse: true, debug: false } as const
  )
}

export function stopWorkspaceCommandConfig(id: TWorkspaceID): TCommandConfig<void> {
  return createBaseConfig({
    args() {
      return [DEVPOD_COMMAND_STOP, id, DEVPOD_FLAG_JSON_LOG_OUTPUT]
    },
    process(rawResult) {
      if (rawResult.code !== 0) {
        throw Error(`Failed to stop Workspace ${id}`)
      }
    },
  })
}

export function rebuildWorkspaceCommandConfig(id: TWorkspaceID): TCommandConfig<void> {
  return createBaseConfig({
    args() {
      return [
        DEVPOD_COMMAND_BUILD,
        id,
        DEVPOD_FLAG_JSON_LOG_OUTPUT,
        DEVPOD_FLAG_FORCE_BUILD,
        DEVPOD_FLAG_RECREATE,
      ]
    },
    process(rawResult) {
      if (rawResult.code !== 0) {
        throw Error(`Failed to rebuild Workspace ${id}`)
      }
    },
  })
}

export function removeWorkspaceCommandConfig(id: TWorkspaceID): TCommandConfig<void> {
  return createBaseConfig({
    args() {
      return [DEVPOD_COMMAND_DELETE, id]
    },
    process(rawResult) {
      if (rawResult.code !== 0) {
        throw Error(`Failed to delete Workspace ${id}`)
      }
    },
  })
}
//#endregion

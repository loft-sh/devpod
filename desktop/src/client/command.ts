import { Child, ChildProcess, EventEmitter, Command as ShellCommand } from "@tauri-apps/api/shell"
import { debug, isError, Result, ResultError, Return } from "../lib"
import { DEVPOD_BINARY, DEVPOD_FLAG_OPTION, DEVPOD_UI_ENV_VAR } from "./constants"
import { TStreamEvent } from "./types"

export type TStreamEventListenerFn = (event: TStreamEvent) => void
export type TEventListener<TEventName extends string> = Parameters<
  EventEmitter<TEventName>["addListener"]
>[1]

export type TCommand<T> = {
  run(): Promise<Result<T>>
  stream(listener: TStreamEventListenerFn): Promise<ResultError>
  cancel(): Promise<ResultError>
}

export class Command implements TCommand<ChildProcess> {
  private sidecarCommand: ShellCommand
  private childProcess?: Child
  private args: string[]

  public static ADDITIONAL_ENV_VARS: string = ""
  public static HTTP_PROXY: string = ""
  public static HTTPS_PROXY: string = ""
  public static NO_PROXY: string = ""

  constructor(args: string[]) {
    debug("commands", "Creating Devpod command with args: ", args)
    const extraEnvVars = Command.ADDITIONAL_ENV_VARS.split(",")
      .map((envVarStr) => envVarStr.split("="))
      .reduce(
        (acc, pair) => {
          const [key, value] = pair
          if (key === undefined || value === undefined) {
            return acc
          }

          return { ...acc, [key]: value }
        },
        {} as Record<string, string>
      )

    // set proxy related environment variables
    if (Command.HTTP_PROXY) {
      extraEnvVars["HTTP_PROXY"] = Command.HTTP_PROXY
    }
    if (Command.HTTPS_PROXY) {
      extraEnvVars["HTTPS_PROXY"] = Command.HTTPS_PROXY
    }
    if (Command.NO_PROXY) {
      extraEnvVars["NO_PROXY"] = Command.NO_PROXY
    }

    // allows the CLI to detect if commands have been invoked from the UI
    extraEnvVars[DEVPOD_UI_ENV_VAR] = "true"

    if (import.meta.env.TAURI_IS_FLATPAK === "true") {
      this.sidecarCommand = new ShellCommand("run-path-devpod-wrapper", args, {
        env: { ...extraEnvVars, ["FLATPAK_ID"]: "sh.loft.devpod" },
      })
    } else {
      this.sidecarCommand = ShellCommand.sidecar(DEVPOD_BINARY, args, { env: extraEnvVars })
    }
    this.args = args
  }

  public async run(): Promise<Result<ChildProcess>> {
    try {
      const rawResult = await this.sidecarCommand.execute()
      debug("commands", `Result for command with args ${this.args}:`, rawResult)

      return Return.Value(rawResult)
    } catch (e) {
      return Return.Failed(e + "")
    }
  }

  public async stream(listener: TStreamEventListenerFn): Promise<ResultError> {
    try {
      this.childProcess = await this.sidecarCommand.spawn()
      await new Promise((res, rej) => {
        const stdoutListener: TEventListener<"data"> = (message) => {
          try {
            const data = JSON.parse(message)

            // special case: the cli sends us a message where "done" is "true"
            // to signal the command is terminated and we should stop listen to it
            // This happens for the vscode browser command as it needs to stay open
            // for port-forwarding, but we don't care anymore about its output.
            if (data?.done === "true") {
              res(Return.Ok())
            } else {
              listener({ type: "data", data })
            }
          } catch (error) {
            console.error("Failed to parse stdout message ", message, error)
          }
        }
        const stderrListener: TEventListener<"data"> = (message) => {
          try {
            const error = JSON.parse(message)
            listener({ type: "error", error })
          } catch (error) {
            console.error("Failed to parse stderr message ", message, error)
          }
        }

        this.sidecarCommand.stderr.addListener("data", stderrListener)
        this.sidecarCommand.stdout.addListener("data", stdoutListener)

        const cleanup = () => {
          this.sidecarCommand.stderr.removeListener("data", stderrListener)
          this.sidecarCommand.stdout.removeListener("data", stdoutListener)
          this.childProcess = undefined
        }

        this.sidecarCommand.on("close", (arg?: { code: number }) => {
          cleanup()
          if (arg?.code !== 0) {
            rej(new Error("exit code: " + arg?.code))
          } else {
            res(Return.Ok())
          }
        })

        this.sidecarCommand.on("error", (arg) => {
          cleanup()
          rej(arg)
        })
      })

      return Return.Ok()
    } catch (e) {
      if (isError(e)) {
        return Return.Failed(e.message)
      }
      console.log(e)

      return Return.Failed("streaming failed")
    }
  }

  /**
   * Cancel the command.
   * Only works if it has been created with the `stream` method.
   */
  public async cancel(): Promise<Result<undefined>> {
    try {
      await this.childProcess?.kill()

      return Return.Ok()
    } catch (e) {
      if (isError(e)) {
        return Return.Failed(e.message)
      }

      return Return.Failed("failed to cancel command")
    }
  }
}

export function isOk(result: ChildProcess): boolean {
  return result.code === 0
}

export function toFlagArg(flag: string, arg: string) {
  return [flag, arg].join("=")
}

export function toMultipleFlagArg(input: string) {
  const equaledInput = input.replace(/([a-zA-Z])\s+([a-zA-Z])/g, "$1=$2")

  return equaledInput.split(" ")
}

export function serializeRawOptions(
  rawOptions: Record<string, unknown>,
  flag: string = DEVPOD_FLAG_OPTION
): string[] {
  return Object.entries(rawOptions).map(([key, value]) => flag + `=${key}=${value}`)
}

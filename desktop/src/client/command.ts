import { ChildProcess, Command as ShellCommand, EventEmitter } from "@tauri-apps/api/shell"
import { debug, isError } from "../lib"
import { Result, ResultError, Return } from "../lib/result"
import { DEVPOD_BINARY, DEVPOD_FLAG_OPTION, DEVPOD_UI_ENV_VAR } from "./constants"
import { TStreamEvent } from "./types"

export type TStreamEventListenerFn = (event: TStreamEvent) => void
export type TEventListener<TEventName extends string> = Parameters<
  EventEmitter<TEventName>["addListener"]
>[1]

export type TCommand<T> = {
  run(): Promise<Result<T>>
  stream(listener: TStreamEventListenerFn): Promise<ResultError>
}

export class Command implements TCommand<ChildProcess> {
  private sidecarCommand
  private args: string[]

  constructor(args: string[]) {
    debug("commands", "Creating Devpod command with args: ", args)
    this.sidecarCommand = ShellCommand.sidecar(DEVPOD_BINARY, args, {
      env: { [DEVPOD_UI_ENV_VAR]: "true" },
    })
    this.args = args
  }

  public withConversion<T>(convert: (childProcess: ChildProcess) => Result<T>): TCommand<T> {
    return {
      run: async () => {
        const result = await this.run()
        if (result.err) {
          return result
        }

        return convert(result.val)
      },
      stream: this.stream,
    }
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
      await this.sidecarCommand.spawn()
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

      return Return.Failed("streaming failed")
    }
  }
}

export function isOk(result: ChildProcess): boolean {
  return result.code === 0
}

export function toFlagArg(flag: string, arg: string) {
  return [flag, arg].join("=")
}

export function serializeRawOptions(rawOptions: Record<string, unknown>): string[] {
  if (!rawOptions) {
    return []
  }

  return Object.entries(rawOptions).map(([key, value]) => DEVPOD_FLAG_OPTION + `=${key}=${value}`)
}

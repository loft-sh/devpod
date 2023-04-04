import { ChildProcess, Command as ShellCommand, EventEmitter } from "@tauri-apps/api/shell"
import { debug, isError } from "../lib"
import { Result, ResultError, Return } from "../lib/result"
import { DEVPOD_BINARY } from "./constants"
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
    this.sidecarCommand = ShellCommand.sidecar(DEVPOD_BINARY, args)
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
            listener({ type: "data", data })
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

        this.sidecarCommand.on("close", () => {
          cleanup()
          res(Return.Ok())
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

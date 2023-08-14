import { TIDE, TLogOutput } from "../types"
import { ChildProcess } from "@tauri-apps/api/shell"
import { Err, Failed, Return } from "./result"
import { TActionObj } from "../contexts"

export function exists<T extends any | null | undefined>(
  arg: T
): arg is Exclude<T, null | undefined> {
  return arg !== undefined && arg !== null
}

export function isError(error: unknown): error is Error {
  return error instanceof Error
}

export function noop(): void {}
export async function noopAsync(): Promise<void> {}

// WARN: All keys and values of `map` need to be serializable by `JSON.stringify` for this to work!
// See the [MDN docs](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/JSON/stringify#description)
// if you're unsure
export function serializeMap<T extends Map<unknown, unknown>>(map: T): string {
  return JSON.stringify(Array.from(map.entries()))
}

export function deserializeMap<T extends Map<unknown, unknown>>(serializedMap: string): T {
  return new Map(JSON.parse(serializedMap)) as T
}

export function isEmpty<T extends { length: number }>(arg: T): boolean {
  return arg.length <= 0
}

export function safeJSONParse<T>(arg: string): T | null {
  try {
    return JSON.parse(arg) as T
  } catch {
    return null
  }
}

export function getErrorFromChildProcess(result: ChildProcess): Err<Failed> {
  const stdout = parseOutput(result.stdout)
  const stderr = parseOutput(result.stderr)
  const sorted = [...stdout, ...stderr].sort((a, b) => {
    if (a.time === b.time) {
      return 0
    }

    const aTime = new Date(a.time).getTime() || 0
    const bTime = new Date(b.time).getTime() || 0
    if (aTime < bTime) {
      return -1
    }

    return 1
  })

  const message: string[] = sorted.reduce((acc, log) => {
    const line = log.message?.trim()
    if (!line) {
      return acc
    }

    acc.push(line)

    return acc
  }, [] as string[])

  return Return.Failed(message.join("\n"))
}

export function parseOutput(arg: string): TLogOutput[] {
  const retOutput: TLogOutput[] = arg.split("\n").reduce((acc, line) => {
    const trimmed = line.trim()
    if (!trimmed) {
      return acc
    }

    const logLine = safeJSONParse(line) as TLogOutput | undefined
    if (!logLine?.message) {
      return acc
    }

    acc.push(logLine)

    return acc
  }, [] as TLogOutput[])

  return retOutput
}

export function getKeys<T extends object>(arg: T): readonly (keyof T)[] {
  return Object.keys(arg) as unknown as readonly (keyof T)[]
}

export function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

export function getActionDisplayName(action: TActionObj): string {
  if (action.name === "checkStatus") {
    return `check status ${action.targetID}`
  }

  return `${action.name} ${action.targetID}`
}

export function getIDEDisplayName(ide: TIDE) {
  // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
  return ide.displayName ?? ide.name ?? "Unknown"
}

export function randomString(length: number): string {
  return [...Array(length)].map(() => (~~(Math.random() * 36)).toString(36)).join("")
}

export function remToPx(rem: string): number {
  return parseFloat(rem) * parseFloat(getComputedStyle(document.documentElement).fontSize)
}

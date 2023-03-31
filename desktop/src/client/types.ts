import { TLogOutput } from "../types"

export type TDebuggable = Readonly<{ setDebug(isEnabled: boolean): void }>
export type TStreamEvent = Readonly<
  | { type: "data"; data: TLogOutput; rawData: string }
  | { type: "error"; error: TLogOutput; rawData: string }
>
export type TStreamEventListenerFn = (event: TStreamEvent) => void

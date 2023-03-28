import { TLogOutput } from "../types"

export type TDebuggable = Readonly<{ setDebug(isEnabled: boolean): void }>
export type TStreamEvent = Readonly<
  { type: "data"; data: TLogOutput } | { type: "error"; error: TLogOutput }
>
export type TStreamEventListenerFn = (event: TStreamEvent) => void

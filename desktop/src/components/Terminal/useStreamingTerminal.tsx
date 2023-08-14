import dayjs from "dayjs"
import { Theme, useToken } from "@chakra-ui/react"
import { useCallback, useMemo, useRef } from "react"
import { TStreamEventListenerFn } from "../../client"
import { TLogOutput } from "../../types"
import { Terminal, TTerminal } from "./Terminal"

export function useStreamingTerminal({
  fontSize,
}: { fontSize?: keyof Theme["fontSizes"] } | undefined = {}) {
  const terminalRef = useRef<TTerminal | null>(null)
  const fontSizeToken = useToken(
    "fontSizes",
    useMemo(() => fontSize ?? "md", [fontSize])
  )
  const terminal = useMemo(
    () => <Terminal ref={terminalRef} fontSize={fontSizeToken} />,
    [fontSizeToken]
  )
  const connectStream = useCallback<TStreamEventListenerFn>(
    (event) => {
      switch (event.type) {
        case "data":
          if (event.data.message === undefined) {
            return
          }
          terminalRef.current?.writeln(formatLine(event.data))
          break
        case "error":
          if (event.error.message === undefined) {
            return
          }
          terminalRef.current?.writeln(formatLine(event.error))
          break
      }
    },
    [terminalRef]
  )

  const clear = useCallback(() => {
    terminalRef.current?.clear()
  }, [terminalRef])

  return { terminal, connectStream, clear }
}

const ANSI_COLOR = {
  Reset: "0",
  White: "97",
  BrightCyan: "96",
  BrightMagenta: "95",
  BrightBlue: "94",
  BrightYellow: "93",
  BrightGreen: "92",
  BrightRed: "91",
  BrightBlack: "90",

  DarkWhite: "37",
  DarkCyan: "36",
  DarkMagenta: "35",
  DarkBlue: "34",
  DarkYellow: "33",
  DarkGreen: "32",
  DarkRed: "31",
  Black: "30",
}
const ANSI_TEXT = {
  Bold: "1",
  Underline: "4",
  NoUnderline: "24",
  Reverse: "7",
  NoReverse: "27",
}

const LOG_COLORS = {
  panic: ANSI_COLOR.DarkMagenta,
  fatal: ANSI_COLOR.DarkRed,
  error: ANSI_COLOR.BrightRed,
  warn: ANSI_COLOR.DarkYellow,
  info: ANSI_COLOR.BrightBlue,
  debug: ANSI_COLOR.BrightGreen,
}

function formatLine({ level, message, time }: TLogOutput) {
  let levelColor = ANSI_COLOR.White
  if (level in LOG_COLORS) {
    levelColor = LOG_COLORS[level as keyof typeof LOG_COLORS]
  }

  const date = `\x1b[${ANSI_COLOR.DarkWhite}m[${dayjs(time).format("HH:mm:ss")}]`
  const prefix = `\x1b[${ANSI_TEXT.Bold};${levelColor}m${level}`
  const data = `\x1b[${ANSI_COLOR.Reset}m${message}`

  return `${date} ${prefix} ${data}`
}

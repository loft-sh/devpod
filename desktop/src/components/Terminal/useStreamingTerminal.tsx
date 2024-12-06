import dayjs from "dayjs"
import { Theme, useToken } from "@chakra-ui/react"
import React, { useCallback, useMemo, useRef } from "react"
import { TStreamEventListenerFn } from "@/client"
import { TLogOutput } from "@/types"
import { Terminal, TTerminal } from "./Terminal"
import { TSearchOptions, useTerminalSearch } from "@/components/Terminal/useTerminalSearch"

export function useStreamingTerminal({
  fontSize,
  borderRadius,
  searchOptions,
}:
  | {
      fontSize?: keyof Theme["fontSizes"]
      borderRadius?: keyof Theme["radii"]
      searchOptions?: TSearchOptions
    }
  | undefined = {}) {
  const terminalRef = useRef<TTerminal | null>(null)

  const {
    internals: { searchStateRef, debounceSearchResults, resetSearch, onResize },
    searchApi,
  } = useTerminalSearch(terminalRef, searchOptions)

  const fontSizeToken = useToken(
    "fontSizes",
    useMemo(() => fontSize ?? "md", [fontSize])
  )

  const borderRadiusToken = useToken(
    "radii",
    useMemo(() => borderRadius ?? "md", [borderRadius])
  )

  const terminal = useMemo(
    () => (
      <Terminal
        ref={terminalRef}
        fontSize={fontSizeToken}
        borderRadius={borderRadiusToken}
        onResize={onResize}
      />
    ),
    [fontSizeToken, onResize, borderRadiusToken]
  )

  const connectStream = useCallback<TStreamEventListenerFn>(
    (event) => {
      switch (event.type) {
        case "data": {
          if (event.data.message === undefined) {
            return
          }

          const formattedLine = formatLine(event.data)
          terminalRef.current?.writeln(formattedLine.ansi)

          searchStateRef.current.preWrappedLines = undefined
          searchStateRef.current.searchableLines.push(...processInputLine(formattedLine.plain))

          debounceSearchResults(searchStateRef.current.searchOptions)

          break
        }
        case "error": {
          if (event.error.message === undefined) {
            return
          }

          const formattedLine = formatLine(event.error)
          terminalRef.current?.writeln(formattedLine.ansi)

          searchStateRef.current.preWrappedLines = undefined
          searchStateRef.current.searchableLines.push(...processInputLine(formattedLine.plain))

          debounceSearchResults(searchStateRef.current.searchOptions)

          break
        }
      }
    },
    [terminalRef, searchStateRef, debounceSearchResults]
  )

  const clear = useCallback(() => {
    resetSearch()
    terminalRef.current?.clear()
  }, [terminalRef, resetSearch])

  return {
    terminal,
    connectStream,
    clear,
    search: searchApi,
  }
}

function processInputLine(line: string) {
  // Default tabStopWidth is 8.
  const withoutTabs = line.replaceAll(/\t/g, " ".repeat(8))

  const subLines = withoutTabs.split(/\r\n|\n/)

  return subLines.map((sl) => {
    const splitByCR = sl.split("\r")

    return splitByCR[splitByCR.length - 1]!
  })
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

  const formattedTime = dayjs(time).format("HH:mm:ss")

  const date = `\x1b[${ANSI_COLOR.DarkWhite}m[${formattedTime}]`
  const prefix = `\x1b[${ANSI_TEXT.Bold};${levelColor}m${level}`
  const data = `\x1b[${ANSI_COLOR.Reset}m${message}`

  return { ansi: `${date} ${prefix} ${data}`, plain: `[${formattedTime}] ${level} ${message}` }
}

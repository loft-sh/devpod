import { useCallback, useMemo, useRef } from "react"
import { TStreamEventListenerFn } from "../../client"
import { Terminal, TTerminal } from "./Terminal"

export function useStreamingTerminal() {
  const terminalRef = useRef<TTerminal | null>(null)
  const terminal = useMemo(() => <Terminal ref={terminalRef} />, [])

  const connectStream = useCallback<TStreamEventListenerFn>(
    (event) => {
      // TODO: Message color
      switch (event.type) {
        case "data":
          terminalRef.current?.writeln(event.data.message)
          break
        case "error":
          terminalRef.current?.writeln(event.error.message)
          break
      }
    },
    [terminalRef]
  )

  return { terminal, connectStream }
}

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
          if (event.data.message === undefined) {
            return
          }
          terminalRef.current?.writeln(event.data.message)
          break
        case "error":
          if (event.error.message === undefined) {
            return
          }
          terminalRef.current?.writeln(event.error.message)
          break
      }
    },
    [terminalRef]
  )

  const clear = useCallback(() => {
    terminalRef.current?.clear()
  }, [])

  return { terminal, connectStream, clear }
}

import { useCallback } from "react"
import { TStreamEventHandlerFn } from "../../client"
import { useTerminal } from "./useTerminal"

export function useStreamingTerminal() {
  const { terminal, terminalRef } = useTerminal()

  const connectStream = useCallback<TStreamEventHandlerFn>(
    (event) => {
      switch (event.type) {
        case "data":
          terminalRef.current?.writeln(event.data.message)
          break
        case "error":
          // TODO: highlight stderr messages
          terminalRef.current?.writeln(event.error.message)
          break
      }
    },
    [terminalRef]
  )

  return { terminal, connectStream }
}

import { useMemo, useRef } from "react"
import { Terminal, TTerminal } from "./Terminal"

export function useTerminal() {
  const terminalRef = useRef<TTerminal | null>(null)
  const terminal = useMemo(() => <Terminal ref={terminalRef} />, [])

  return { terminal, terminalRef }
}

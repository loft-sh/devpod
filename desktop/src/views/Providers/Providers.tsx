import { useProviders } from "../../contexts/DevPodContext/DevPodContext"
import { forwardRef, useEffect, useImperativeHandle, useLayoutEffect, useMemo, useRef } from "react"
import { exists } from "../../lib"
import {
  Accordion,
  AccordionButton,
  AccordionIcon,
  AccordionItem,
  AccordionPanel,
  Box,
  Code,
  Text,
} from "@chakra-ui/react"
import { Terminal } from "xterm"
import { FitAddon } from "xterm-addon-fit"

type TProviderRow = Readonly<{ name: string; options: string }>
export function Providers() {
  const { terminal, terminalHandle } = useTerminal()
  const [providers] = useProviders()
  const providerRows = useMemo<readonly TProviderRow[]>(() => {
    const maybeProviders = providers?.providers
    if (!exists(maybeProviders)) {
      return []
    }

    return Object.entries(maybeProviders).map(([name, details]) => {
      return { name, options: JSON.stringify(details, null, 2) }
    })
  }, [providers])

  useEffect(() => {
    terminalHandle?.writeln("Hello")
    terminalHandle?.writeln("Hello")
  }, [terminalHandle])

  return (
    <>
      {terminal}
      <div>Providers</div>
      <Accordion allowMultiple>
        {providerRows.map((row) => (
          <AccordionItem key={row.name}>
            <AccordionButton>
              <AccordionIcon />
              <Text>{row.name}</Text>
            </AccordionButton>
            <AccordionPanel>
              <Code padding={4} whiteSpace="pre" display="block" borderRadius="md">
                {row.options}
              </Code>
            </AccordionPanel>
          </AccordionItem>
        ))}
      </Accordion>
    </>
  )
}

function useTerminal() {
  const terminalRef = useRef<Terminal | null>(null)
  const terminal = useMemo(() => <T ref={terminalRef} />, [])

  return { terminal, terminalHandle: terminalRef.current }
}

type TTerminalRef = Readonly<{
  clear: VoidFunction
  write: (data: string) => void
  writeln: (data: string) => void
}>

const T = forwardRef<TTerminalRef, {}>(function T(_, ref) {
  const containerRef = useRef<HTMLDivElement>(null)
  const terminalRef = useRef<Terminal | null>(null)
  const termFitRef = useRef<FitAddon | null>(null)

  useImperativeHandle(ref, () => ({
    clear() {
      terminalRef.current?.clear()
    },
    write(data) {
      terminalRef.current?.write(data)
      termFitRef.current?.fit()
    },
    writeln(data) {
      terminalRef.current?.writeln(data)
      termFitRef.current?.fit()
    },
  }))

  useEffect(() => {
    const resizeHandler = () => {
      termFitRef.current?.fit()
    }
    window.addEventListener("resize", resizeHandler, true)

    return () => window.removeEventListener("resize", resizeHandler, true)
  }, [])

  useLayoutEffect(() => {
    if (!exists(terminalRef.current)) {
      const terminal = new Terminal({
        convertEol: true,
        fontSize: 12,
        scrollback: 25000,
        // cursorBlink: this.props.cursorBlink != null ? this.props.cursorBlink : false,
        // disableStdin: this.props.disableStdin != null ? this.props.disableStdin : true,
        theme: {
          background: "#263544",
          foreground: "#AFC6D2",
        },
      })
      terminalRef.current = terminal

      terminal.onKey((key) => {
        if (terminal.hasSelection() && key.domEvent.ctrlKey && key.domEvent.key === "c") {
          document.execCommand("copy")
        }
      })

      const termFit = new FitAddon()
      termFitRef.current = termFit
      terminal.loadAddon(termFit)

      terminal.open(containerRef.current!)
      termFit.fit()
    }
  }, [])

  return (
    <Box width="full" height="full">
      <div ref={containerRef} />
    </Box>
  )
})

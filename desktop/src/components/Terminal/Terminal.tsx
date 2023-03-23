import { Box, useColorModeValue, useToken } from "@chakra-ui/react"
import { css } from "@emotion/react"
import { forwardRef, useEffect, useImperativeHandle, useLayoutEffect, useMemo, useRef } from "react"
import { Terminal as XTermTerminal, ITheme as IXTermTheme } from "xterm"
import { FitAddon } from "xterm-addon-fit"
import { exists } from "../../lib"

type TTerminalRef = Readonly<{
  clear: VoidFunction
  write: (data: string) => void
  writeln: (data: string) => void
}>
export type TTerminal = TTerminalRef

export const Terminal = forwardRef<TTerminalRef, {}>(function T(_, ref) {
  const containerRef = useRef<HTMLDivElement>(null)
  const terminalRef = useRef<XTermTerminal | null>(null)
  const termFitRef = useRef<FitAddon | null>(null)

  const backgroundToken = useColorModeValue("gray.900", "gray.300")
  const backgroundColor = useToken("colors", backgroundToken)

  const textToken = useColorModeValue("gray.100", "gray.900")
  const textColor = useToken("colors", textToken)

  const scrollBarThumbToken = useColorModeValue("gray.500", "gray.200")
  const scrollBarThumbColor = useToken("colors", scrollBarThumbToken)

  const terminalTheme = useMemo<Partial<IXTermTheme>>(
    () => ({
      background: backgroundColor,
      foreground: textColor,
    }),
    [backgroundColor, textColor]
  )

  useLayoutEffect(() => {
    if (!exists(terminalRef.current)) {
      const terminal = new XTermTerminal({
        convertEol: true,
        scrollback: 25_000,
        theme: terminalTheme,
        // TODO: should be configurable via props
        cursorStyle: "underline",
        disableStdin: true,
        cursorBlink: false,
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

      // Clean up aaaall the things :)
      return () => {
        termFitRef.current?.dispose()
        termFitRef.current = null

        terminalRef.current?.dispose()
        terminalRef.current = null
      }
    }

    // Don't initialize more than once! Use imperative api to update terminal state
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  useEffect(() => {
    const resizeHandler = () => {
      try {
        termFitRef.current?.fit()
      } catch {
        /* ignore */
      }
    }
    window.addEventListener("resize", resizeHandler, true)

    return () => window.removeEventListener("resize", resizeHandler, true)
  }, [])

  useEffect(() => {
    let maybeTheme = terminalRef.current?.options.theme
    if (exists(maybeTheme)) {
      maybeTheme = terminalTheme
    }
  }, [terminalTheme])

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

  return (
    <Box width="full" height="full">
      <Box
        height="full"
        as="div"
        backgroundColor={terminalTheme.background}
        borderRadius="md"
        borderWidth={8}
        boxSizing="content-box" // needs to be set to accomodate for the way xterm measures it's container
        borderColor={terminalTheme.background}
        ref={containerRef}
        css={css`
          .xterm-viewport {
            &::-webkit-scrollbar-button {
              display: none;
              height: 13px;
              border-radius: 0px;
              background-color: transparent;
            }
            &::-webkit-scrollbar-button:hover {
              background-color: transparent;
            }
            &::-webkit-scrollbar-thumb {
              border-radius: 4px;
              background-color: ${scrollBarThumbColor};
            }
            &::-webkit-scrollbar-track {
              background-color: transparent;
            }
            &::-webkit-scrollbar-track:hover {
              background-color: transparent;
            }
            &::-webkit-scrollbar {
              width: 6px;
            }
          }
        `}
      />
    </Box>
  )
})

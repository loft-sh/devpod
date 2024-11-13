import { Box, useColorModeValue, useToken } from "@chakra-ui/react"
import { css } from "@emotion/react"
import { forwardRef, useEffect, useImperativeHandle, useLayoutEffect, useMemo, useRef } from "react"
import { Terminal as XTermTerminal, ITheme as IXTermTheme } from "xterm"
import { FitAddon } from "xterm-addon-fit"
import { exists, remToPx } from "../../lib"

type TTerminalRef = Readonly<{
  clear: VoidFunction
  write: (data: string) => void
  writeln: (data: string) => void
}>
type TTerminalProps = Readonly<{ fontSize: string }>
export type TTerminal = TTerminalRef

export const Terminal = forwardRef<TTerminalRef, TTerminalProps>(function T({ fontSize }, ref) {
  const containerRef = useRef<HTMLDivElement>(null)
  const terminalRef = useRef<XTermTerminal | null>(null)
  const termFitRef = useRef<FitAddon | null>(null)

  const backgroundColor = useToken("colors", "gray.900")
  const textColor = useToken("colors", "gray.100")

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
        cursorStyle: "underline",
        disableStdin: true,
        cursorBlink: false,
        fontSize: remToPx(fontSize),
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

      // Perform initial fit. Dimensions are only available after the terminal has been rendered once.
      const disposable = terminal.onRender(() => {
        if (termFit.proposeDimensions()) {
          termFit.fit()
          disposable.dispose()
        }
      })

      terminal.open(containerRef.current!)

      // Clean up aaaall the things :)
      return () => {
        disposable.dispose()

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

  useEffect(() => {
    // TODO: resize when global font size changes
    let maybeFontSize = terminalRef.current?.options.fontSize
    if (exists(maybeFontSize)) {
      maybeFontSize = remToPx(fontSize)
    }
  }, [fontSize])

  useImperativeHandle(
    ref,
    () => {
      return {
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
      }
    },
    [terminalRef]
  )

  return (
    <Box width="full" height="full">
      <Box
        height="full"
        as="div"
        backgroundColor={terminalTheme.background}
        borderRadius="md"
        borderWidth={8}
        boxSizing="content-box" // needs to be set to accommodate for the way xterm measures it's container
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

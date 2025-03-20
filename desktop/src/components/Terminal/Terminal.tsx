import { Box, useColorModeValue, useToken } from "@chakra-ui/react"
import { css } from "@emotion/react"
import React, {
  forwardRef,
  useEffect,
  useImperativeHandle,
  useLayoutEffect,
  useMemo,
  useRef,
} from "react"
import { ITerminalAddon, ITheme as IXTermTheme, Terminal as XTermTerminal } from "@xterm/xterm"
import { FitAddon } from "@xterm/addon-fit"
import { exists, remToPx } from "@/lib"

type TTerminalRef = Readonly<{
  clear: VoidFunction
  write: (data: string) => void
  writeln: (data: string) => void
  highlight: (
    row: number,
    col: number,
    len: number,
    color: string,
    invertText: boolean
  ) => (() => void) | undefined
  getTerminal: () => XTermTerminal | null
}>
type TTerminalProps = Readonly<{
  fontSize: string
  borderRadius?: string
  onResize?: (cols: number, rows: number) => void
}>
export type TTerminal = TTerminalRef

export const Terminal = forwardRef<TTerminalRef, TTerminalProps>(function T(
  { fontSize, onResize, borderRadius },
  ref
) {
  const containerRef = useRef<HTMLDivElement>(null)
  const terminalRef = useRef<XTermTerminal | null>(null)
  const termFitRef = useRef<FitAddon | null>(null)

  const backgroundColorToken = useColorModeValue("gray.900", "background.darkest")
  const backgroundColor = useToken("colors", backgroundColorToken)
  const textColor = useToken("colors", "gray.100")

  const scrollBarThumbToken = useColorModeValue("gray.500", "gray.200")
  const scrollBarThumbColor = useToken("colors", scrollBarThumbToken)

  const selectionBackgroundToken = useColorModeValue("gray.600", "gray.600")
  const selectionBackgroundColor = useToken("colors", selectionBackgroundToken)

  const terminalTheme = useMemo<Partial<IXTermTheme>>(
    () => ({
      background: backgroundColor,
      foreground: textColor,
      selectionBackground: selectionBackgroundColor,
    }),
    [backgroundColor, selectionBackgroundColor, textColor]
  )

  useLayoutEffect(() => {
    if (!exists(terminalRef.current)) {
      const terminal = new XTermTerminal({
        convertEol: true,
        scrollback: 25_000,
        theme: terminalTheme,
        allowProposedApi: true,
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

      const loadAddon = <T extends ITerminalAddon>(
        AddonClass: new () => T,
        ref: React.MutableRefObject<T | null>
      ) => {
        const addon = new AddonClass()
        ref.current = addon
        terminal.loadAddon(addon)

        return addon
      }

      const termFit = loadAddon(FitAddon, termFitRef)

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

  // Apply outer resize handler to the terminal here.
  // Terminal should be guaranteed to be present by the time we get here.
  useEffect(() => {
    if (!onResize) {
      return
    }

    const disposable = terminalRef.current?.onResize((event) => {
      onResize(event.cols, event.rows)
    })

    return () => {
      disposable?.dispose()
    }
  }, [onResize])

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
        highlight(row: number, startCol: number, len: number, color: string, invertText: boolean) {
          const terminal = terminalRef.current

          if (!terminal) {
            return undefined
          }

          const rowRelative = -terminal.buffer.active.baseY - terminal.buffer.active.cursorY + row

          const marker = terminal.registerMarker(rowRelative)
          const decoration = terminal.registerDecoration({
            marker,
            x: startCol,
            width: len,
            backgroundColor: color,
            foregroundColor: invertText ? "#000000" : "#FFFFFF",
            layer: "top",
            overviewRulerOptions: {
              color: color,
            },
          })

          return () => {
            marker.dispose()
            decoration?.dispose()
          }
        },
        getTerminal() {
          return terminalRef.current
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
        borderRadius={borderRadius ?? "md"}
        borderWidth={6}
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

import { SIDEBAR_WIDTH } from "@/constants"
import { ExclamationCircle } from "@/icons"
import { exists, isError } from "@/lib"
import {
  BoxProps,
  HStack,
  IconButton,
  Popover,
  PopoverContent,
  PopoverTrigger,
  useBreakpointValue,
  useColorModeValue,
} from "@chakra-ui/react"
import { motion } from "framer-motion"
import { RefObject, createContext, useContext, useEffect, useMemo, useRef, useState } from "react"
import { ErrorMessageBox } from "../Error"
import { useBorderColor } from "@/Theme"

type TModalBottomBarProps = Readonly<{
  isModal?: boolean
  hasSidebar?: boolean
  stickToBottom?: boolean
  children: React.ReactNode
}>

const BottomActionBarContext = createContext<{ ref: RefObject<HTMLDivElement> } | undefined>(
  undefined
)
export function BottomActionBar({
  isModal = false,
  hasSidebar = true,
  stickToBottom,
  children,
}: TModalBottomBarProps) {
  const ref = useRef<HTMLDivElement>(null)
  const bottomBarBackgroundColor = useColorModeValue("white", "gray.900")
  const bottomBarBackgroundColorModal = useColorModeValue("white", "background.darkest")
  const borderColor = useBorderColor()
  const translateX = useBreakpointValue({
    base: hasSidebar ? "translateX(-3rem)" : "",
    xl: isModal ? "translateX(-3rem)" : "",
  })
  const paddingX = useBreakpointValue({ base: "3rem", xl: isModal ? "3rem" : "4" })
  const value = useMemo(() => ({ ref }), [ref])

  const width = useMemo(() => {
    if (isModal) {
      return "calc(100% + 5.5rem)"
    }

    if (hasSidebar) {
      return { base: `calc(100vw - ${SIDEBAR_WIDTH})`, xl: "full" }
    }

    return { base: "100vw", xl: "full" }
  }, [hasSidebar, isModal])

  const otherProps = useMemo<BoxProps>(() => {
    if (stickToBottom) {
      return {
        position: "fixed",
        bottom: "2rem",
      }
    }

    return {
      position: "sticky",
      bottom: "-1.1rem",
    }
  }, [stickToBottom])

  return (
    <BottomActionBarContext.Provider value={value}>
      <HStack
        ref={ref}
        as={motion.div}
        {...otherProps}
        initial={{ transform: `translateY(100%) ${translateX}` }}
        animate={{ transform: `translateY(0) ${translateX}` }}
        marginTop="10"
        left="0"
        width={width}
        height="20"
        alignItems="center"
        borderTopWidth="thin"
        borderColor={borderColor}
        backgroundColor={isModal ? bottomBarBackgroundColor : bottomBarBackgroundColorModal}
        justifyContent="space-between"
        paddingX={paddingX}
        zIndex="overlay">
        {children}
      </HStack>
    </BottomActionBarContext.Provider>
  )
}

type TBottomActionBarErrorProps = Readonly<{
  error?: Error | null
  containerRef?: RefObject<HTMLDivElement>
}>
export function BottomActionBarError({ error, containerRef }: TBottomActionBarErrorProps) {
  const ctx = useContext(BottomActionBarContext)
  const { height, width } = useErrorDimensions(containerRef, ctx?.ref)
  // Open error popover when error changes
  const errorButtonRef = useRef<HTMLButtonElement>(null)
  useEffect(() => {
    if (error) {
      errorButtonRef.current?.click()
    }
  }, [error])

  return (
    <Popover placement="top" computePositionOnMount>
      <PopoverTrigger>
        <IconButton
          ref={errorButtonRef}
          visibility={error ? "visible" : "hidden"}
          variant="ghost"
          aria-label="Show errors"
          icon={
            <motion.span
              key={error ? "error" : undefined}
              animate={{ scale: [1, 1.2, 1] }}
              transition={{ type: "keyframes", ease: ["easeInOut"] }}>
              <ExclamationCircle boxSize="8" color="red.400" />
            </motion.span>
          }
          isDisabled={!exists(error)}
        />
      </PopoverTrigger>
      <PopoverContent width={width} margin="4" zIndex="overlay">
        {isError(error) && <ErrorMessageBox maxHeight={height} overflowY="auto" error={error} />}
      </PopoverContent>
    </Popover>
  )
}

function useErrorDimensions(
  ref: RefObject<HTMLElement> | undefined,
  _parentRef: RefObject<HTMLElement> | undefined,
  defaultHeight: BoxProps["height"] = "5xl",
  defaultWidth: BoxProps["width"] = "5xl"
) {
  const [errorHeight, setErrorHeight] = useState<BoxProps["height"]>(defaultHeight)
  const [errorWidth] = useState<BoxProps["width"]>(defaultWidth)

  useEffect(() => {
    const curr = ref?.current
    if (!curr) {
      return
    }
    const observer = new ResizeObserver((entries) => {
      for (const entry of entries) {
        if (entry.target === curr) {
          const heightPx = entry.contentRect.height

          setErrorHeight(`calc(${heightPx}px - 4rem)`)
        }
      }
    })
    observer.observe(curr)

    return () => observer.disconnect()
  }, [ref])

  return { height: errorHeight, width: errorWidth }
}

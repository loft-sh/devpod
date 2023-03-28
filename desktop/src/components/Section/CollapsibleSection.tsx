import { Box, Button, useDisclosure } from "@chakra-ui/react"
import { AnimatePresence, motion, Variants } from "framer-motion"
import { forwardRef, ReactNode, useLayoutEffect, useRef } from "react"

const variants: Variants = {
  enter: {
    opacity: 1,
    height: "auto",
    transition: {
      opacity: { duration: 0.2 },
      height: { duration: 0.3 },
    },
  },
  exit: {
    opacity: 0,
    height: 0,
    transition: {
      opacity: { duration: 0.3 },
      height: { duration: 0.2 },
    },
  },
}

type TCollapsibleSectionProps = Readonly<{
  title: ReactNode | ((isOpen: boolean) => ReactNode)
  children: ReactNode
  isOpen?: boolean
  isDisabled?: boolean
  onOpenChange?: (isOpen: boolean, element: HTMLDivElement | null) => void
}>
export const CollapsibleSection = forwardRef<HTMLDivElement, TCollapsibleSectionProps>(
  function CollapsibleSection(
    { title, children, onOpenChange, isOpen: isOpenProp = false, isDisabled = false },
    ref
  ) {
    const motionRef = useRef<HTMLDivElement>(null)
    const { isOpen, onOpen, onClose, getDisclosureProps, getButtonProps } = useDisclosure()
    const buttonProps = getButtonProps({ isDisabled })
    const disclosureProps = getDisclosureProps()

    useLayoutEffect(() => {
      if (isOpenProp) {
        onOpen()
      } else {
        onClose()
      }
    }, [isOpenProp, onClose, onOpen])

    return (
      <Box width="full">
        <Button ref={ref} variant="ghost" width="full" {...buttonProps}>
          <Box as="span" flex="1" textAlign="left">
            {typeof title === "function" ? title(isOpen) : title}
          </Box>
        </Button>
        <AnimatePresence initial={false}>
          {isOpen && (
            <motion.div
              ref={motionRef}
              variants={variants}
              initial="exit"
              animate="enter"
              exit="exit"
              onAnimationComplete={() => onOpenChange?.(isOpen, motionRef.current)}
              style={{
                overflow: "hidden",
                display: "block",
              }}>
              <Box {...disclosureProps} marginTop={4} paddingLeft={4} width="full">
                {children}
              </Box>
            </motion.div>
          )}
        </AnimatePresence>
      </Box>
    )
  }
)

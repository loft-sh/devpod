import { Box, BoxProps, Button, ButtonProps, Icon, useDisclosure } from "@chakra-ui/react"
import { AnimatePresence, motion, Variants } from "framer-motion"
import { forwardRef, ReactNode, useLayoutEffect, useRef } from "react"
import { AiOutlineCaretRight } from "react-icons/ai"

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
  showIcon?: boolean
  headerProps?: ButtonProps
  contentProps?: BoxProps
  onOpenChange?: (isOpen: boolean, element: HTMLDivElement | null) => void
}>
export const CollapsibleSection = forwardRef<HTMLDivElement, TCollapsibleSectionProps>(
  function CollapsibleSection(
    {
      title,
      headerProps,
      contentProps,
      children,
      onOpenChange,
      isOpen: isOpenProp = false,
      isDisabled = false,
      showIcon,
    },
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
        <Button ref={ref} variant="ghost" width="full" {...headerProps} {...buttonProps}>
          <Box as="span" flex="1" textAlign="left">
            {showIcon && (
              <Icon
                marginRight={"3px"}
                fontSize={"12px"}
                transition={"transform .2s"}
                transform={isOpen ? "rotate(90deg)" : ""}
                as={AiOutlineCaretRight}
              />
            )}
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
              <Box
                {...disclosureProps}
                marginTop={4}
                paddingLeft={4}
                width="full"
                {...contentProps}>
                {children}
              </Box>
            </motion.div>
          )}
        </AnimatePresence>
      </Box>
    )
  }
)

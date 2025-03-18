import { useBorderColor } from "@/Theme"
import { STATUS_BAR_HEIGHT } from "@/constants"
import { Box, Flex, HStack, useColorModeValue, useToken } from "@chakra-ui/react"
import { ReactNode } from "react"
import { StatusBar } from "./StatusBar"
import { Toolbar } from "./Toolbar"
import { isMacOS } from "@/lib"

type TProLayoutProps = Readonly<{
  toolbarItems: ReactNode
  statusBarItems: ReactNode
  children: ReactNode
}>
export function ProLayout({ toolbarItems, statusBarItems, children }: TProLayoutProps) {
  const contentBackgroundColor = useColorModeValue("white", "background.darkest")
  const toolbarHeight = useToken("sizes", "10")
  const statusBarHeight = useToken("sizes", "8")
  const borderColor = useBorderColor()

  return (
    <Flex width="100vw" maxWidth="100vw" overflow="hidden">
      <Box width="full" height="full">
        <Box
          data-tauri-drag-region // keep!
          backgroundColor={contentBackgroundColor}
          position="relative"
          width="full"
          height="full"
          overflowY="auto">
          <Toolbar
            backgroundColor={contentBackgroundColor}
            height={toolbarHeight}
            position="sticky"
            data-tauri-drag-region // keep!
            width="full">
            <HStack
              justifyContent="space-between"
              paddingLeft={isMacOS ? "24" : "8"}
              data-tauri-drag-region // keep!
            >
              {toolbarItems}
            </HStack>
          </Toolbar>
          <Box
            as="main"
            paddingTop="8"
            paddingBottom={STATUS_BAR_HEIGHT}
            paddingX="8"
            width="full"
            height={`calc(100vh - ${toolbarHeight} - ${statusBarHeight})`}
            overflowX="hidden"
            overflowY="auto">
            {children}
          </Box>
          <StatusBar
            height={STATUS_BAR_HEIGHT}
            position="fixed"
            bottom="0"
            width="full"
            borderTopWidth="thin"
            borderTopColor={borderColor}
            backgroundColor={contentBackgroundColor}>
            {statusBarItems}
          </StatusBar>
        </Box>
      </Box>
    </Flex>
  )
}

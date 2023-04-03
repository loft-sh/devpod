import {
  Box,
  BoxProps,
  Flex,
  Grid,
  Link,
  Text,
  useColorModeValue,
  useToken,
  VStack,
} from "@chakra-ui/react"
import { ReactElement, ReactNode } from "react"
import { LinkProps, NavLink as RouterLink } from "react-router-dom"
import { useSettings } from "../../contexts"
import { DevpodIcon } from "../../icons"
import { useBorderColor } from "../../Theme"

type TSidebarProps = Readonly<{ children?: readonly ReactElement[] }> & BoxProps
export function Sidebar({ children, ...boxProps }: TSidebarProps) {
  const iconColor = useToken("colors", "primary.500")
  const borderColor = useBorderColor()
  const backgroundColor = useColorModeValue("white", "black")
  const isLeft = useSettings().sidebarPosition === "left"

  return (
    <Grid
      resizable="horizontal"
      as="aside"
      templateRows="6rem 1fr 6rem"
      width="full"
      height="100vh"
      borderRightColor={borderColor}
      borderRightWidth="thin"
      position="relative"
      {...boxProps}>
      <Flex
        alignItems="start"
        flexFlow={isLeft ? "row" : "row-reverse"}
        justify="flex-start"
        width="full">
        <Box width="8" />
        <DevpodIcon boxSize={8} color={iconColor} />
      </Flex>
      <VStack as="nav" align="start">
        {children}
      </VStack>
      <VStack></VStack>

      {/* Background Material */}
      <Box
        aria-hidden
        width="full"
        height="full"
        position="absolute"
        backgroundColor={backgroundColor}
        zIndex={-1}
        opacity={0.7}
      />
    </Grid>
  )
}

type TSidebarMenuProps = Pick<LinkProps, "to"> & Readonly<{ children?: ReactNode; icon: ReactNode }>
export function SidebarMenuItem({ to, children, icon }: TSidebarMenuProps) {
  const settings = useSettings()
  const backgroundColorToken = useColorModeValue("blackAlpha.100", "whiteAlpha.200")
  const backgroundColor = useToken("colors", backgroundColorToken)
  const borderColorToken = useColorModeValue("blackAlpha.200", "whiteAlpha.300")
  const borderColor = useToken("colors", borderColorToken)
  const isLeft = settings.sidebarPosition === "left"

  return (
    <Box paddingX="4" width="full">
      <Link
        display="flex"
        paddingX="4"
        paddingY="3"
        as={RouterLink}
        to={to}
        width="full"
        borderRadius="md"
        flexDirection="row"
        flexGrow="nowrap"
        alignItems="center"
        flexFlow={isLeft ? "row" : "row-reverse"}
        justifyContent="flex-start"
        borderWidth="thin"
        borderColor="transparent"
        opacity={0.6}
        _hover={{ textDecoration: "none", backgroundColor }}
        // @ts-ignore // this function is added by react-router-dom's `NavLink`
        style={({ isActive }) => ({
          ...(isActive
            ? {
                backgroundColor,
                borderColor,
                opacity: 1,
              }
            : {}),
        })}>
        {icon}
        <Box width="2" />
        <Text color="chakra-body-text">{children}</Text>
      </Link>
    </Box>
  )
}

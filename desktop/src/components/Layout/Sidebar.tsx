import { useSystemTheme } from "@/lib"
import {
  Box,
  BoxProps,
  Flex,
  Grid,
  HStack,
  Link,
  Text,
  useColorMode,
  useColorModeValue,
  useToken,
  VStack,
} from "@chakra-ui/react"
import { cloneElement, ReactElement, ReactNode } from "react"
import { LinkProps, NavLink as RouterLink } from "react-router-dom"
import { useSettings } from "../../contexts"
import { DevpodWordmark } from "../../icons"
import { useBorderColor } from "../../Theme"
import { LoftOSSBadge } from "../LoftOSSBadge"

type TSidebarProps = Readonly<{ children?: readonly ReactElement[] }> & BoxProps
export function Sidebar({ children, ...boxProps }: TSidebarProps) {
  const { colorMode } = useColorMode()
  const systemTheme = useSystemTheme()
  const borderColor = useBorderColor()
  const backgroundColor = useColorModeValue("white", "black")
  const alternativeBackgroundColor = useColorModeValue("gray.100", "gray.900")
  const wordmarkColor = useColorModeValue("black", "white")
  const isLeft = useSettings().sidebarPosition === "left"
  const { transparency } = useSettings()

  const sharedBackgroundMaterialProps = {
    "aria-hidden": true,
    width: "full",
    height: "full",
    position: "absolute",
    zIndex: -1,
  } as const

  return (
    <Grid
      as="aside"
      templateRows="4rem 1fr 6rem"
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
        <DevpodWordmark marginTop={2} width={32} height={10} color={wordmarkColor} />
      </Flex>
      <VStack marginTop="8" as="nav" align="start">
        {children}
      </VStack>
      <HStack alignSelf="end" paddingTop="4" paddingLeft="8" paddingBottom="4">
        <LoftOSSBadge />
      </HStack>

      {/* Background Material */}
      {transparency ? (
        <Box
          {...sharedBackgroundMaterialProps}
          backgroundColor={backgroundColor}
          opacity={systemTheme === "dark" && colorMode !== "dark" ? 0.8 : 0.2}
        />
      ) : (
        <Box {...sharedBackgroundMaterialProps} backgroundColor={alternativeBackgroundColor} />
      )}
    </Grid>
  )
}

type TSidebarMenuProps = Pick<LinkProps, "to"> &
  Readonly<{ children?: ReactNode; icon: ReactElement }>
export function SidebarMenuItem({ to, children, icon: iconProps }: TSidebarMenuProps) {
  const settings = useSettings()
  const backgroundColorToken = useColorModeValue("gray.300", "gray.700")
  const backgroundColor = useToken("colors", backgroundColorToken)
  const borderColorToken = useColorModeValue("primary.600", "primary.600")
  const borderColor = useToken("colors", borderColorToken)
  const backgroundActiveColorToken = useColorModeValue("primary.500", "primary.700")
  const backgroundActiveColor = useToken("colors", backgroundActiveColorToken)
  const isLeft = settings.sidebarPosition === "left"
  const icon = cloneElement(iconProps, { boxSize: 4 })

  return (
    <Box paddingX="4" width="full">
      <Link
        variant="ghost"
        display="flex"
        paddingX="4"
        paddingY="2"
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
        opacity={0.8}
        fontSize="md"
        color="gray.900"
        _dark={{ color: "gray.100" }}
        _hover={{ textDecoration: "none", backgroundColor }}
        // @ts-ignore // this function is added by react-router-dom's `NavLink`
        style={({ isActive }) => ({
          ...(isActive
            ? {
                backgroundColor: backgroundActiveColor,
                color: "white",
                borderColor,
                opacity: 1,
              }
            : {}),
        })}>
        {icon}
        <Box width="2" />
        <Text>{children}</Text>
      </Link>
    </Box>
  )
}

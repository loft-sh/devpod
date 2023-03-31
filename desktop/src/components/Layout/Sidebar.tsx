import {
  Box,
  BoxProps,
  Flex,
  Grid,
  Link,
  useColorModeValue,
  useToken,
  VStack,
} from "@chakra-ui/react"
import { ReactElement, ReactNode } from "react"
import { LinkProps, NavLink as RouterLink } from "react-router-dom"
import { DevpodIcon } from "../../icons"
import { Notifications } from "./Notifications"

type TSidebarProps = Readonly<{ children?: readonly ReactElement[] }> & BoxProps
export function Sidebar({ children, ...boxProps }: TSidebarProps) {
  const iconColor = useToken("colors", "primary")
  const borderColor = useColorModeValue("gray.100", "gray.700")

  return (
    <Grid
      templateRows="6rem 1fr 6rem"
      width="full"
      height="100vh"
      borderRightColor={borderColor}
      borderRightWidth="medium"
      {...boxProps}>
      <Flex paddingLeft="6" alignItems="center" justify="flex-start" width="full">
        <DevpodIcon boxSize={8} color={iconColor} />
      </Flex>
      <VStack align="start">{children}</VStack>
      <VStack>
        <Notifications />
      </VStack>
    </Grid>
  )
}

type TSidebarMenuProps = Pick<LinkProps, "to"> & Readonly<{ children?: ReactNode }>
export function SidebarMenuItem({ to, children }: TSidebarMenuProps) {
  return (
    <Box paddingX="4" width="full">
      <Link
        display="block"
        paddingX="4"
        paddingY="4"
        as={RouterLink}
        to={to}
        width="full"
        borderRadius="sm"
        _activeLink={{ fontWeight: "bold" }}>
        {children}
      </Link>
    </Box>
  )
}

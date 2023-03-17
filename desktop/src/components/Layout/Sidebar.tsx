import { Box, Flex, Grid, Link, useToken, VStack } from "@chakra-ui/react"
import { ReactNode } from "react"
import { LinkProps, NavLink as RouterLink } from "react-router-dom"
import { DevpodIcon } from "../../icons"

type TSidebarProps = Readonly<{ children?: ReactNode }>
export function Sidebar({ children }: TSidebarProps) {
  // FIXME: refactor into type-safe hook
  const iconColor = useToken("colors", "primary")
  const sidebarBackgroundColor = useToken("colors", "gray.100")

  return (
    <Grid templateRows="6rem 1fr 6rem" width="full" backgroundColor={sidebarBackgroundColor}>
      <Flex paddingLeft="6" alignItems="center" justify="flex-start" width="full">
        <DevpodIcon boxSize={8} color={iconColor} />
      </Flex>
      <VStack align="start">{children}</VStack>
      <Box />
    </Grid>
  )
}

type TSidebarMenuProps = Pick<LinkProps, "to"> & Readonly<{ children?: ReactNode }>
export function SidebarMenuItem({ to, children }: TSidebarMenuProps) {
  const activeBackgroundColor = useToken("colors", "gray.300")

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
        _activeLink={{ fontWeight: "bold", backgroundColor: activeBackgroundColor }}>
        {children}
      </Link>
    </Box>
  )
}

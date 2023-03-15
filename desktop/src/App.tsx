import {
  Text,
  Box,
  Code,
  Container,
  Grid,
  Link,
  VStack,
  HStack,
  Menu,
  MenuButton,
  MenuList,
  MenuItem,
  Checkbox,
} from "@chakra-ui/react"
import { useEffect } from "react"
import { Outlet, useRouteError, Link as RouterLink, useMatch, useNavigate } from "react-router-dom"
import { Sidebar, SidebarMenuItem } from "./components"
import { Routes } from "./routes"
import { version } from "../package.json"
import { Debug, useArch, useDebug, usePlatform } from "./lib"

export function App() {
  const navigate = useNavigate()
  const rootRouteMatch = useMatch(Routes.ROOT)
  const platform = usePlatform()
  const arch = useArch()
  const debug = useDebug()

  useEffect(() => {
    if (rootRouteMatch !== null) {
      navigate(Routes.WORKSPACES)
    }
  }, [navigate, rootRouteMatch])

  return (
    <VStack spacing={4} height="100vh" width="100vw" maxWidth="100vw" overflow="hidden">
      <Box width="full" height="full" overflowY="auto">
        <Grid height="full" templateColumns="15rem 1fr">
          <Sidebar>
            <SidebarMenuItem to={Routes.WORKSPACES}>Workspaces</SidebarMenuItem>
            <SidebarMenuItem to={Routes.PROVIDERS}>Providers</SidebarMenuItem>
          </Sidebar>

          <HStack
            justify="space-between"
            paddingX="6"
            position="fixed"
            bottom="0"
            backgroundColor="gray.300"
            width="full"
            fontSize="sm"
            zIndex="1">
            <Text>
              Version {version} | {platform ?? "unknown platform"} | {arch ?? "unknown arch"}
            </Text>
            {Debug.isEnabled && (
              <Menu>
                <MenuButton>Debug</MenuButton>
                <MenuList>
                  <MenuItem onClick={() => Debug.toggle?.("logs")}>
                    <Checkbox isChecked={debug.logs} />
                    <Text paddingLeft="4">Debug Logs</Text>
                  </MenuItem>
                </MenuList>
              </Menu>
            )}
          </HStack>
          <Box position="relative" width="full" overflow="hidden">
            <Box paddingX="8" paddingY="8" width="full" height="full" overflowY="auto">
              <Outlet />
            </Box>
          </Box>
        </Grid>
      </Box>
    </VStack>
  )
}

export function ErrorPage() {
  const error = useRouteError()

  return (
    <Container padding="16">
      <VStack>
        <Text>Whoops, something went wrong or this route doesn&apos;t exist.</Text>
        <Box paddingBottom="6">
          <Link as={RouterLink} to={Routes.ROOT}>
            Go back to home
          </Link>
        </Box>
        <Code>{JSON.stringify(error, null, 2)}</Code>{" "}
      </VStack>
    </Container>
  )
}

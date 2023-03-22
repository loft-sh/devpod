import {
  Box,
  BoxProps,
  Checkbox,
  Code,
  Container,
  Flex,
  Grid,
  GridItem,
  GridProps,
  HStack,
  Link,
  Menu,
  MenuButton,
  MenuItem,
  MenuList,
  Text,
  useColorModeValue,
  VStack,
} from "@chakra-ui/react"
import { useEffect, useMemo } from "react"
import { Link as RouterLink, Outlet, useMatch, useNavigate, useRouteError } from "react-router-dom"
import { version } from "../package.json"
import { Sidebar, SidebarMenuItem } from "./components"
import { useSettings } from "./contexts"
import { Debug, useArch, useDebug, usePlatform } from "./lib"
import { Routes } from "./routes"

const TITLE_BAR_SAFE_AREA: BoxProps["height"] = "10"
const STATUS_BAR_SAFE_AREA: BoxProps["height"] = "10"

export function App() {
  const navigate = useNavigate()
  const rootRouteMatch = useMatch(Routes.ROOT)
  const platform = usePlatform()
  const arch = useArch()
  const debug = useDebug()
  const statusBarBackgroundColor = useColorModeValue("gray.300", "gray.600")
  const { sidebarPosition } = useSettings()

  const mainGridProps = useMemo<GridProps>(() => {
    if (sidebarPosition === "right") {
      return { templateAreas: `"main sidebar"`, gridTemplateColumns: "1fr 15rem" }
    }

    return { templateAreas: `"sidebar main"`, gridTemplateColumns: "15rem 1fr" }
  }, [sidebarPosition])

  useEffect(() => {
    if (rootRouteMatch !== null) {
      navigate(Routes.WORKSPACES)
    }
  }, [navigate, rootRouteMatch])

  return (
    <Flex height="100vh" width="100vw" maxWidth="100vw" overflow="hidden">
      <Box
        data-tauri-drag-region // keep!
        height={TITLE_BAR_SAFE_AREA}
        position="fixed"
        top="0"
        width="full"
        textAlign={"center"}>
        <Text
          data-tauri-drag-region // keep!
          fontWeight="bold"
          marginTop="2">
          Devpod Desktop
        </Text>
      </Box>
      <Box width="full" height="full">
        <Grid height="full" {...mainGridProps}>
          <GridItem area="sidebar">
            <Sidebar paddingTop={TITLE_BAR_SAFE_AREA}>
              <SidebarMenuItem to={Routes.WORKSPACES}>Workspaces</SidebarMenuItem>
              <SidebarMenuItem to={Routes.PROVIDERS}>Providers</SidebarMenuItem>
              <SidebarMenuItem to={Routes.SETTINGS}>Settings</SidebarMenuItem>
            </Sidebar>
          </GridItem>

          <GridItem area="main" height="100vh" width="full" overflowX="auto">
            <Box
              data-tauri-drag-region // keep!
              paddingTop={TITLE_BAR_SAFE_AREA}
              position="relative"
              width="full"
              height="full"
              overflowY="auto"
              paddingBottom={STATUS_BAR_SAFE_AREA}>
              <Box paddingX="8" paddingY="8" width="full" height="full" overflowY="auto">
                <Outlet />
              </Box>
            </Box>
          </GridItem>
        </Grid>
      </Box>

      <HStack
        justify="space-between"
        paddingX="6"
        position="fixed"
        bottom="0"
        backgroundColor={statusBarBackgroundColor}
        width="full"
        fontSize="sm"
        zIndex="1">
        <Text>
          Version {version} | {platform ?? "unknown platform"} | {arch ?? "unknown arch"}
        </Text>
        {debug.isEnabled && (
          <Menu>
            <MenuButton>Debug</MenuButton>
            <MenuList>
              <MenuItem onClick={() => Debug.toggle?.("logs")}>
                <Checkbox isChecked={debug.options.logs} />
                <Text paddingLeft="4">Debug Logs</Text>
              </MenuItem>
            </MenuList>
          </Menu>
        )}
      </HStack>
    </Flex>
  )
}

export function ErrorPage() {
  const error = useRouteError()

  return (
    <Container padding="16">
      <VStack>
        <Text>Whoops, something went wrong or this page doesn&apos;t exist.</Text>
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

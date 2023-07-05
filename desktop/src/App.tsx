import { ExternalLinkIcon } from "@chakra-ui/icons"
import {
  Box,
  BoxProps,
  Button,
  Code,
  Container,
  Flex,
  Grid,
  GridItem,
  GridProps,
  Link,
  Text,
  useColorModeValue,
  useToken,
  VStack,
} from "@chakra-ui/react"
import { useEffect, useMemo } from "react"
import { Outlet, Link as RouterLink, useMatch, useNavigate, useRouteError } from "react-router-dom"
import { client } from "./client"
import { Sidebar, SidebarMenuItem, StatusBar, Toolbar } from "./components"
import { SIDEBAR_WIDTH, STATUS_BAR_HEIGHT } from "./constants"
import { ToolbarProvider, useChangeSettings, useSettings } from "./contexts"
import { Briefcase, Cog, Stack3D } from "./icons"
import { isLinux, isMacOS, isWindows } from "./lib"
import { Routes } from "./routes"
import { useBorderColor } from "./Theme"
import { useAppReady } from "./useAppReady"
import { useWelcomeModal } from "./useWelcomeModal"

const showTitleBar = isMacOS || isLinux || isWindows
const showDevPodTitle = isMacOS || isLinux

export function App() {
  const { modal: appReadyModal } = useAppReady()
  const navigate = useNavigate()
  const rootRouteMatch = useMatch(Routes.ROOT)
  const { sidebarPosition } = useSettings()
  const contentBackgroundColor = useColorModeValue("white", "black")
  const toolbarHeight = useToken("sizes", showTitleBar ? "28" : "20")
  const borderColor = useBorderColor()

  const titleBarSafeArea = useMemo<BoxProps["height"]>(() => {
    return showTitleBar ? "12" : 0
  }, [])

  const mainGridProps = useMemo<GridProps>(() => {
    if (sidebarPosition === "right") {
      return { templateAreas: `"main sidebar"`, gridTemplateColumns: `1fr ${SIDEBAR_WIDTH}` }
    }

    return { templateAreas: `"sidebar main"`, gridTemplateColumns: `${SIDEBAR_WIDTH} 1fr` }
  }, [sidebarPosition])

  useEffect(() => {
    if (rootRouteMatch !== null) {
      navigate(Routes.WORKSPACES)
    }
  }, [navigate, rootRouteMatch])

  const { modal: welcomeModal } = useWelcomeModal()
  usePartyParrot()

  const handleAnnouncementClicked = () => {
    client.openLink("https://devpod.sh/engine")
  }

  return (
    <>
      <Flex height="100vh" width="100vw" maxWidth="100vw" overflow="hidden">
        {showTitleBar && (
          <Box
            data-tauri-drag-region // keep!
            height={titleBarSafeArea}
            position="fixed"
            top="0"
            width="full"
            display="grid"
            gridTemplateColumns="1fr 1fr 1fr"
            textAlign={"center"}
            zIndex="tooltip"
            justifyItems="center">
            {showDevPodTitle && (
              <Text
                gridColumn="2"
                data-tauri-drag-region // keep!
                fontWeight="bold"
                marginTop="2">
                DevPod
              </Text>
            )}
            <Button
              data-tauri-drag-region // keep!
              variant="announcement"
              gridColumn="3"
              marginTop="2"
              width="fit-content"
              justifySelf="end"
              marginRight="10"
              rightIcon={<ExternalLinkIcon />}
              onClick={handleAnnouncementClicked}>
              Try Loft DevPod Engine
            </Button>
          </Box>
        )}

        <Box width="full" height="full">
          <Grid height="full" {...mainGridProps}>
            <GridItem area="sidebar">
              <Sidebar paddingTop={titleBarSafeArea}>
                <SidebarMenuItem to={Routes.WORKSPACES} icon={<Briefcase />}>
                  Workspaces
                </SidebarMenuItem>
                <SidebarMenuItem to={Routes.PROVIDERS} icon={<Stack3D />}>
                  Providers
                </SidebarMenuItem>
                <SidebarMenuItem to={Routes.SETTINGS} icon={<Cog />}>
                  Settings
                </SidebarMenuItem>
              </Sidebar>
            </GridItem>

            <GridItem area="main" height="100vh" width="full" overflowX="auto">
              <ToolbarProvider>
                <Box
                  data-tauri-drag-region // keep!
                  backgroundColor={contentBackgroundColor}
                  position="relative"
                  width="full"
                  height="full"
                  overflowY="auto">
                  <Toolbar
                    paddingTop={titleBarSafeArea}
                    backgroundColor={contentBackgroundColor}
                    height={toolbarHeight}
                    position="sticky"
                    zIndex={1}
                    width="full"
                  />
                  <Box
                    as="main"
                    paddingTop="8"
                    paddingBottom={STATUS_BAR_HEIGHT}
                    paddingX="8"
                    width="full"
                    height={`calc(100vh - ${toolbarHeight})`}
                    overflowY="auto">
                    <Outlet />
                  </Box>
                  <StatusBar
                    height={STATUS_BAR_HEIGHT}
                    position="fixed"
                    bottom="0"
                    width={`calc(100% - ${SIDEBAR_WIDTH})`}
                    borderTopWidth="thin"
                    borderTopColor={borderColor}
                    backgroundColor={contentBackgroundColor}
                  />
                </Box>
              </ToolbarProvider>
            </GridItem>
          </Grid>
        </Box>
      </Flex>

      {welcomeModal}
      {appReadyModal}
    </>
  )
}

export function ErrorPage() {
  const error = useRouteError()
  const contentBackgroundColor = useColorModeValue("white", "black")

  return (
    <Box height="100vh" width="100vw" backgroundColor={contentBackgroundColor}>
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
    </Box>
  )
}

function usePartyParrot() {
  const { set: setSettings, settings } = useChangeSettings()

  useEffect(() => {
    const handler = (event: KeyboardEvent) => {
      if (event.shiftKey && event.ctrlKey && event.key.toLowerCase() === "p") {
        const current = settings.partyParrot
        setSettings("partyParrot", !current)
      }
    }
    document.addEventListener("keyup", handler)

    return () => document.addEventListener("keyup", handler)
  }, [setSettings, settings.partyParrot])
}

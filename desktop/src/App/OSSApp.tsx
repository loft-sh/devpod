import { client } from "@/client"
import { QueryKeys } from "@/queryKeys"
import {
  Box,
  Flex,
  Grid,
  GridItem,
  GridProps,
  HStack,
  LinkBox,
  LinkOverlay,
  Text,
  VStack,
  useColorModeValue,
  useToken,
} from "@chakra-ui/react"
import { useQuery } from "@tanstack/react-query"
import { useEffect, useMemo } from "react"
import { Outlet, Link as RouterLink, useMatch, useNavigate } from "react-router-dom"
import { useBorderColor } from "../Theme"
import {
  Notifications,
  ProSwitcher,
  Sidebar,
  SidebarMenuItem,
  StatusBar,
  Toolbar,
} from "../components"
import { SIDEBAR_WIDTH, STATUS_BAR_HEIGHT } from "../constants"
import { ToolbarProvider, useProviders, useSettings } from "../contexts"
import { Briefcase, Cog, Stack3D } from "../icons"
import { isLinux, isMacOS } from "../lib"
import { Routes } from "../routes"
import { useWelcomeModal } from "../useWelcomeModal"
import { showTitleBar, titleBarSafeArea } from "./constants"
import { useAppReady } from "./useAppReady"

export function OSSApp() {
  const navigate = useNavigate()
  const { errorModal, changelogModal, proLoginModal } = useAppReady()
  const rootRouteMatch = useMatch(Routes.ROOT)
  const { sidebarPosition } = useSettings()
  const contentBackgroundColor = useColorModeValue("white", "background.darkest")
  const actionHoverColor = useColorModeValue("gray.100", "gray.700")
  const toolbarHeight = useToken("sizes", showTitleBar ? "28" : "20")
  const borderColor = useBorderColor()
  const showTitle = isMacOS || isLinux

  const providerUpdateInfo = useProviderUpdates()
  const providerUpdateCount = providerUpdateInfo?.length ?? 0

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

  return (
    <>
      <Flex width="100vw" maxWidth="100vw" overflow="hidden">
        {showTitleBar && <TitleBar showTitle={showTitle} />}

        <Box width="full" height="full" >
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
                    width="full">
                    <Grid
                      alignContent="center"
                      templateRows="1fr"
                      templateColumns="minmax(auto, 18rem) 3fr fit-content(15rem)"
                      width="full"
                      paddingX="4">
                      <GridItem display="flex" alignItems="center">
                        <Toolbar.Title />
                      </GridItem>
                      <GridItem
                        marginLeft={2}
                        display="flex"
                        alignItems="center"
                        justifyContent="start"
                        columnGap={4}>
                        <Toolbar.Actions />
                      </GridItem>
                      <GridItem display="flex" alignItems="center" justifyContent="center">
                        <Box mr="4">
                          <Notifications
                            getActionDestination={(action) => Routes.toAction(action.id)}
                            badgeNumber={providerUpdateCount}
                            providerUpdates={
                              providerUpdateInfo &&
                              providerUpdateCount > 0 && (
                                <>
                                  {providerUpdateInfo.map(({ providerName }) => (
                                    <LinkBox
                                      key={providerName}
                                      padding={2}
                                      fontSize="sm"
                                      borderRadius="md"
                                      width="full"
                                      display="flex"
                                      flexFlow="row nowrap"
                                      alignItems="center"
                                      gap={3}
                                      _hover={{ backgroundColor: actionHoverColor }}>
                                      <Stack3D color="gray.400" />
                                      <VStack align="start" spacing="0">
                                        <Text>
                                          <LinkOverlay
                                            as={RouterLink}
                                            to={Routes.PROVIDERS}
                                            textTransform="capitalize">
                                            <Text fontWeight="bold">Provider {providerName}</Text>
                                          </LinkOverlay>
                                        </Text>
                                        <Text marginTop="-1">Update available</Text>
                                      </VStack>
                                    </LinkBox>
                                  ))}
                                </>
                              )
                            }
                          />
                        </Box>
                        <ProSwitcher />
                      </GridItem>
                    </Grid>
                  </Toolbar>
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
                    backgroundColor={contentBackgroundColor}>
                    <HStack>
                      <StatusBar.Version />
                      <StatusBar.Platform />
                      <StatusBar.Arch />
                    </HStack>

                    <HStack>
                      <StatusBar.ZoomMenu />
                      <StatusBar.GitHubStar />
                      <StatusBar.OSSDocs />
                      <StatusBar.OSSReportIssue />
                      <StatusBar.DebugMenu />
                    </HStack>
                  </StatusBar>
                </Box>
              </ToolbarProvider>
            </GridItem>
          </Grid>
        </Box>
      </Flex>

      {welcomeModal}
      {errorModal}
      {changelogModal}
      {proLoginModal}
    </>
  )
}

type TTitleBarProps = Readonly<{
  showTitle?: boolean
}>
function TitleBar({ showTitle = true }: TTitleBarProps) {
  return (
    <Box
      data-tauri-drag-region // keep!
      height={titleBarSafeArea}
      position="fixed"
      top="0"
      width="full"
      textAlign="center"
      zIndex="modal"
      justifyItems="center">
      {showTitle && (
        <Text
          data-tauri-drag-region // keep!
          fontWeight="bold"
          marginTop="2">
          DevPod
        </Text>
      )}
    </Box>
  )
}

function useProviderUpdates() {
  const [[providers]] = useProviders()

  const { data: providerUpdateInfo } = useQuery({
    // eslint-disable-next-line @tanstack/query/exhaustive-deps
    queryKey: QueryKeys.PROVIDERS_CHECK_UPDATE_ALL,
    queryFn: async () => {
      if (providers === undefined || Object.keys(providers).length === 0) {
        return
      }

      const results = await Promise.allSettled(
        Object.entries(providers)
          .filter(([, provider]) => !provider.isProxyProvider)
          .map(async ([p]) => ({
            name: p,
            update: await client.providers.checkUpdate(p),
          }))
      )

      return results
        .map((r) => {
          if (r.status !== "fulfilled" || r.value.update.err) {
            return null
          }

          if (!r.value.update.val.updateAvailable) {
            return null
          }

          return { providerName: r.value.name, updateAvailable: r.value.update.val.updateAvailable }
        })
        .filter((r): r is Exclude<typeof r, null> => r !== null)
    },
    refetchInterval: 1000 * 60 * 60 * 30, // 30 minutes
    staleTime: Infinity,
  })

  return providerUpdateInfo
}

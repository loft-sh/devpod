import { STATUS_BAR_HEIGHT } from "@/constants"
import { ProviderProvider } from "@/contexts/DevPodContext/DevPodProvider"
import { BellDuotone, CogDuotone } from "@/icons"
import { TConnectionStatus, useConnectionStatus } from "@/lib"
import { QueryKeys } from "@/queryKeys"
import { Routes } from "@/routes"
import { TPlatformVersionInfo } from "@/types"
import {
  Box,
  Divider,
  HStack,
  IconButton,
  Link,
  List,
  ListItem,
  Text,
  Tooltip,
  useColorModeValue,
} from "@chakra-ui/react"
import { useQuery } from "@tanstack/react-query"
import { useMemo } from "react"
import { Outlet, Link as RouterLink } from "react-router-dom"
import { Notifications, ProLayout, StatusBar, Toolbar } from "../components"
import {
  ProInstancesProvider,
  ProProvider,
  ProWorkspaceStore,
  ToolbarProvider,
  WorkspaceStoreProvider,
  useProContext,
  useProHost,
} from "../contexts"

export function ProApp() {
  const host = useProHost()
  if (!host) {
    throw new Error("No host found. This shouldn't happen")
  }

  const store = useMemo(() => new ProWorkspaceStore(host), [host])

  return (
    <WorkspaceStoreProvider store={store}>
      <ProviderProvider>
        <ProInstancesProvider>
          <ToolbarProvider>
            <ProProvider host={host}>
              <ProAppContent host={host} />
            </ProProvider>
          </ToolbarProvider>
        </ProInstancesProvider>
      </ProviderProvider>
    </WorkspaceStoreProvider>
  )
}

type TProAppContentProps = Readonly<{ host: string }>
function ProAppContent({ host }: TProAppContentProps) {
  const connectionStatus = useConnectionStatus()
  const versionInfo = usePlatformVersion()
  const iconColor = useColorModeValue("primary.600", "primary.400")

  return (
    <ProLayout
      toolbarItems={
        <>
          <HStack gap="4">
            <Box>
              <Toolbar.Title />
            </Box>
            <Box>
              <Toolbar.Actions />
            </Box>
          </HStack>
          <HStack pr="2">
            <Link as={RouterLink} to={Routes.toProSettings(host)}>
              <IconButton
                variant="ghost"
                size="md"
                rounded="full"
                aria-label="Go to settings"
                icon={<CogDuotone color={iconColor} />}
              />
            </Link>
            <Notifications
              getActionDestination={(action) => Routes.toProWorkspace(host, action.targetID)}
              icon={<BellDuotone color={iconColor} position="absolute" />}
            />
          </HStack>
        </>
      }
      statusBarItems={
        <>
          <HStack />
          <HStack gap="1">
            <Tooltip label="Client version">
              {/* The box is just here for tooltip to take a ref */}
              <Box>
                <StatusBar.Version />
              </Box>
            </Tooltip>
            {versionInfo?.currentProviderVersion && (
              <Tooltip label="Provider version">
                <Text>
                  {versionInfo.currentProviderVersion}
                  {versionInfo.currentProviderVersion !== versionInfo.remoteProviderVersion
                    ? `/${versionInfo.remoteProviderVersion}`
                    : ""}
                </Text>
              </Tooltip>
            )}
            {versionInfo?.serverVersion && (
              <Tooltip label="Platform version">
                <Text>{versionInfo.serverVersion}</Text>
              </Tooltip>
            )}
            <StatusBar.Platform />
            <StatusBar.Arch />
            <StatusBar.DebugMenu />
            <Divider orientation="vertical" h={STATUS_BAR_HEIGHT} mx="2" />
            <ConnectionStatus status={connectionStatus} />
          </HStack>
        </>
      }>
      <Outlet />
    </ProLayout>
  )
}

function usePlatformVersion(): TPlatformVersionInfo | undefined {
  const { host, client } = useProContext()
  const { data } = useQuery({
    queryKey: QueryKeys.versionInfo(host),
    queryFn: async () => {
      return (await client.getVersion()).unwrap()
    },
    refetchInterval: 1_000 * 60, // every minute
  })

  return data
}

type TConnectionStatusProps = Readonly<{
  status: TConnectionStatus
}>
function ConnectionStatus({ status }: TConnectionStatusProps) {
  if (status.isLoading) {
    return null
  }

  const content = (
    <HStack gap="1">
      <Box boxSize="2" bg={status.healthy ? "green.400" : "red.400"} rounded="full" />
      <Text color="gray.600" _dark={{ color: "gray.400" }} textTransform="capitalize">
        {status.healthy ? "Connected" : "Disconnected"}
      </Text>
    </HStack>
  )

  if (status.details && status.details.length > 0) {
    return (
      <Tooltip
        label={
          <List>
            {status.details.map((detail, i) => (
              <ListItem key={i}>{detail}</ListItem>
            ))}
            )
          </List>
        }>
        {content}
      </Tooltip>
    )
  }

  return content
}

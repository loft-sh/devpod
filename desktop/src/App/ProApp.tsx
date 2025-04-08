import { STATUS_BAR_HEIGHT } from "@/constants"
import { ProviderProvider } from "@/contexts/DevPodContext/DevPodProvider"
import { BellDuotone, CogDuotone, LockDuotone } from "@/icons"
import { TConnectionStatus, useConnectionStatus } from "@/lib"
import { QueryKeys } from "@/queryKeys"
import { Routes } from "@/routes"
import { TPlatformVersionInfo } from "@/types"
import {
  Avatar,
  Box,
  Button,
  Divider,
  HStack,
  IconButton,
  Link,
  List,
  ListItem,
  Popover,
  PopoverArrow,
  PopoverContent,
  PopoverHeader,
  PopoverTrigger,
  Portal,
  Text,
  Tooltip,
  useColorModeValue,
  useDisclosure,
} from "@chakra-ui/react"
import { ManagementV1Self } from "@loft-enterprise/client/gen/models/managementV1Self"
import { useQuery } from "@tanstack/react-query"
import { ReactElement, ReactNode, cloneElement, useMemo } from "react"
import { Outlet, Link as RouterLink, To } from "react-router-dom"
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
import { DaemonClient } from "@/client/pro/client"

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
  const { managementSelfQuery: selfQuery, client } = useProContext()
  const connectionStatus = useConnectionStatus()
  const versionInfo = usePlatformVersion()

  return (
    <ProLayout
      toolbarItems={
        <>
          <HStack gap="4">
            <Box>
              <Link variant="" to={Routes.toProInstance(host)} as={RouterLink}>
                <Toolbar.Title />
              </Link>
            </Box>
            <Box>
              <Toolbar.Actions />
            </Box>
          </HStack>
          <HStack pr="2">
            {client instanceof DaemonClient ? (
              <>
                <Notifications
                  getActionDestination={(action) => Routes.toProWorkspace(host, action.targetID)}
                  icon={
                    <BellDuotone
                      color={"primary.600"}
                      _dark={{ color: "primary.300" }}
                      position="absolute"
                    />
                  }
                />
                <Divider orientation="vertical" h="10" />
                <UserMenu host={host} self={selfQuery.data} />
              </>
            ) : (
              <>
                <Link as={RouterLink} to={Routes.toProSettings(host)}>
                  <IconButton
                    variant="ghost"
                    size="md"
                    rounded="full"
                    aria-label="Go to settings"
                    icon={<CogDuotone color={"primary.600"} _dark={{ color: "primary.300" }} />}
                  />
                </Link>
                <Notifications
                  getActionDestination={(action) => Routes.toProWorkspace(host, action.targetID)}
                  icon={
                    <BellDuotone
                      color={"primary.600"}
                      _dark={{ color: "primary.300" }}
                      position="absolute"
                    />
                  }
                />
              </>
            )}
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
      <Box
        boxSize="2"
        bg={status.healthy && status.online ? "green.400" : "red.400"}
        rounded="full"
      />
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

type TProfileMenuProps = Readonly<{
  host: string
  self: ManagementV1Self | undefined
}>
function UserMenu({ host, self }: TProfileMenuProps) {
  const iconColor = useColorModeValue("primary.600", "primary.300")
  const { isOpen, onClose, onToggle } = useDisclosure()

  const userName = self?.status?.user?.displayName ?? self?.status?.user?.name ?? "Anonymous"

  return (
    <>
      <Popover placement="bottom" isOpen={isOpen} onClose={onClose} returnFocusOnClose={false}>
        <PopoverTrigger>
          <IconButton
            onClick={onToggle}
            variant="ghost"
            size="md"
            rounded="full"
            aria-label="User Menu"
            icon={
              <Avatar
                name={userName}
                size="xs"
                fontWeight="bold"
                bg={iconColor}
                _dark={{ color: "gray.900" }}
                src={self?.status?.user?.icon}
              />
            }
          />
        </PopoverTrigger>
        <Portal>
          <PopoverContent zIndex="popover" w="72">
            <PopoverArrow />
            <PopoverHeader textOverflow="ellipsis" overflow="hidden" maxW="72" whiteSpace="nowrap">
              {userName}
            </PopoverHeader>
            <List my="4" onClick={onClose}>
              <ListItem>
                {/* TODO: Implement when we need it
                 UserLinkButton to={Routes.toProProfile(host)} icon={<ProfileDuotone />}>
                  Profile
                </UserLinkButton>*/}
              </ListItem>
              <ListItem>
                <UserLinkButton to={Routes.toProCredentials(host)} icon={<LockDuotone />}>
                  Credentials
                </UserLinkButton>
              </ListItem>
              <ListItem>
                <UserLinkButton to={Routes.toProSettings(host)} icon={<CogDuotone />}>
                  Settings
                </UserLinkButton>
              </ListItem>
            </List>
          </PopoverContent>
        </Portal>
      </Popover>
    </>
  )
}

type TUserLinkButton = Readonly<{ children: ReactNode; to: To; icon: ReactElement }>
function UserLinkButton({ children, to, icon }: TUserLinkButton) {
  const clonedIcon = cloneElement(icon, { boxSize: 5 })

  return (
    <Button
      as={RouterLink}
      size="sm"
      fontWeight="semibold"
      variant={"ghost"}
      w="full"
      display="flex"
      justifyContent="start"
      alignItems="center"
      leftIcon={clonedIcon}
      to={to}>
      {children}
    </Button>
  )
}

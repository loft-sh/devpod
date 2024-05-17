import {
  Center,
  IconButton,
  LinkBox,
  LinkOverlay,
  Popover,
  PopoverArrow,
  PopoverBody,
  PopoverContent,
  PopoverHeader,
  PopoverTrigger,
  Portal,
  Spinner,
  Text,
  Image,
  useColorModeValue,
  VStack,
  Box,
  Badge,
  HStack,
  Heading,
  Button,
  Link,
  Divider,
} from "@chakra-ui/react"
import dayjs from "dayjs"
import { useMemo } from "react"
import { Link as RouterLink, useLocation } from "react-router-dom"
import { useSettings, useAllWorkspaceActions, useProviders } from "../../contexts"
import { Bell, CheckCircle, ExclamationCircle, ExclamationTriangle, Stack3D } from "../../icons"
import { getActionDisplayName, useUpdate } from "../../lib"
import { Routes } from "../../routes"
import { Ripple } from "../Animation"
import { client } from "../../client"
import { QueryKeys } from "../../queryKeys"
import { useQuery } from "@tanstack/react-query"

export function Notifications() {
  const location = useLocation()
  const actions = useAllWorkspaceActions()
  const backgroundColor = useColorModeValue("white", "gray.900")
  const subheadingTextColor = useColorModeValue("gray.500", "gray.400")
  const actionHoverColor = useColorModeValue("gray.100", "gray.800")
  const hasActiveActions = actions.active.length > 0
  const settings = useSettings()
  const { pendingUpdate, install: installUpdate, isInstalling, isInstallDisabled } = useUpdate()
  const providerUpdateInfo = useProviderUpdates()
  const providerUpdateCount = providerUpdateInfo?.length ?? 0

  const combinedActions = useMemo(() => {
    return [...actions.active, ...actions.history]
  }, [actions.active, actions.history])

  return (
    <Popover placement="bottom">
      <PopoverTrigger>
        <Center marginRight="4">
          <IconButton
            variant="ghost"
            size="md"
            rounded="full"
            aria-label="Show onging operations"
            icon={
              <>
                <Bell boxSize={6} position="absolute" />
                {(pendingUpdate || providerUpdateCount !== 0) && (
                  <Badge
                    colorScheme="red"
                    position="absolute"
                    variant="solid"
                    bgColor="red.500"
                    borderRadius="full"
                    right="0"
                    top="0">
                    {pendingUpdate ? 1 + providerUpdateCount : providerUpdateCount}
                  </Badge>
                )}
                {hasActiveActions && <Ripple boxSize={10} />}
              </>
            }
          />
        </Center>
      </PopoverTrigger>
      <Portal>
        <Box width="full" height="full" zIndex="popover" position="relative">
          <PopoverContent backgroundColor={backgroundColor} zIndex="popover">
            <PopoverArrow backgroundColor={backgroundColor} />
            <PopoverHeader>Notifications</PopoverHeader>
            <PopoverBody overflow="hidden" maxHeight="20rem" paddingInline="0">
              {pendingUpdate && (
                <HStack
                  justifyContent="space-between"
                  paddingX="7"
                  paddingTop="2"
                  paddingBottom="3"
                  width="calc(100% + 1.5rem)"
                  transform={"translateX(-0.75rem)"}
                  borderBottomWidth="thin"
                  borderColor="inherit"
                  spacing="0">
                  <VStack align="start" spacing="0">
                    <Heading size="xs">{pendingUpdate.tag_name} is available</Heading>
                    <Text fontSize="xs">
                      See{" "}
                      <Link onClick={() => client.open(pendingUpdate.html_url)}>
                        release notes
                      </Link>
                    </Text>
                  </VStack>
                  <Button
                    isLoading={isInstalling}
                    isDisabled={isInstallDisabled}
                    onClick={() => installUpdate()}
                    variant="outline">
                    Install now
                  </Button>
                </HStack>
              )}
              <Box width="full" overflowY="auto" maxHeight="17rem" height="full" padding="1">
                {combinedActions.length === 0 && providerUpdateCount === 0 && (
                  <Text padding={2}>No notifications</Text>
                )}
                {providerUpdateInfo && providerUpdateCount > 0 && (
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
                          <Text color={subheadingTextColor} marginTop="-1">
                            Update available
                          </Text>
                        </VStack>
                      </LinkBox>
                    ))}
                    <Divider />
                  </>
                )}
                {combinedActions.map((action) => (
                  <LinkBox
                    key={action.id}
                    padding={2}
                    fontSize="sm"
                    borderRadius="md"
                    width="full"
                    display="flex"
                    flexFlow="row nowrap"
                    alignItems="center"
                    gap={3}
                    _hover={{ backgroundColor: actionHoverColor }}>
                    {action.status === "pending" ? (
                      settings.partyParrot ? (
                        <Image
                          width="6"
                          height="6"
                          src={"https://cdn3.emoji.gg/emojis/2747_PartyParrot.gif"}
                        />
                      ) : (
                        <Spinner color="blue.300" size="sm" />
                      )
                    ) : null}
                    {action.status === "success" && <CheckCircle color="green.300" />}
                    {action.status === "error" && <ExclamationCircle color="red.300" />}
                    {action.status === "cancelled" && <ExclamationTriangle color="orange.300" />}

                    <VStack align="start" spacing="0">
                      <Text fontWeight="bold">
                        <LinkOverlay
                          as={RouterLink}
                          to={Routes.toAction(action.id)}
                          state={{ origin: location.pathname }}
                          textTransform="capitalize">
                          {getActionDisplayName(action)}
                        </LinkOverlay>
                      </Text>
                      {action.finishedAt !== undefined && (
                        <Text color={subheadingTextColor} marginTop="-1">
                          {dayjs(action.finishedAt).fromNow()}
                        </Text>
                      )}
                    </VStack>
                  </LinkBox>
                ))}
              </Box>
            </PopoverBody>
          </PopoverContent>
        </Box>
      </Portal>
    </Popover>
  )
}

function useProviderUpdates() {
  const [[providers]] = useProviders()
  const { data: providerUpdateInfo } = useQuery({
    // eslint-disable-next-line @tanstack/query/exhaustive-deps
    queryKey: QueryKeys.PROVIDERS_CHECK_UPDATE_ALL,
    queryFn: async () => {
      console.log("Checking for updates", providers)
      if (providers === undefined || Object.keys(providers).length === 0) {
        return
      }

      const results = await Promise.allSettled(
        Object.keys(providers).map(async (p) => ({
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

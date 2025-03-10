import {
  Badge,
  Box,
  Button,
  Center,
  Divider,
  HStack,
  Heading,
  IconButton,
  Image,
  Link,
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
  VStack,
  useColorModeValue,
} from "@chakra-ui/react"
import dayjs from "dayjs"
import { JSX, ReactNode, useMemo } from "react"
import { Link as RouterLink, To, useLocation } from "react-router-dom"
import { client } from "../../client"
import { TActionObj, useAllWorkspaceActions, useSettings } from "../../contexts"
import { Bell, CheckCircle, ExclamationCircle, ExclamationTriangle } from "../../icons"
import { getActionDisplayName, useUpdate } from "../../lib"
import { Ripple } from "../Animation"

type TNotificationsProps = Readonly<{
  badgeNumber?: number
  providerUpdates?: ReactNode
  icon?: JSX.Element
  getActionDestination: (action: TActionObj) => To
}>
export function Notifications({
  icon,
  badgeNumber = 0,
  providerUpdates,
  getActionDestination,
}: TNotificationsProps) {
  const location = useLocation()
  const actions = useAllWorkspaceActions()
  const backgroundColor = useColorModeValue("white", "gray.900")
  const subheadingTextColor = useColorModeValue("gray.500", "gray.400")
  const actionHoverColor = useColorModeValue("gray.100", "gray.700")
  const hasActiveActions = actions.active.length > 0
  const settings = useSettings()
  const { pendingUpdate, install: installUpdate, isInstalling, isInstallDisabled } = useUpdate()

  const combinedActions = useMemo(() => {
    return [...actions.active, ...actions.history]
  }, [actions.active, actions.history])

  const maybeIconColor = useMemo(() => icon?.props.color, [icon])

  return (
    <Popover placement="bottom">
      <PopoverTrigger>
        <Center>
          <IconButton
            variant="ghost"
            size="md"
            rounded="full"
            aria-label="Show onging operations"
            {...(maybeIconColor ? { color: maybeIconColor } : {})}
            icon={
              <>
                {icon ? icon : <Bell boxSize={6} position="absolute" />}
                {(pendingUpdate || badgeNumber !== 0) && (
                  <Badge
                    colorScheme="red"
                    position="absolute"
                    variant="solid"
                    bgColor="red.500"
                    borderRadius="full"
                    right="0"
                    top="0">
                    {pendingUpdate ? 1 + badgeNumber : badgeNumber}
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
                      <Link onClick={() => client.open(pendingUpdate.html_url)}>release notes</Link>
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
                {combinedActions.length === 0 && badgeNumber === 0 && (
                  <Text padding={2}>No notifications</Text>
                )}

                {providerUpdates && (
                  <>
                    {providerUpdates}
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
                          to={getActionDestination(action)}
                          state={{ origin: location.pathname, actionID: action.id }}
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

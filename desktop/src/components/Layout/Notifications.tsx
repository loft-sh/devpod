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
  useColorModeValue,
  VStack,
} from "@chakra-ui/react"
import dayjs from "dayjs"
import { useMemo } from "react"
import { Link as RouterLink } from "react-router-dom"
import { useWorkspaceActions } from "../../contexts"
import { Bell, CheckCircle, ExclamationCircle, ExclamationTriangle } from "../../icons"
import { Routes } from "../../routes"
import { Ripple } from "../Animation"

export function Notifications() {
  const actions = useWorkspaceActions()
  const backgroundColor = useColorModeValue("white", "gray.900")
  const subheadingTextColor = useColorModeValue("gray.500", "gray.400")
  const actionHoverColor = useColorModeValue("gray.100", "gray.800")
  const hasActiveActions = actions.active.length > 0

  const combinedActions = useMemo(() => {
    return [...actions.active, ...actions.history]
  }, [actions.active, actions.history])

  return (
    <Popover placement="bottom">
      <PopoverTrigger>
        <Center>
          <IconButton
            variant="ghost"
            size="md"
            rounded="full"
            aria-label="Show onging operations"
            icon={
              <>
                <Bell boxSize={6} position="absolute" />
                {hasActiveActions && <Ripple boxSize={10} />}
              </>
            }
          />
        </Center>
      </PopoverTrigger>
      <Portal>
        <PopoverContent backgroundColor={backgroundColor}>
          <PopoverArrow backgroundColor={backgroundColor} />
          <PopoverHeader paddingY="4" fontWeight="bold">
            Notifications
          </PopoverHeader>
          <PopoverBody overflowY="scroll" maxHeight="20rem">
            {combinedActions.length === 0 && <Text>No notifications</Text>}
            {combinedActions.map((action) => (
              <LinkBox
                key={action.id}
                padding={2}
                borderRadius="md"
                width="full"
                display="flex"
                flexFlow="row nowrap"
                alignItems="center"
                gap={3}
                _hover={{ backgroundColor: actionHoverColor }}>
                {action.status === "pending" && <Spinner color="blue.300" size="sm" />}
                {action.status === "success" && <CheckCircle color="green.300" />}
                {action.status === "error" && <ExclamationCircle color="red.300" />}
                {action.status === "cancelled" && <ExclamationTriangle color="orange.300" />}

                <VStack align="start">
                  <Text fontWeight="bold">
                    <LinkOverlay
                      as={RouterLink}
                      to={Routes.toWorkspace(action.workspaceID, action.id)}>
                      {action.name} {action.workspaceID}
                    </LinkOverlay>
                  </Text>
                  {action.finishedAt !== undefined && (
                    <Text marginTop={"0 !important"} color={subheadingTextColor}>
                      {dayjs(action.finishedAt).fromNow()}
                    </Text>
                  )}
                </VStack>
              </LinkBox>
            ))}
          </PopoverBody>
        </PopoverContent>
      </Portal>
    </Popover>
  )
}

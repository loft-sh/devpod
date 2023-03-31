import { BellIcon, CheckCircleIcon, WarningIcon } from "@chakra-ui/icons"
import {
  Center,
  Icon,
  IconButton,
  IconProps,
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
import { motion } from "framer-motion"
import { useMemo } from "react"
import { Link as RouterLink } from "react-router-dom"
import { useWorkspaceActions } from "../../contexts"
import { Routes } from "../../routes"

export function Notifications() {
  const actions = useWorkspaceActions()
  const subheadingTextColor = useColorModeValue("gray.500", "gray.400")
  const actionHoverColor = useColorModeValue("gray.100", "gray.800")
  const hasActiveActions = actions.active.length > 0

  const combinedActions = useMemo(() => {
    return [...actions.active, ...actions.history]
  }, [actions.active, actions.history])

  return (
    <Popover placement="top">
      <PopoverTrigger>
        <Center>
          <IconButton
            variant="solid"
            size="md"
            rounded="full"
            color="gray.500"
            aria-label="Show onging operations"
            icon={
              <>
                <BellIcon boxSize={6} position="absolute" />
                {hasActiveActions && <Pulse boxSize={10} />}
              </>
            }
          />
        </Center>
      </PopoverTrigger>
      <Portal>
        <PopoverContent>
          <PopoverArrow />
          <PopoverHeader>Notifications</PopoverHeader>
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
                {action.status === "success" && <CheckCircleIcon color="green.300" />}
                {action.status === "error" && <WarningIcon color="red.300" />}
                {action.status === "cancelled" && <WarningIcon color="orange.300" />}

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

const initial = {
  r: 4,
  opacity: 0.3,
}
const animate = { r: 12, opacity: 0 }
const transition = { duration: 4, repeat: Infinity }
function Pulse(props: IconProps) {
  return (
    <Icon {...props} viewBox="0 0 24 24">
      <motion.circle
        cx="12"
        cy="12"
        fill="currentColor"
        initial={initial}
        animate={animate}
        transition={transition}
      />
      <motion.circle
        cx="12"
        cy="12"
        fill="currentColor"
        initial={initial}
        animate={animate}
        transition={{ ...transition, delay: 1 }}
      />
      <motion.circle
        cx="12"
        cy="12"
        initial={initial}
        animate={animate}
        transition={{ ...transition, delay: 2 }}
      />
    </Icon>
  )
}

import { useStreamingTerminal } from "@/components"
import { useAction } from "@/contexts"
import { useWorkspaceActions } from "@/contexts/DevPodContext/workspaces/useWorkspace"
import { CheckCircle, ExclamationCircle, ExclamationTriangle } from "@/icons"
import { exists, useDownloadLogs } from "@/lib"
import { Routes } from "@/routes"
import { DownloadIcon } from "@chakra-ui/icons"
import {
  Accordion,
  AccordionButton,
  AccordionIcon,
  AccordionItem,
  AccordionPanel,
  Box,
  Button,
  HStack,
  IconButton,
  LinkBox,
  LinkOverlay,
  Spinner,
  Text,
  Tooltip,
  VStack,
} from "@chakra-ui/react"
import dayjs from "dayjs"
import { useEffect, useState } from "react"
import { HiStop } from "react-icons/hi"
import { Link as RouterLink, useLocation } from "react-router-dom"
import { TTabProps } from "./types"

export function Logs({ host, instance }: TTabProps) {
  const [accordionIndex, setAccordionIndex] = useState<number>(0)
  const actions = useWorkspaceActions(instance.id)

  const location = useLocation()

  useEffect(() => {
    // NOTE: It's important to use `exists` here as actionID could be 0
    if (exists(location.state?.actionID) && actions && actions.length > 0) {
      const maybeActionIdx = actions.findIndex((action) => action.id === location.state.actionID)
      if (!exists(maybeActionIdx)) {
        return
      }
      setAccordionIndex(maybeActionIdx)
    }
  }, [actions, location.state?.actionID])

  return (
    <VStack align="start" w="full">
      <Accordion
        w="full"
        allowToggle
        index={accordionIndex}
        onChange={(idx) => setAccordionIndex(idx as number)}>
        {actions?.map((action) => (
          <AccordionItem key={action.id} w="full">
            {({ isExpanded }) => (
              <ActionAccordionItem
                actionID={action.id}
                isExpanded={isExpanded}
                host={host}
                instanceID={instance.id}
              />
            )}
          </AccordionItem>
        ))}
      </Accordion>
    </VStack>
  )
}

type TActionAccordionItemProps = Readonly<{
  actionID: string
  isExpanded: boolean
  host: string
  instanceID: string
}>
function ActionAccordionItem({
  host,
  instanceID,
  actionID,
  isExpanded,
}: TActionAccordionItemProps) {
  const action = useAction(actionID)

  return action?.data ? (
    <>
      <h2>
        <AccordionButton
          as={LinkBox}
          w="full"
          display="flex"
          alignItems="center"
          gap="2"
          padding={2}
          borderRadius="md"
          width="full"
          flexFlow="row nowrap">
          {action.data.status === "pending" && <Spinner color="blue.300" size="sm" />}
          {action.data.status === "success" && <CheckCircle color="green.300" boxSize="5" />}
          {action.data.status === "error" && <ExclamationCircle color="red.300" boxSize="5" />}
          {action.data.status === "cancelled" && (
            <ExclamationTriangle color="orange.300" boxSize="5" />
          )}

          <LinkOverlay
            as={RouterLink}
            to={Routes.toProWorkspaceDetail(host, instanceID, "logs")}
            fontWeight="semibold"
            textTransform="capitalize"
            state={{ origin: location.pathname, actionID: actionID }}>
            {action.data.name}
          </LinkOverlay>

          <Tooltip label={dayjs(action.data.createdAt).format()}>
            <Text color="gray.600">{dayjs(action.data.createdAt).fromNow()}</Text>
          </Tooltip>

          {action.data.status === "pending" && (
            <Button
              variant="outline"
              aria-label="Cancel action"
              leftIcon={<HiStop />}
              onClick={(e) => {
                e.stopPropagation()
                action.cancel()
              }}>
              Cancel
            </Button>
          )}

          <HStack ml="auto">
            {action.data.status !== "pending" && <DownloadLogsButton actionID={actionID} />}
            <AccordionIcon />
          </HStack>
        </AccordionButton>
      </h2>
      <AccordionPanel>{isExpanded && <ActionTerminal actionID={actionID} />}</AccordionPanel>
    </>
  ) : null
}
type TActionTerminalProps = Readonly<{
  actionID: string
}>
function ActionTerminal({ actionID }: TActionTerminalProps) {
  const action = useAction(actionID)
  const { terminal, connectStream, clear: clearTerminal } = useStreamingTerminal()

  useEffect(() => {
    clearTerminal()

    return action?.connectOrReplay(connectStream)
  }, [action, clearTerminal, connectStream])

  return (
    <Box h="50vh" w="full" mb="8">
      {terminal}
    </Box>
  )
}

type TDownloadLogsButtonProps = Readonly<{ actionID: string }>
function DownloadLogsButton({ actionID }: TDownloadLogsButtonProps) {
  const { download, isDownloading } = useDownloadLogs()

  return (
    <Tooltip label="Save Logs">
      <IconButton
        ml="auto"
        mr="4"
        isLoading={isDownloading}
        title="Save Logs"
        variant="ghost"
        aria-label="Save Logs"
        icon={<DownloadIcon />}
        onClick={(e) => {
          e.stopPropagation()
          download({ actionID })
        }}
      />
    </Tooltip>
  )
}

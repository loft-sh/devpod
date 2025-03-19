import { TerminalSearchBar, useStreamingTerminal } from "@/components"
import { TSearchOptions } from "@/components/Terminal/useTerminalSearch"
import { useAction } from "@/contexts"
import { useWorkspaceActions } from "@/contexts/DevPodContext/workspaces/useWorkspace"
import { CheckCircle, ExclamationCircle, ExclamationTriangle } from "@/icons"
import EmptyImage from "@/images/empty_default.svg"
import EmptyDarkImage from "@/images/empty_default_dark.svg"
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
  Heading,
  IconButton,
  Image,
  LinkBox,
  LinkOverlay,
  Spinner,
  Text,
  Tooltip,
  VStack,
  useColorMode,
  useColorModeValue,
} from "@chakra-ui/react"
import dayjs from "dayjs"
import { useEffect, useMemo, useState } from "react"
import { HiStop } from "react-icons/hi"
import { Link as RouterLink, useLocation } from "react-router-dom"
import { TTabProps } from "./types"

const MAX_LOGS = 10

export function Logs({ host, instance }: TTabProps) {
  const [accordionIndex, setAccordionIndex] = useState<number>(0)
  const workspaceActions = useWorkspaceActions(instance.id)
  const actions = useMemo(() => {
    return workspaceActions?.slice(0, MAX_LOGS)
  }, [workspaceActions])
  const { colorMode } = useColorMode()

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
    <VStack align="start" w="full" h={"full"}>
      {!actions?.length ? (
        <VStack h={"full"} w={"full"} justifyContent={"center"} alignItems={"center"} flexGrow={1}>
          <Image src={colorMode == "dark" ? EmptyDarkImage : EmptyImage} />
          <Text
            fontWeight={"semibold"}
            fontSize={"sm"}
            color={"gray.600"}
            _dark={{ color: "gray.300" }}>
            No logs to show yet
          </Text>
        </VStack>
      ) : (
        <Accordion
          w="full"
          allowToggle
          index={accordionIndex}
          onChange={(idx) => setAccordionIndex(idx as number)}>
          {actions.map((action) => (
            <AccordionItem mb={"2"} key={action.id} w="full" border={"none"}>
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
      )}
    </VStack>
  )
}

type TActionAccordionItemProps = Readonly<{
  host: string
  actionID: string
  isExpanded: boolean
  instanceID: string
}>
function ActionAccordionItem({
  host,
  instanceID,
  actionID,
  isExpanded,
}: TActionAccordionItemProps) {
  const action = useAction(actionID)
  const bgColor = useColorModeValue("white", "background.darkest")

  const handleCancelClicked = () => {
    action?.cancel()
  }

  return action?.data ? (
    <>
      <Heading as={"h2"}>
        <AccordionButton
          as={LinkBox}
          w="full"
          display="flex"
          alignItems="center"
          gap="2"
          paddingY={2}
          paddingX={3}
          borderWidth="thin"
          borderRadius="md"
          boxSizing={"border-box"}
          borderBottomRadius={isExpanded ? 0 : undefined}
          backgroundColor={bgColor}
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
            <Text fontWeight="normal" color="gray.600" _dark={{ color: "gray.400" }}>
              {dayjs(action.data.createdAt).fromNow()}
            </Text>
          </Tooltip>

          {action.data.status === "pending" && (
            <Button
              variant="outline"
              aria-label="Cancel action"
              leftIcon={<HiStop />}
              onClick={(e) => {
                e.stopPropagation()
                handleCancelClicked()
              }}>
              Cancel
            </Button>
          )}

          <HStack ml="auto">
            {action.data.status !== "pending" && <DownloadLogsButton actionID={actionID} />}
            <AccordionIcon />
          </HStack>
        </AccordionButton>
      </Heading>
      <AccordionPanel
        bgColor={bgColor}
        borderWidth={isExpanded ? "thin" : "unset"}
        borderTop={"none"}
        borderBottomRadius={"md"}
        padding={0}>
        {isExpanded && <ActionTerminal actionID={actionID} />}
      </AccordionPanel>
    </>
  ) : null
}
type TActionTerminalProps = Readonly<{
  actionID: string
}>
function ActionTerminal({ actionID }: TActionTerminalProps) {
  const action = useAction(actionID)

  const [searchOptions, setSearchOptions] = useState<TSearchOptions>({})

  const {
    terminal,
    connectStream,
    clear: clearTerminal,
    search: { totalSearchResults, nextSearchResult, prevSearchResult, activeSearchResult },
  } = useStreamingTerminal({ searchOptions, borderRadius: "none" })

  useEffect(() => {
    clearTerminal()

    return action?.connectOrReplay((e) => {
      connectStream(e)
    })
  }, [action, clearTerminal, connectStream])

  return (
    <VStack w={"full"}>
      <TerminalSearchBar
        paddingX={4}
        paddingY={3}
        prevSearchResult={prevSearchResult}
        nextSearchResult={nextSearchResult}
        totalSearchResults={totalSearchResults}
        activeSearchResult={activeSearchResult}
        onUpdateSearchOptions={setSearchOptions}
      />

      <Box h="50vh" w="full" mb="4">
        {terminal}
      </Box>
    </VStack>
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

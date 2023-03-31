import { Box, Button, HStack, Text, VStack } from "@chakra-ui/react"
import { useCallback, useEffect, useMemo } from "react"
import { useSearchParams, useNavigate, useParams } from "react-router-dom"
import { ErrorMessageBox, useStreamingTerminal } from "../../components"
import { useWorkspace } from "../../contexts"
import { exists, isError } from "../../lib"
import { Routes } from "../../routes"

export function Workspace() {
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()
  const params = useParams()
  const workspaceID = useMemo(() => Routes.getWorkspaceId(params), [params])
  const actionID = useMemo(() => Routes.getActionIDFromSearchParams(searchParams), [searchParams])
  const { terminal, connectStream } = useStreamingTerminal()

  const workspace = useWorkspace(workspaceID)

  const handleStartClicked = useCallback(async () => {
    if (!exists(workspace.data)) {
      return
    }

    workspace.start({ ideConfig: workspace.data.ide }, connectStream)
  }, [connectStream, workspace])

  // Wire up terminal to current action
  useEffect(() => {
    if (workspace.current === undefined) {
      return
    }

    workspace.current.connect(connectStream)
  }, [connectStream, workspace])

  // Listen to search param actionID
  useEffect(() => {
    if (actionID === undefined || workspace.isLoading) {
      return
    }

    setSearchParams() // Clear search params
    workspace.history.replay(actionID, connectStream)
  }, [actionID, connectStream, setSearchParams, workspace.history, workspace.isLoading])

  // Navigate to workspaces when workspace is deleted
  useEffect(() => {
    if (workspace.current?.name !== "remove" || workspace.current.status !== "success") {
      return
    }

    navigate(Routes.WORKSPACES)
  }, [navigate, workspace])

  const maybeError = workspace.current?.error
  if (isError(maybeError)) {
    return (
      <>
        <Text>Whoops, something went wrong</Text>
        <ErrorMessageBox error={maybeError} />
      </>
    )
  }
  if (workspace.data === undefined) {
    return null
  }

  return (
    <>
      <HStack marginTop="-6">
        <Button onClick={handleStartClicked} isLoading={workspace.current?.name === "start"}>
          Start
        </Button>
        <Button
          onClick={() => workspace.stop(connectStream)}
          isLoading={workspace.current?.name === "stop"}>
          Stop
        </Button>
        <Button
          onClick={() => workspace.rebuild(connectStream)}
          isLoading={workspace.current?.name === "rebuild"}>
          Rebuild
        </Button>
        <Button
          colorScheme="red"
          onClick={() => workspace.remove(connectStream)}
          isLoading={workspace.current?.name === "remove"}>
          Delete
        </Button>
      </HStack>

      <VStack align="start" marginTop="10">
        <Text>Status: {workspace.data.status}</Text>
        <Text>Source: {workspace.data.source?.localFolder ?? "unknown"}</Text>
        <Text>Provider: {workspace.data.provider?.name ?? "unknown"}</Text>
        <Text>IDE: {workspace.data.ide?.ide ?? "unknown"}</Text>
      </VStack>

      <Box minHeight="60" maxHeight="2xl" minWidth="sm" maxWidth="100%">
        {terminal}
      </Box>
    </>
  )
}

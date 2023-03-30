import { Box, Button, HStack, Text, VStack } from "@chakra-ui/react"
import { useCallback, useEffect, useMemo } from "react"
import { useParams } from "react-router"
import { ErrorMessageBox, useStreamingTerminal } from "../../components"
import { useWorkspace } from "../../contexts"
import { exists, isError } from "../../lib"
import { Routes } from "../../routes"

export function Workspace() {
  const params = useParams()
  const workspaceID = useMemo(() => Routes.getWorkspaceId(params), [params])
  const { terminal, connectStream } = useStreamingTerminal()

  const workspace = useWorkspace(workspaceID)

  const handleStartClicked = useCallback(async () => {
    if (!exists(workspace.data)) {
      return
    }

    workspace.start({ ideConfig: workspace.data.ide }, connectStream)
  }, [connectStream, workspace])

  useEffect(() => {
    if (!exists(workspace)) {
      return
    }

    // TODO: implement
    // workspace.current.connect(connectStream)
  }, [connectStream, workspace])

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
        <Button onClick={() => workspace.stop()} isLoading={workspace.current?.name === "stop"}>
          Stop
        </Button>
        <Button
          onClick={() => workspace.rebuild()}
          isLoading={workspace.current?.name === "rebuild"}>
          Rebuild
        </Button>
        <Button
          colorScheme="red"
          onClick={() => workspace.remove()}
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

import { Box, Button, CircularProgress, HStack, Text, VStack } from "@chakra-ui/react"
import { useCallback, useEffect, useMemo } from "react"
import { useParams } from "react-router"
import { useStreamingTerminal } from "../../components"
import { useWorkspace } from "../../contexts"
import { exists, isError } from "../../lib"
import { Routes } from "../../routes"

export function Workspace() {
  const params = useParams()
  const workspaceID = useMemo(() => Routes.getWorkspaceId(params), [params])
  const { terminal, connectStream } = useStreamingTerminal()

  // TODO: add "global" operation status
  const [[workspace, { status, error }], { start, stop, remove, rebuild }] =
    useWorkspace(workspaceID)

  const handleStartClicked = useCallback(async () => {
    if (!exists(workspace)) {
      return
    }

    start.run({
      workspaceID: workspace.id,
      config: { ideConfig: workspace.ide },
      onStream: connectStream,
    })
  }, [connectStream, start, workspace])

  useEffect(() => {
    if (!exists(workspaceID)) {
      return
    }

    start.connect({
      workspaceID,
      onStream: connectStream,
    })
  }, [workspaceID, connectStream, start])

  if (status === "loading") {
    return <CircularProgress isIndeterminate />
  }

  if (isError(error)) {
    return (
      <>
        <Text>Whoops, something went wrong</Text>
        <Box backgroundColor="red.100" marginTop="4" padding="4" borderRadius="md">
          <Text color="red.700">{error.message}</Text>
        </Box>
      </>
    )
  }

  if (workspace === undefined) {
    return null
  }

  return (
    <>
      <HStack marginTop="-6">
        <Button onClick={handleStartClicked} isLoading={start.status === "loading"}>
          Start
        </Button>
        <Button
          onClick={() => stop.run({ workspaceID: workspace.id })}
          isLoading={stop.status === "loading"}>
          Stop
        </Button>
        <Button
          onClick={() => rebuild.run({ workspaceID: workspace.id })}
          isLoading={rebuild.status === "loading"}>
          Rebuild
        </Button>
        <Button
          colorScheme="red"
          onClick={() => remove.run({ workspaceID: workspace.id })}
          isLoading={remove.status === "loading"}>
          Delete
        </Button>
      </HStack>

      <VStack align="start" marginTop="10">
        <Text>Status: {workspace.status}</Text>
        <Text>Source: {workspace.source?.localFolder ?? "unknown"}</Text>
        <Text>Provider: {workspace.provider?.name ?? "unknown"}</Text>
        <Text>IDE: {workspace.ide?.ide ?? "unknown"}</Text>
      </VStack>

      <Box height="60" minWidth="sm" maxWidth="100%">
        {terminal}
      </Box>
    </>
  )
}

import { Box, IconButton, Text, VStack } from "@chakra-ui/react"
import { useCallback, useEffect, useMemo } from "react"
import { useNavigate, useParams, useSearchParams } from "react-router-dom"
import { ErrorMessageBox, ToolbarActions, useStreamingTerminal } from "../../components"
import { useWorkspace } from "../../contexts"
import { Pause, Play, Trash } from "../../icons"
import { ArrowPath } from "../../icons/ArrowPath"
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

    workspace.start({ id: workspaceID!, ideConfig: workspace.data.ide }, connectStream)
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
      <ToolbarActions>
        {workspace.data.status === "Running" ? (
          <IconButton
            aria-label="Stop workspace"
            variant="ghost"
            icon={<Pause />}
            onClick={() => workspace.stop(connectStream)}
            isLoading={workspace.current?.name === "stop"}
          />
        ) : (
          <IconButton
            aria-label="Start workspace"
            variant="ghost"
            icon={<Play />}
            onClick={handleStartClicked}
            isLoading={workspace.current?.name === "start"}
          />
        )}

        <IconButton
          aria-label="Rebuild workspace"
          variant="ghost"
          icon={<ArrowPath />}
          onClick={() => workspace.rebuild(connectStream)}
          isLoading={workspace.current?.name === "rebuild"}
        />
        <IconButton
          colorScheme="red"
          aria-label="Remove workspace"
          variant="ghost"
          icon={<Trash />}
          onClick={() => workspace.remove(connectStream)}
          isLoading={workspace.current?.name === "remove"}
        />
      </ToolbarActions>
      <Box minHeight="60" maxHeight="2xl" minWidth="sm" maxWidth="100%">
        {terminal}
      </Box>
    </>
  )
}

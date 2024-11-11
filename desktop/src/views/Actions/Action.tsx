import { useDownloadLogs } from "@/lib"
import { DownloadIcon } from "@chakra-ui/icons"
import { Box, Button, IconButton, Tooltip } from "@chakra-ui/react"
import { useEffect, useMemo } from "react"
import { HiStop } from "react-icons/hi2"
import { useNavigate } from "react-router"
import { useParams, useSearchParams } from "react-router-dom"
import { ToolbarActions, useStreamingTerminal } from "../../components"
import { useAction } from "../../contexts"
import { Routes } from "../../routes"

export function Action() {
  const [searchParams] = useSearchParams()
  const params = useParams()
  const navigate = useNavigate()
  const actionID = useMemo(() => Routes.getActionID(params), [params])
  const action = useAction(actionID)
  const { terminal, connectStream, clear } = useStreamingTerminal()
  const { download, isDownloading: isDownloadingLogs } = useDownloadLogs()

  useEffect(() => {
    if (action === undefined) {
      return
    }

    clear()

    return action.connectOrReplay(connectStream)
  }, [action, actionID, clear, connectStream])

  useEffect(() => {
    const onSuccess = searchParams.get("onSuccess")
    if (onSuccess && action?.data.status === "success") {
      navigate(onSuccess)
    }
  }, [searchParams, action, navigate])

  return (
    <>
      <ToolbarActions>
        {action?.data.status === "pending" && (
          <Button
            variant="outline"
            aria-label="Cancel action"
            leftIcon={<HiStop />}
            onClick={() => action.cancel()}>
            Cancel
          </Button>
        )}
        {actionID !== undefined && (
          <Tooltip label="Save Logs">
            <IconButton
              isLoading={isDownloadingLogs}
              title="Save Logs"
              variant="outline"
              aria-label="Save Logs"
              icon={<DownloadIcon />}
              onClick={() => download({ actionID })}
            />
          </Tooltip>
        )}
      </ToolbarActions>
      <Box height="calc(100% - 3rem)" width="full">
        {terminal}
      </Box>
    </>
  )
}

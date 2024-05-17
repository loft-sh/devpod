import { Box, Button, IconButton, Tooltip, useToast } from "@chakra-ui/react"
import { useEffect, useMemo } from "react"
import { HiStop } from "react-icons/hi2"
import { useNavigate } from "react-router"
import { useParams, useSearchParams } from "react-router-dom"
import { ToolbarActions, useStreamingTerminal } from "../../components"
import { TActionID, useAction } from "../../contexts"
import { Routes } from "../../routes"
import { DownloadIcon } from "@chakra-ui/icons"
import { client } from "@/client"
import { useMutation } from "@tanstack/react-query"
import { dialog } from "@tauri-apps/api"

export function Action() {
  const [searchParams] = useSearchParams()
  const params = useParams()
  const navigate = useNavigate()
  const actionID = useMemo(() => Routes.getActionID(params), [params])
  const action = useAction(actionID)
  const { terminal, connectStream, clear } = useStreamingTerminal()

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

  const toast = useToast()
  const { mutate: handleDownloadLogsClicked, isLoading: isDownloadingLogs } = useMutation({
    mutationFn: async ({ actionID }: { actionID: TActionID }) => {
      const actionLogFile = (await client.workspaces.getActionLogFile(actionID)).unwrap()

      if (actionLogFile === undefined) {
        throw new Error(`Unable to retrieve file for action ${actionID}`)
      }

      const targetFile = await dialog.save({
        title: "Save Logs",
        filters: [{ name: "format", extensions: ["log", "txt"] }],
      })

      // user cancelled "save file" dialog
      if (targetFile === null) {
        return
      }

      await client.copyFile(actionLogFile, targetFile)
      client.open(targetFile)
    },
    onError(error) {
      toast({
        title: `Failed to save logs: ${error}`,
        status: "error",
        isClosable: true,
        duration: 30_000, // 30 sec
      })
    },
  })

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
              onClick={() => handleDownloadLogsClicked({ actionID })}
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

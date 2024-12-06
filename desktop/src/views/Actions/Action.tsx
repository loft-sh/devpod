import { useDownloadLogs } from "@/lib"
import { DownloadIcon } from "@chakra-ui/icons"
import { Box, Button, IconButton, Tooltip } from "@chakra-ui/react"
import { useEffect, useMemo, useState } from "react"
import { HiStop } from "react-icons/hi2"
import { useNavigate } from "react-router"
import { useParams, useSearchParams } from "react-router-dom"
import { TerminalSearchBar, ToolbarActions, useStreamingTerminal } from "@/components"
import { useAction } from "@/contexts"
import { Routes } from "@/routes"
import { TSearchOptions } from "@/components/Terminal/useTerminalSearch"

export function Action() {
  const [searchParams] = useSearchParams()
  const params = useParams()
  const navigate = useNavigate()
  const actionID = useMemo(() => Routes.getActionID(params), [params])
  const action = useAction(actionID)

  const [searchOptions, setSearchOptions] = useState<TSearchOptions>({})

  const {
    terminal,
    connectStream,
    clear,
    search: { totalSearchResults, nextSearchResult, prevSearchResult, activeSearchResult },
  } = useStreamingTerminal({ searchOptions })

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
      <TerminalSearchBar
        prevSearchResult={prevSearchResult}
        nextSearchResult={nextSearchResult}
        totalSearchResults={totalSearchResults}
        activeSearchResult={activeSearchResult}
        onUpdateSearchOptions={setSearchOptions}
      />
      <Box height="calc(100% - 8rem)" mt={8} width="full">
        {terminal}
      </Box>
    </>
  )
}

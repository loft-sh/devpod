import { Box } from "@chakra-ui/react"
import { useEffect, useMemo } from "react"
import { useParams } from "react-router-dom"
import { useStreamingTerminal } from "../../components"
import { useAction } from "../../contexts"
import { Routes } from "../../routes"

export function Action() {
  const params = useParams()
  const actionID = useMemo(() => Routes.getActionID(params), [params])
  const action = useAction(actionID)
  const { terminal, connectStream, clear } = useStreamingTerminal()

  useEffect(() => {
    if (action === undefined) {
      return
    }

    action.connectOrReplay(connectStream)
  }, [connectStream, action])

  // Clear terminal when actionID changes
  useEffect(() => {
    clear()
  }, [actionID, clear])

  return (
    <>
      <Box height="calc(100% - 3rem)" width="full">
        {terminal}
      </Box>
    </>
  )
}

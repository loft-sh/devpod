import { Box } from "@chakra-ui/react"
import { useEffect, useMemo, useRef } from "react"
import { useParams, useSearchParams } from "react-router-dom"
import { useStreamingTerminal } from "../../components"
import { useAction } from "../../contexts"
import { Routes } from "../../routes"
import { useNavigate } from "react-router"

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
  }, [actionID, connectStream])

  useEffect(() => {
    const onSuccess = searchParams.get("onSuccess")
    if (onSuccess && action?.data.status === "success") {
      navigate(onSuccess)
    }
  }, [searchParams, action])

  return (
    <Box height="calc(100% - 3rem)" width="full">
      {terminal}
    </Box>
  )
}

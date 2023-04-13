import { Box } from "@chakra-ui/react"
import { useQueryClient } from "@tanstack/react-query"
import { useNavigate } from "react-router-dom"
import { QueryKeys } from "../../../queryKeys"
import { Routes } from "../../../routes"
import { SetupProviderSteps } from "./SetupProviderSteps"

export function AddProvider() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()

  return (
    <Box paddingBottom={80}>
      <SetupProviderSteps
        onFinish={() => {
          navigate(Routes.PROVIDERS)
          queryClient.invalidateQueries(QueryKeys.PROVIDERS)
        }}
      />
    </Box>
  )
}

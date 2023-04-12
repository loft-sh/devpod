import { Box } from "@chakra-ui/react"
import { useNavigate } from "react-router-dom"
import { Routes } from "../../../routes"
import { SetupProviderSteps } from "./SetupProviderSteps"

export function AddProvider() {
  const navigate = useNavigate()

  return (
    <Box paddingBottom={80}>
      <SetupProviderSteps onFinish={() => navigate(Routes.PROVIDERS)} />
    </Box>
  )
}

import { Box } from "@chakra-ui/react"
import { Outlet } from "react-router"
import { NavigationViewLayout } from "../../components"
import { useWorkspaceTitle } from "./useWorkspaceTitle"

export function Workspaces() {
  const title = useWorkspaceTitle()

  return (
    <NavigationViewLayout title={title}>
      <Box paddingTop="10">
        <Outlet />
      </Box>
    </NavigationViewLayout>
  )
}

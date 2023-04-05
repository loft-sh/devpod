import { Box } from "@chakra-ui/react"
import { Outlet } from "react-router-dom"
import { NavigationViewLayout } from "../../components"
import { useActionTitle } from "./useActionTitle"

export function Actions() {
  const title = useActionTitle()

  return (
    <NavigationViewLayout title={title}>
      <Box height="full" width="full">
        <Outlet />
      </Box>
    </NavigationViewLayout>
  )
}

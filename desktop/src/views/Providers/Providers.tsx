import { Box } from "@chakra-ui/react"
import { Outlet } from "react-router"
import { NavigationViewLayout } from "../../components"
import { useProviderTitle } from "./useProviderTitle"

export function Providers() {
  const title = useProviderTitle()

  return (
    <>
      <NavigationViewLayout title={title}>
        <Box paddingTop="10" height="full">
          <Outlet />
        </Box>
      </NavigationViewLayout>
    </>
  )
}

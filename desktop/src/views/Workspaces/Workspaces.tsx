import { Box, Heading, HStack } from "@chakra-ui/react"
import { Outlet } from "react-router"
import { exists } from "../../lib"
import { useWorkspaceTitle } from "./useWorkspaceTitle"

export function Workspaces() {
  const title = useWorkspaceTitle()

  return (
    <>
      {exists(title) && (
        <HStack align="center">
          {exists(title.leadingAction) && title.leadingAction}
          <Heading as={title.priority === "high" ? "h1" : "h2"} size="xl">
            {title.label}
          </Heading>
          {exists(title.trailingAction) && title.trailingAction}
        </HStack>
      )}

      <Box paddingTop="10">
        <Outlet />
      </Box>
    </>
  )
}

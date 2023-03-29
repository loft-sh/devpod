import { Button, Card, CardBody, CardFooter, Heading, Link, Stack, Text } from "@chakra-ui/react"
import { Link as RouterLink } from "react-router-dom"
import { Routes } from "../../routes"
import { TWorkspace } from "../../types"
import React from "react"
import { useWorkspaceManager } from "../../contexts"

type TWorkspaceProps = {
  workspace: TWorkspace
}

export function Workspace({ workspace }: TWorkspaceProps) {
  const { start, stop, remove } = useWorkspaceManager()

  return (
    <Card
      key={workspace.id}
      direction={{ base: "column", sm: "row" }}
      overflow="hidden"
      variant="outline">
      <Stack>
        <CardBody>
          <Heading size="md">
            <Link as={RouterLink} to={Routes.toWorkspace(workspace.id)}>
              <Text fontWeight="bold">{workspace.id}</Text>
            </Link>
          </Heading>

          {workspace.provider?.name && <Text>Provider: {workspace.provider.name}</Text>}
          <Text>Status: {workspace.status}</Text>
        </CardBody>

        <CardFooter>
          <Button
            colorScheme="primary"
            onClick={() =>
              start.run({ workspaceID: workspace.id, config: { ideConfig: workspace.ide } })
            }
            isLoading={start.status === "loading"}>
            Start
          </Button>
          <Button
            onClick={() => stop.run({ workspaceID: workspace.id })}
            isLoading={stop.status === "loading"}>
            Stop
          </Button>
          <Button
            colorScheme="red"
            onClick={() => remove.run({ workspaceID: workspace.id })}
            isLoading={remove.status === "loading"}>
            Delete
          </Button>
        </CardFooter>
      </Stack>
    </Card>
  )
}

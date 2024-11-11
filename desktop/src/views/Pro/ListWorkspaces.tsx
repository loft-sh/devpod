import { ProWorkspaceInstance, useProContext, useWorkspaces } from "@/contexts"
import { DevPodIcon } from "@/icons"
import emptyWorkspacesImage from "@/images/empty_workspaces.svg"
import { Routes } from "@/routes"
import {
  Button,
  Center,
  Container,
  HStack,
  Heading,
  Image,
  List,
  ListItem,
  Spinner,
  Text,
  VStack,
} from "@chakra-ui/react"
import { useNavigate } from "react-router"
import { WorkspaceInstanceCard } from "./Workspace"

export function ListWorkspaces() {
  const instances = useWorkspaces<ProWorkspaceInstance>()
  const { host, isLoadingWorkspaces } = useProContext()
  const navigate = useNavigate()

  const handleCreateClicked = () => {
    navigate(Routes.toProWorkspaceCreate(host))
  }

  const hasWorkspaces = instances.length > 0

  return (
    <VStack align="start" gap="4" w="full" h="full">
      {hasWorkspaces ? (
        <>
          <HStack align="center" justify="space-between" mb="8" w="full">
            <Heading fontWeight="thin">Workspaces</Heading>
            <Button
              variant="outline"
              colorScheme="primary"
              leftIcon={<DevPodIcon boxSize={5} />}
              onClick={handleCreateClicked}>
              Create Workspace
            </Button>
          </HStack>
          <List w="full" mb="4">
            {instances.map((instance) => (
              <ListItem key={instance.id}>
                <WorkspaceInstanceCard host={host} instanceName={instance.id} />
              </ListItem>
            ))}
          </List>
        </>
      ) : isLoadingWorkspaces ? (
        <Center w="full" h="60%" flexFlow="column nowrap">
          <Spinner size="xl" thickness="4px" speed="1s" color="gray.600" />
          <Text mt="4">Loading Workspaces...</Text>
        </Center>
      ) : (
        <Container maxW="container.lg" h="full">
          <VStack align="center" justify="center" w="full" h="full">
            <Heading fontWeight="thin" color="gray.600">
              Create a DevPod Workspace
            </Heading>
            <Image src={emptyWorkspacesImage} w="100%" h="40vh" my="12" />

            <Button
              variant="solid"
              colorScheme="primary"
              leftIcon={<DevPodIcon boxSize={5} />}
              onClick={handleCreateClicked}>
              Create Workspace
            </Button>
          </VStack>
        </Container>
      )}
    </VStack>
  )
}

import { useAppReady } from "@/App/useAppReady"
import { useProContext, useProInstances } from "@/contexts"
import { DevPodIcon } from "@/icons"
import emptyWorkspacesImage from "@/images/empty_workspaces.svg"
import { useConnectionStatus, useReLoginProModal } from "@/lib"
import { Button, Container, Heading, Image, Text, VStack } from "@chakra-ui/react"
import { useMemo } from "react"
import { Outlet } from "react-router-dom"

export function ProInstance() {
  const connectionStatus = useConnectionStatus()
  const { host } = useProContext()
  const { errorModal, changelogModal, proLoginModal } = useAppReady()
  const [[proInstances]] = useProInstances()
  const proInstance = useMemo(() => {
    return proInstances?.find((proInstance) => proInstance.host === host)
  }, [host, proInstances])
  const { modal: reLoginProModal, handleOpenLogin: handleReLoginClicked } = useReLoginProModal()

  if (proInstance?.authenticated === false && connectionStatus.state === "connected") {
    return (
      <Container maxW="container.lg" h="full">
        <VStack align="center" justify="center" w="full" h="full">
          <Heading fontWeight="thin" color="gray.600">
            You&apos;ve been logged out
          </Heading>
          <Text>{host}</Text>
          <Image src={emptyWorkspacesImage} w="100%" h="40vh" my="12" />

          <Button
            variant="solid"
            colorScheme="primary"
            leftIcon={<DevPodIcon boxSize={5} />}
            onClick={() => handleReLoginClicked({ host })}>
            Log In
          </Button>
        </VStack>
        {reLoginProModal}
      </Container>
    )
  }

  return (
    <>
      <Outlet />

      {errorModal}
      {changelogModal}
      {proLoginModal}
    </>
  )
}

import { useAppReady } from "@/App/useAppReady"
import { useProContext, useProInstances } from "@/contexts"
import { DevPodIcon } from "@/icons"
import disconnectedImage from "@/images/disconnected.svg"
import disconnectedDarkImage from "@/images/disconnected_dark.svg"
import { hasCapability, useConnectionStatus, useReLoginProModal } from "@/lib"
import { Button, Container, Heading, Image, VStack, useColorMode } from "@chakra-ui/react"
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
  const { colorMode } = useColorMode()

  const loginContent = (
    <Container maxW="container.lg" h="full">
      <VStack align="center" justify="center" w="full" h="full">
        <Image
          src={colorMode == "dark" ? disconnectedDarkImage : disconnectedImage}
          w="100%"
          h="40vh"
        />

        <Heading fontWeight="thin" mb="4" color="gray.600">
          You&apos;ve been logged out
        </Heading>
        <Button
          variant="primary"
          leftIcon={<DevPodIcon boxSize={5} />}
          onClick={() => handleReLoginClicked({ host })}>
          Log In
        </Button>
      </VStack>
      {reLoginProModal}
    </Container>
  )

  if (hasCapability(proInstance, "daemon") && connectionStatus.loginRequired) {
    return loginContent
  } else if (proInstance?.authenticated === false && connectionStatus.healthy) {
    // TODO: This branch can be deprecated after removing proxy provider
    return loginContent
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

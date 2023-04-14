import {
  Button,
  Code,
  Heading,
  HStack,
  Link,
  Modal,
  ModalBody,
  ModalContent,
  ModalFooter,
  ModalOverlay,
  Text,
  useDisclosure,
  VStack,
} from "@chakra-ui/react"
import { useMutation } from "@tanstack/react-query"
import { useCallback, useEffect, useMemo } from "react"
import { useNavigate } from "react-router"
import { client } from "./client"
import { CollapsibleSection, ErrorMessageBox, LoftOSSBadge, Step, Steps } from "./components"
import { Briefcase, CheckCircle, CommandLine, DevpodWordmark, Stack3D } from "./icons"
import { isError } from "./lib"
import { Routes } from "./routes"
import { SetupProviderSteps } from "./views"

const IS_FIRST_VISIT_KEY = "devpod-is-first-visit"

export function useWelcomeModal() {
  const navigate = useNavigate()
  const { isOpen, onClose, onOpen } = useDisclosure()
  const {
    mutate: addBinaryToPath,
    isLoading,
    error,
    status,
  } = useMutation<void, Error>({
    mutationFn: async () => {
      ;(await client.installCLI()).unwrap()
    },
  })
  const handleSetupFinished = useCallback(() => {
    onClose()
    navigate(Routes.WORKSPACE_CREATE)
  }, [navigate, onClose])

  // Only show the welcome modal once
  useEffect(() => {
    const maybeFirstVisit = localStorage.getItem(IS_FIRST_VISIT_KEY)
    if (maybeFirstVisit === null && !isOpen) {
      onOpen()
      localStorage.setItem(IS_FIRST_VISIT_KEY, "false")

      return
    }
  }, [isOpen, onOpen])

  const modal = useMemo(() => {
    return (
      <Modal
        onClose={onClose}
        isOpen={isOpen}
        isCentered
        size="4xl"
        scrollBehavior="inside"
        closeOnOverlayClick={false}>
        <ModalOverlay />
        <ModalContent>
          <ModalBody>
            <VStack align="start" spacing="8" paddingX="4" paddingTop="4">
              <Steps finishText="Get Started" onFinish={handleSetupFinished}>
                <Step>
                  <HStack width="full" justifyContent="space-between" alignItems="center">
                    <HStack>
                      <Heading as="h1" size="lg" marginRight="2">
                        Welcome to
                      </Heading>
                      <DevpodWordmark width="40" height="16" />
                    </HStack>
                    <LoftOSSBadge />
                  </HStack>

                  <Text fontWeight="bold">
                    DevPod is a tool to create reproducible developer environments.
                  </Text>
                  <Text>
                    Each developer environment runs in a separate container and is specified through
                    a devcontainer.json. Through DevPod providers these containers can be created on
                    the local computer, any reachable remote machine or in a public or private
                    cloud. It&apos;s also possible to extend DevPod and write your own custom
                    providers. <br />
                    For more information, head over to our{" "}
                    <Link
                      color="primary.600"
                      onClick={() => client.openLink("https://devpod.sh/docs")}>
                      documentation.
                    </Link>
                  </Text>

                  <Text fontWeight="bold">Let&apos;s set you up!</Text>
                </Step>

                <Step>
                  <HStack>
                    <Stack3D boxSize="6" />
                    <Heading as="h1" size="lg" marginRight="2">
                      Providers
                    </Heading>
                  </HStack>
                  <Text>
                    Providers determine how and where your workspaces run. So, in order to create a
                    workspace you will need to connect a provider first. Providers can be simple
                    docker containers on your local machine or more complex cloud based virtual
                    machines.
                    <Text as="span" color="gray.500">
                      If you just want to explore the app, feel free to skip this step. You can
                      always add new providers in the <Code variant="decorative">Providers</Code>{" "}
                      tab.
                    </Text>
                  </Text>

                  <CollapsibleSection title="Setup your first provider" isOpen={false} showIcon>
                    <SetupProviderSteps />
                  </CollapsibleSection>
                </Step>

                <Step>
                  <HStack>
                    <CommandLine boxSize="6" />
                    <Heading as="h1" size="lg" marginRight="2">
                      CLI
                    </Heading>
                  </HStack>

                  <Text>
                    DevPod ships with a powerful CLI that allows you to create, manage and connect
                    to your workspaces and providers. You can either{" "}
                    <Link onClick={() => client.openLink("")}>download the standalone binary</Link>{" "}
                    or directly add it to your <Code>$PATH</Code>.
                    <Text as="span" color="gray.500">
                      Again, feel free to skip this step. You can always install the CLI from the{" "}
                      <Code variant="decorative">Settings</Code> tab.
                    </Text>
                  </Text>
                  <Button
                    isLoading={isLoading}
                    isDisabled={status === "success"}
                    onClick={() => addBinaryToPath()}>
                    {status === "success" ? <CheckCircle color="green.500" /> : "Add CLI to Path"}
                  </Button>
                  {isError(error) && <ErrorMessageBox error={error} />}
                </Step>

                <Step>
                  <HStack>
                    <Briefcase boxSize="6" />
                    <Heading as="h1" size="lg" marginRight="2">
                      Workspaces
                    </Heading>
                  </HStack>

                  <Text>
                    Workspaces are your reproducible development environment on a per-project basis.
                    Turn a local folder, git repository or docker container into a workspace and
                    connect it to your favorite coding tool. Or just ssh into them and go crazy.
                  </Text>
                </Step>
              </Steps>
            </VStack>
          </ModalBody>
          <ModalFooter />
        </ModalContent>
      </Modal>
    )
  }, [addBinaryToPath, error, handleSetupFinished, isLoading, isOpen, onClose, status])

  return { modal }
}

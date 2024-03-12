import { client } from "@/client"
import { useProInstances, useSettings, useWorkspaces } from "@/contexts"
import { Briefcase, CheckCircle, DevPodProBadge, ExclamationTriangle, Plus } from "@/icons"
import { exists } from "@/lib"
import { TProID, TProInstance } from "@/types"
import { useLoginProModal, useReLoginProModal } from "@/views/ProInstances/useLoginProModal"
import { useDeleteProviderModal } from "@/views/Providers/useDeleteProviderModal"
import { ChevronDownIcon, CloseIcon } from "@chakra-ui/icons"
import {
  Box,
  Button,
  ButtonGroup,
  HStack,
  Heading,
  Icon,
  IconButton,
  Link,
  Popover,
  PopoverArrow,
  PopoverBody,
  PopoverContent,
  PopoverHeader,
  PopoverTrigger,
  Portal,
  Text,
  Tooltip,
  VStack,
  useColorModeValue,
} from "@chakra-ui/react"
import dayjs from "dayjs"
import { useEffect, useMemo, useState } from "react"
import { HiArrowRightOnRectangle, HiClock } from "react-icons/hi2"
import { IconTag } from "../Tag"

export function Pro() {
  const [[proInstances]] = useProInstances()
  const { modal: loginProModal, handleOpenLogin: handleConnectClicked } = useLoginProModal()
  const { modal: reLoginProModal, handleOpenLogin: handleReLoginClicked } = useReLoginProModal()
  const [isDeleting, setIsDeleting] = useState(false)

  const backgroundColor = useColorModeValue("white", "gray.900")
  const handleAnnouncementClicked = () => {
    client.openLink("https://devpod.sh/pro")
  }
  const { experimental_devPodPro } = useSettings()
  const isProUnauthenticated = proInstances?.some(({ authenticated }) => !authenticated)

  return experimental_devPodPro ? (
    <>
      <Popover isLazy isOpen={isDeleting ? true : undefined}>
        <PopoverTrigger>
          <Button
            variant="outline"
            rightIcon={<ChevronDownIcon boxSize={6} />}
            {...(isProUnauthenticated && {
              leftIcon: <ExclamationTriangle boxSize={4} color="orange.300" />,
            })}>
            DevPod Pro
          </Button>
        </PopoverTrigger>
        <Portal>
          <PopoverContent backgroundColor={backgroundColor} zIndex="popover">
            <PopoverArrow backgroundColor={backgroundColor} />
            <PopoverHeader>
              <VStack align="start" spacing="0">
                <Heading size="sm" as="h3">
                  Your Pro Instances
                </Heading>
                <Text fontSize="xs">Manage DevPod Pro</Text>
              </VStack>
              <ButtonGroup variant="outline">
                <Tooltip label="Connect to Pro instance">
                  <IconButton
                    aria-label="Connect to Pro Instace"
                    onClick={() => handleConnectClicked()}
                    icon={<Icon as={HiArrowRightOnRectangle} boxSize={5} />}
                  />
                </Tooltip>
                <Tooltip label="Create new Pro instance">
                  <IconButton aria-label="Create new Pro Instance" isDisabled icon={<Plus />} />
                </Tooltip>
              </ButtonGroup>
            </PopoverHeader>
            <PopoverBody>
              <Box
                width="full"
                overflowY="auto"
                maxHeight="17rem"
                height="full"
                marginTop="3"
                marginBottom="2"
                padding="1">
                {proInstances === undefined || proInstances.length === 0 ? (
                  <VStack align="start" padding="2" spacing="0">
                    <Text fontWeight="bold">No Pro instances</Text>
                    <Text lineHeight={"1.2rem"} fontSize="sm" color="gray.500">
                      You don&apos;t have any Pro instances set up. Connect to an existing Instance
                      or create a new one. <br />
                      <Link
                        color="primary.600"
                        onClick={() => client.openLink("https://devpod.sh/pro")}>
                        Learn more
                      </Link>
                    </Text>
                    <ButtonGroup width="full" marginTop="4" variant="primary">
                      <Button onClick={() => handleConnectClicked()}>Login to Pro</Button>
                      <Button isDisabled>Create new Pro</Button>
                    </ButtonGroup>
                  </VStack>
                ) : (
                  proInstances.map((proInstance) => {
                    const host = proInstance.host
                    if (!host) {
                      return null
                    }

                    return (
                      <ProInstanceRow
                        key={host}
                        {...proInstance}
                        host={host}
                        onIsDeletingChanged={setIsDeleting}
                        onLoginClicked={() => handleReLoginClicked({ host })}
                      />
                    )
                  })
                )}
              </Box>
            </PopoverBody>
          </PopoverContent>
        </Portal>
      </Popover>
      {loginProModal}
      {reLoginProModal}
    </>
  ) : (
    <Button
      variant="outline"
      leftIcon={<DevPodProBadge width="9" height="8" />}
      onClick={handleAnnouncementClicked}>
      Try DevPod Pro
    </Button>
  )
}

type TProInstaceRowProps = Omit<TProInstance, "host"> &
  Readonly<{
    host: TProID
    onIsDeletingChanged: (isDeleting: boolean) => void
    onLoginClicked?: VoidFunction
  }>
function ProInstanceRow({
  host,
  creationTimestamp,
  onIsDeletingChanged,
  provider,
  authenticated,
  onLoginClicked,
}: TProInstaceRowProps) {
  const [, { disconnect }] = useProInstances()
  const workspaces = useWorkspaces()
  const proInstanceWorkspaces = useMemo(
    () => workspaces.filter((workspace) => workspace.provider?.name === provider),
    [provider, workspaces]
  )
  const {
    modal: deleteProviderModal,
    open: openDeleteProviderModal,
    isOpen,
  } = useDeleteProviderModal(
    host,
    "Pro instance",
    "disconnect",
    proInstanceWorkspaces.length > 0,
    () => disconnect.run({ id: host })
  )
  useEffect(() => {
    onIsDeletingChanged(isOpen)
    // `onIsDeletingChanged` is expected to be a stable reference
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isOpen])

  return (
    <>
      <HStack width="full" padding="2" justifyContent="space-between">
        <VStack align="start" spacing="0" fontSize="sm">
          <HStack>
            <Text fontWeight="bold">{host}</Text>
            {exists(authenticated) && (
              <IconTag
                variant="ghost"
                icon={
                  authenticated ? (
                    <CheckCircle color={"green.300"} />
                  ) : (
                    <ExclamationTriangle color="orange.300" />
                  )
                }
                label=""
                paddingInlineStart="0"
                infoText={authenticated ? "Authenticated" : "Not Authenticated"}
                {...(authenticated ? {} : { onClick: onLoginClicked, cursor: "pointer" })}
              />
            )}
          </HStack>
          <HStack>
            <IconTag
              variant="ghost"
              icon={<Briefcase />}
              paddingInlineStart="0"
              label={proInstanceWorkspaces.length.toString(10)}
              infoText={`${proInstanceWorkspaces.length} workspaces`}
            />
            {exists(creationTimestamp) && (
              <IconTag
                variant="ghost"
                icon={<Icon as={HiClock} />}
                label={dayjs(new Date(creationTimestamp)).format("MMM D, YY")}
                infoText={`Created ${dayjs(new Date(creationTimestamp)).fromNow()}`}
              />
            )}
          </HStack>
        </VStack>

        {exists(host) && (
          <Tooltip label="Disconnect from Instance">
            <IconButton
              variant="ghost"
              size="xs"
              aria-label="Disconnect from Instance"
              onClick={openDeleteProviderModal}
              icon={<CloseIcon />}
            />
          </Tooltip>
        )}
      </HStack>

      {deleteProviderModal}
    </>
  )
}

import { client } from "@/client"
import { useProInstances, useWorkspaces } from "@/contexts"
import { Briefcase, DevPodProBadge, Plus } from "@/icons"
import { exists } from "@/lib"
import { TProID, TProInstance } from "@/types"
import { useLoginProModal } from "@/views/ProInstances/useLoginProModal"
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
  Popover,
  PopoverArrow,
  PopoverBody,
  PopoverContent,
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
  const [isDeleting, setIsDeleting] = useState(false)

  const backgroundColor = useColorModeValue("white", "gray.900")
  const handleAnnouncementClicked = () => {
    client.openLink("https://devpod.sh/engine")
  }

  return process.env.DEVPOD_PRO ? (
    <>
      <Popover isLazy isOpen={isDeleting ? true : undefined}>
        <PopoverTrigger>
          <Button variant="outline" rightIcon={<ChevronDownIcon boxSize={6} />}>
            DevPod Pro
          </Button>
        </PopoverTrigger>
        <Portal>
          <PopoverContent backgroundColor={backgroundColor} zIndex="popover">
            <PopoverArrow backgroundColor={backgroundColor} />
            <PopoverBody>
              <HStack
                paddingX="3"
                paddingTop="1"
                paddingBottom="2"
                width="calc(100% + 1.5rem)"
                transform="translateX(-0.75rem)"
                spacing="0"
                justifyContent="space-between"
                borderBottomWidth="thin"
                borderColor="inherit">
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
                      onClick={handleConnectClicked}
                      icon={<Icon as={HiArrowRightOnRectangle} boxSize={5} />}
                    />
                  </Tooltip>
                  <Tooltip label="Create new Pro instance">
                    <IconButton aria-label="Create new Pro Instance" isDisabled icon={<Plus />} />
                  </Tooltip>
                </ButtonGroup>
              </HStack>

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
                    <Text marginTop="4" fontWeight="bold">
                      No Pro instances
                    </Text>
                    <Text lineHeight={"1.2rem"} fontSize="sm" color="gray.500">
                      You don&apos;t have any Pro instances set up. Connect to an existing Instance
                      or create a new one.
                    </Text>
                    <ButtonGroup width="full" marginTop="2" variant="primary">
                      <Button onClick={handleConnectClicked}>Login to Pro</Button>
                      <Button isDisabled>Create new Pro</Button>
                    </ButtonGroup>
                  </VStack>
                ) : (
                  proInstances.map(
                    (proInstance) =>
                      proInstance.id && (
                        <ProInstaceRow
                          key={proInstance.id}
                          {...proInstance}
                          id={proInstance.id}
                          onIsDeletingChanged={setIsDeleting}
                        />
                      )
                  )
                )}
              </Box>
            </PopoverBody>
          </PopoverContent>
        </Portal>
      </Popover>
      {loginProModal}
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

type TProInstaceRowProps = Omit<TProInstance, "id"> &
  Readonly<{ id: TProID; onIsDeletingChanged: (isDeleting: boolean) => void }>
function ProInstaceRow({ id, creationTimestamp, onIsDeletingChanged }: TProInstaceRowProps) {
  const [, { disconnect }] = useProInstances()
  const workspaces = useWorkspaces()
  const proInstanceWorkspaces = useMemo(
    () => workspaces.filter((workspace) => workspace.provider?.name === id),
    [id, workspaces]
  )
  const {
    modal: deleteProviderModal,
    open: openDeleteProviderModal,
    isOpen,
  } = useDeleteProviderModal(
    id,
    "Pro instance",
    "disconnect",
    proInstanceWorkspaces.length > 0,
    () => disconnect.run({ id })
  )
  useEffect(() => {
    onIsDeletingChanged(isOpen)
    // `onIsDeletingChanged` is expectd to be a stable reference
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isOpen])

  return (
    <>
      <HStack width="full" padding="2" justifyContent="space-between">
        <VStack align="start" spacing="0" fontSize="sm">
          <Text fontWeight="bold">{id}</Text>
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
                label={dayjs(new Date(creationTimestamp)).fromNow()}
                infoText={`Created ${dayjs(new Date(creationTimestamp)).fromNow()}`}
              />
            )}
          </HStack>
        </VStack>

        {exists(id) && (
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

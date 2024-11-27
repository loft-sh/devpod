import { Close, Connect, DevpodWordmark, Ellipsis, Folder } from "@/icons"
import { getDisplayName, Result, useLoginProModal } from "@/lib"
import { TProInstance } from "@/types"
import { useDeleteProviderModal } from "@/views/Providers"
import { ArrowUpDownIcon, CheckIcon } from "@chakra-ui/icons"
import {
  Box,
  Button,
  HStack,
  Heading,
  IconButton,
  Image,
  List,
  ListItem,
  Menu,
  MenuButton,
  MenuItem,
  MenuList,
  Popover,
  PopoverBody,
  PopoverContent,
  PopoverTrigger,
  Portal,
  Text,
  VStack,
  Tooltip,
  Spinner,
} from "@chakra-ui/react"
import { ManagementV1Project } from "@loft-enterprise/client/gen/models/managementV1Project"
import { ReactNode, useMemo } from "react"
import { useProInstances } from "../proInstances"

export const HOST_OSS = "Open Source"
type THostPickerProps = Readonly<{
  currentHost: string
  onHostChange: (newHost: string) => void

  currentProject: ManagementV1Project
  projects: readonly ManagementV1Project[]
  onProjectChange: (newProject: ManagementV1Project) => void
  onCancelWatch?: () => Promise<Result<undefined>>
  waitingForCancel: boolean
}>
export function ContextSwitcher({
  currentHost,
  projects,
  currentProject,
  onProjectChange,
  onHostChange,
  onCancelWatch,
  waitingForCancel,
}: THostPickerProps) {
  const [[rawProInstances]] = useProInstances()
  const proInstances = useMemo(() => {
    const p: (TProInstance & { image?: string | ReactNode })[] =
      rawProInstances
        ?.slice()
        .sort((a, b) => {
          if (a.host === currentHost) {
            return -1
          }
          if (b.host === currentHost) {
            return 1
          }

          return 0
        })
        .map((proInstance) => ({ ...proInstance })) ?? []

    p.push({
      host: HOST_OSS,
      image: <DevpodWordmark w="20" h="6" />,
      authenticated: undefined,
      provider: undefined,
      creationTimestamp: undefined,
    })

    return p
  }, [currentHost, rawProInstances])

  const { modal: loginProModal, handleOpenLogin: handleConnectClicked } = useLoginProModal()
  const handleConnectPlatform = () => {
    handleConnectClicked()
  }

  return (
    <>
      <Popover>
        <PopoverTrigger>
          <Button variant="ghost" color="gray.700" rightIcon={<ArrowUpDownIcon />}>
            {getDisplayName(currentProject, "Unknown Project")}
          </Button>
        </PopoverTrigger>
        <Portal>
          <PopoverContent minWidth={"25rem"}>
            <PopoverBody p="0">
              {waitingForCancel ? (
                <HStack alignItems={"center"} justifyContent={"center"} paddingY={"4"}>
                  <Spinner />
                </HStack>
              ) : (
                <List>
                  {proInstances.map(({ host, authenticated, image }) => (
                    <ListItem key={host}>
                      <PlatformDetails
                        currentHost={currentHost}
                        host={host!}
                        image={image}
                        onCancelWatch={onCancelWatch}
                        authenticated={authenticated}
                        onConnect={handleConnectPlatform}
                        onClick={() => onHostChange(host!)}
                      />
                      {host === currentHost && (
                        <VStack
                          w="full"
                          align="start"
                          pb="4"
                          pt="2"
                          pl="2"
                          borderBottomWidth="thin"
                          borderBottomStyle="solid">
                          <Heading pl="4" size="xs" color="gray.500" textTransform="uppercase">
                            Projects
                          </Heading>
                          <List w="full">
                            {projects.map((project) => (
                              <ListItem key={project.metadata!.name}>
                                <Button
                                  _hover={{ bgColor: "gray.100" }}
                                  variant="unstyled"
                                  w="full"
                                  display="flex"
                                  justifyContent="start"
                                  alignItems="center"
                                  leftIcon={<Folder boxSize={5} />}
                                  pl="4"
                                  color="gray.600"
                                  fontWeight="normal"
                                  rightIcon={
                                    project.metadata?.name === currentProject.metadata?.name ? (
                                      <CheckIcon />
                                    ) : undefined
                                  }
                                  onClick={() => onProjectChange(project)}>
                                  {getDisplayName(project)}
                                </Button>
                              </ListItem>
                            ))}
                          </List>
                        </VStack>
                      )}
                    </ListItem>
                  ))}
                </List>
              )}
            </PopoverBody>
          </PopoverContent>
        </Portal>
      </Popover>

      {loginProModal}
    </>
  )
}
type TPlatformDetailsProps = Readonly<{
  host: string
  currentHost: string
  image: ReactNode
  authenticated?: boolean | null
  onClick: VoidFunction
  onConnect: VoidFunction
  onCancelWatch?: () => Promise<Result<undefined>>
}>
function PlatformDetails({
  host,
  currentHost,
  image,
  authenticated,
  onClick,
  onConnect,
  onCancelWatch,
}: TPlatformDetailsProps) {
  const [, { disconnect }] = useProInstances()
  const { modal: deleteProviderModal, open: openDeleteProviderModal } = useDeleteProviderModal(
    host,
    "Pro instance",
    "disconnect",
    async () => {
      await onCancelWatch?.()
      disconnect.run({ id: host })
    }
  )

  return (
    <>
      <HStack
        _hover={{ bgColor: "gray.100", cursor: "pointer" }}
        w="full"
        px="4"
        h="12"
        onClick={onClick}
        {...(currentHost != host
          ? {
              borderBottomStyle: "solid",
              borderBottomWidth: "thin",
            }
          : {})}>
        <HStack
          w="full"
          overflow="hidden"
          textOverflow="ellipsis"
          whiteSpace="nowrap"
          justify="space-between">
          {image ? (
            typeof image === "string" ? (
              <Image src={image} />
            ) : (
              image
            )
          ) : (
            <Tooltip maxW={"25rem"} label={host} openDelay={0} closeDelay={0}>
              <Text
                maxW="50%"
                fontWeight="semibold"
                fontSize="sm"
                overflow="hidden"
                textOverflow="ellipsis">
                {host}
              </Text>
            </Tooltip>
          )}
          <HStack maxW="50%">
            {authenticated != null && (
              <Box
                flexShrink="0"
                boxSize="2"
                bg={authenticated ? "green.400" : "orange.400"}
                rounded="full"
              />
            )}
            <Tooltip maxW={"25rem"} label={host} openDelay={0} closeDelay={0}>
              <Text
                overflow="hidden"
                textOverflow="ellipsis"
                whiteSpace="nowrap"
                marginTop="1px"
                fontSize="xs"
                fontWeight="normal">
                {host}
              </Text>
            </Tooltip>
            {host !== HOST_OSS && (
              <Menu>
                <MenuButton
                  onClick={(e) => e.stopPropagation()}
                  as={IconButton}
                  variant="ghost"
                  aria-label="More actions"
                  colorScheme="gray"
                  icon={<Ellipsis transform={"rotate(90deg)"} boxSize={5} />}
                />
                <MenuList color="gray.700" onClick={(e) => e.stopPropagation()}>
                  <MenuItem icon={<Connect boxSize={4} />} onClick={onConnect}>
                    Connect another platform
                  </MenuItem>
                  <MenuItem icon={<Close boxSize={4} />} onClick={openDeleteProviderModal}>
                    Disconnect
                  </MenuItem>
                </MenuList>
              </Menu>
            )}
          </HStack>
        </HStack>
      </HStack>
      {deleteProviderModal}
    </>
  )
}

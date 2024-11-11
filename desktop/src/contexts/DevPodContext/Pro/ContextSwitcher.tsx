import { Close, Connect, DevpodWordmark, Ellipsis, Folder } from "@/icons"
import { getDisplayName, useLoginProModal } from "@/lib"
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
}>
export function ContextSwitcher({
  currentHost,
  projects,
  currentProject,
  onProjectChange,
  onHostChange,
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
          <PopoverContent>
            <PopoverBody p="0">
              <List>
                {proInstances.map(({ host, authenticated, image }) => (
                  <ListItem key={host}>
                    <PlatformDetails
                      currentHost={currentHost}
                      host={host!}
                      image={image}
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
}>
function PlatformDetails({
  host,
  currentHost,
  image,
  authenticated,
  onClick,
  onConnect,
}: TPlatformDetailsProps) {
  const [, { disconnect }] = useProInstances()
  const { modal: deleteProviderModal, open: openDeleteProviderModal } = useDeleteProviderModal(
    host,
    "Pro instance",
    "disconnect",
    () => disconnect.run({ id: host })
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
        <HStack w="full" justify="space-between">
          {image ? (
            typeof image === "string" ? (
              <Image src={image} />
            ) : (
              image
            )
          ) : (
            <Text
              maxW="50%"
              fontWeight="semibold"
              fontSize="sm"
              overflow="hidden"
              textOverflow="ellipsis">
              {host}
            </Text>
          )}
          <HStack>
            {authenticated != null && (
              <Box boxSize="2" bg={authenticated ? "green.400" : "orange.400"} rounded="full" />
            )}
            <Text fontSize="xs" fontWeight="normal">
              {host}
            </Text>
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

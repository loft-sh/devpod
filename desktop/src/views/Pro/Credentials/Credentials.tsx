import { DaemonClient } from "@/client/pro/client"
import { ErrorMessageBox } from "@/components"
import { useProContext } from "@/contexts"
import { LockDuotone, Trash } from "@/icons"
import EmptyImage from "@/images/empty_default.svg"
import EmptyDarkImage from "@/images/empty_default_dark.svg"
import { deepCopy } from "@/lib"
import { randomWords } from "@/lib/randomWords"
import { QueryKeys } from "@/queryKeys"
import { TGitCredentialData, TUserSecretType, UserSecret } from "@/types"
import {
  Button,
  HStack,
  Heading,
  IconButton,
  Image,
  Modal,
  ModalBody,
  ModalCloseButton,
  ModalContent,
  ModalHeader,
  ModalOverlay,
  Skeleton,
  Tab,
  TabList,
  TabPanel,
  TabPanels,
  Table,
  TableContainer,
  Tabs,
  Tag,
  Tbody,
  Td,
  Text,
  Th,
  Thead,
  Tooltip,
  Tr,
  VStack,
  useColorMode,
  useDisclosure,
} from "@chakra-ui/react"
import { ManagementV1UserProfile } from "@loft-enterprise/client/gen/models/managementV1UserProfile"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useCallback, useMemo } from "react"
import { BackToWorkspaces } from "../BackToWorkspaces"
import { AddGitHTTPCredentials } from "./AddGitHTTPCredentials"
import { AddGitSSHCredentials } from "./AddGitSSHCredentials"

export function Credentials() {
  const queryClient = useQueryClient()
  const { client, managementSelfQuery: s } = useProContext()
  const { data: userProfile, isLoading } = useQuery({
    queryKey: QueryKeys.userProfile(s.data?.status?.user?.name),
    queryFn: async () => {
      return (await (client as DaemonClient).getUserProfile()).unwrap()
    },
  })
  const { colorMode } = useColorMode()
  const { modal, show } = useCreateCredentialModal(userProfile, s.data?.status?.user?.name)
  const secrets = Object.entries(userProfile?.secrets ?? {})

  const deleteSecret = useMutation({
    mutationFn: async ({ name }: Readonly<{ name: string }>) => {
      const newSecrets = deepCopy(userProfile?.secrets)
      delete newSecrets?.[name]

      return (
        await (client as DaemonClient).updateUserProfile({ ...userProfile, secrets: newSecrets })
      ).unwrap()
    },
    onSuccess: (newData) => {
      // optimistic update
      queryClient.setQueryData(QueryKeys.userProfile(s.data?.status?.user?.name), newData)
      queryClient.invalidateQueries(QueryKeys.userProfile(s.data?.status?.user?.name))
    },
  })

  return (
    <>
      <VStack align="start">
        <BackToWorkspaces />
        <HStack align="center" justify="space-between" mb="2" w="full">
          <Heading fontWeight="thin">Credentials</Heading>
          <Button
            variant="outline"
            colorScheme="primary"
            leftIcon={<LockDuotone boxSize={5} />}
            onClick={show}>
            Add Credentials
          </Button>
        </HStack>
        <Text my="4" variant="muted">
          Credentials connect DevPod Pro to one or multiple of your git providers. You can upload
          both HTTPS tokens and SSH private keys.
        </Text>
        {isLoading ? (
          <Table>
            <Thead>
              <Tr>
                <Th>
                  <Skeleton h="5" w="36" />
                </Th>
                <Th>
                  <Skeleton h="5" w="24" />
                </Th>
              </Tr>
            </Thead>
            <Tbody>
              {[...Array(3)].map((_, i) => (
                <Tr key={i}>
                  <Td>
                    <Skeleton h="5" w="48" />
                  </Td>
                  <Td>
                    <Skeleton h="5" w="32" />
                  </Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        ) : secrets.length === 0 ? (
          <VStack
            h={"full"}
            w={"full"}
            justifyContent={"center"}
            alignItems={"center"}
            flexGrow={1}
            my="8">
            <Image src={colorMode === "dark" ? EmptyDarkImage : EmptyImage} />
            <Text variant="muted" fontWeight={"semibold"} fontSize={"sm"}>
              No credentials found
            </Text>
          </VStack>
        ) : (
          <TableContainer w="full">
            <Table size="sm">
              <Thead w="full">
                <Tr>
                  <Th>Name</Th>
                  <Th>Type</Th>
                  <Th>Host</Th>
                  <Th>User</Th>
                  <Th />
                </Tr>
              </Thead>
              <Tbody>
                {secrets.map(([name, secret]) => {
                  let gitData = {} as TGitCredentialData
                  try {
                    gitData = JSON.parse(secret.data ?? "") as TGitCredentialData
                  } catch {
                    // noop
                  }

                  return (
                    <Tr key={name}>
                      <Td>{name}</Td>
                      <Td>
                        <SecretTypeTag type={secret.type} />
                      </Td>
                      <Td>{gitData.host}</Td>
                      <Td>{gitData.user}</Td>
                      <Td textAlign="end">
                        <Tooltip label="Delete secret">
                          <IconButton
                            ml="auto"
                            aria-label="Delete secret"
                            icon={<Trash boxSize={5} />}
                            variant="ghost"
                            size="sm"
                            colorScheme="red"
                            onClick={() => deleteSecret.mutate({ name })}
                          />
                        </Tooltip>
                      </Td>
                    </Tr>
                  )
                })}
              </Tbody>
            </Table>
          </TableContainer>
        )}
      </VStack>

      {modal}
    </>
  )
}

type TSecretTypeTagProps = Readonly<{ type: string | undefined }>
function SecretTypeTag({ type }: TSecretTypeTagProps) {
  const displayName = useMemo(() => {
    if (type === UserSecret.GIT_HTTP) {
      return "https"
    }
    // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
    if (type === UserSecret.GIT_SSH) {
      return "ssh"
    }

    return ""
  }, [type])

  return <Tag>{displayName}</Tag>
}

function useCreateCredentialModal(
  userProfile: ManagementV1UserProfile | undefined,
  userName: string | undefined
) {
  const queryClient = useQueryClient()
  const { client } = useProContext()
  const { isOpen, onClose, onOpen } = useDisclosure()
  const addCredentials = useMutation({
    mutationFn: async ({
      newCredentials,
      name,
      type,
    }: Readonly<{ name: string; type: TUserSecretType; newCredentials: TGitCredentialData }>) => {
      return (
        await (client as DaemonClient).updateUserProfile({
          ...userProfile,
          secrets: {
            ...userProfile?.secrets,
            [name]: {
              type,
              data: JSON.stringify(newCredentials),
            },
          },
        })
      ).unwrap()
    },
  })
  const areInputsDisabled = useMemo(
    () => addCredentials.status === "success" || addCredentials.status === "loading",
    [addCredentials.status]
  )

  const handleAddCredentials = useCallback(
    (type: TUserSecretType) => (name: string | undefined, data: TGitCredentialData) => {
      if (!name) {
        name = randomWords({ amount: 2, maxLength: 40 }).join("-")
      }
      addCredentials.mutate(
        { name, type, newCredentials: data },
        {
          onSuccess: () => {
            queryClient.invalidateQueries(QueryKeys.userProfile(userName))
            onClose()
          },
        }
      )
    },
    [addCredentials, onClose, queryClient, userName]
  )

  const modal = useMemo(() => {
    return (
      <Modal
        onClose={onClose}
        isOpen={isOpen}
        isCentered
        size="4xl"
        scrollBehavior="inside"
        closeOnEsc
        closeOnOverlayClick>
        <ModalOverlay />
        <ModalContent>
          <ModalCloseButton />
          <ModalHeader>Add New Credentials</ModalHeader>
          <ModalBody borderRadius={"md"}>
            <Tabs colorScheme="gray">
              <TabList>
                <Tab>SSH</Tab>
                <Tab>HTTPS</Tab>
              </TabList>
              <TabPanels paddingTop="2">
                <TabPanel>
                  {addCredentials.error ? (
                    <ErrorMessageBox ml="4" error={Error(addCredentials.error as any)} />
                  ) : null}
                  <AddGitSSHCredentials
                    isDisabled={areInputsDisabled}
                    onCreate={handleAddCredentials(UserSecret.GIT_SSH)}
                  />
                </TabPanel>
                <TabPanel>
                  {addCredentials.error ? (
                    <ErrorMessageBox error={Error(addCredentials.error as any)} />
                  ) : null}
                  <AddGitHTTPCredentials
                    isDisabled={areInputsDisabled}
                    onCreate={handleAddCredentials(UserSecret.GIT_HTTP)}
                  />
                </TabPanel>
              </TabPanels>
            </Tabs>
          </ModalBody>
        </ModalContent>
      </Modal>
    )
  }, [addCredentials.error, areInputsDisabled, handleAddCredentials, isOpen, onClose])

  const show = useCallback(() => {
    onOpen()
  }, [onOpen])

  return { modal, show }
}

import {
  Box,
  Button,
  ButtonGroup,
  Card,
  CardBody,
  CardFooter,
  CardHeader,
  Center,
  HStack,
  Heading,
  Icon,
  IconButton,
  Image,
  Link,
  Switch,
  Text,
  Tooltip,
} from "@chakra-ui/react"
import { UseMutationResult, useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useMemo } from "react"
import { HiDuplicate } from "react-icons/hi"
import { HiArrowPath, HiPencil } from "react-icons/hi2"
import { Link as RouterLink, useNavigate } from "react-router-dom"
import { client } from "../../client"
import { IconTag } from "../../components"
import { useWorkspaces } from "../../contexts"
import { ProviderPlaceholder, Stack3D, Trash } from "../../icons"
import { exists } from "../../lib"
import { QueryKeys } from "../../queryKeys"
import { Routes } from "../../routes"
import {
  TProvider,
  TProviderID,
  TProviderSource,
  TRunnable,
  TWithProviderID,
  TWorkspace,
} from "../../types"
import { useSetupProviderModal } from "./useSetupProviderModal"
import { useDeleteProviderModal } from "./useDeleteProviderModal"

type TProviderCardProps = {
  id: string
  provider: TProvider
  remove: TRunnable<TWithProviderID> &
    Pick<UseMutationResult, "status" | "error"> & { target: TWithProviderID | undefined }
}

export function ProviderCard({ id, provider, remove }: TProviderCardProps) {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const workspaces = useWorkspaces<TWorkspace>()
  const providerWorkspaces = useMemo(
    () => workspaces.filter((workspace) => workspace.provider?.name === id),
    [id, workspaces]
  )
  const { modal: setupProviderModal, show: showSetupProviderModal } = useSetupProviderModal()
  const { data: providerUpdate } = useQuery({
    queryKey: QueryKeys.providerUpdate(id),
    queryFn: async () => {
      const result = (await client.providers.checkUpdate(id)).unwrap()

      return result
    },
  })
  const { mutate: updateDefaultProvider } = useMutation<
    void,
    unknown,
    Readonly<{ providerID: TProviderID }>
  >({
    mutationFn: async ({ providerID }) => {
      ;(await client.providers.useProvider(providerID)).unwrap()
    },
    onSettled: () => {
      queryClient.invalidateQueries(QueryKeys.PROVIDERS)
    },
  })
  const { mutate: updateProvider } = useMutation<
    void,
    unknown,
    Readonly<{ providerID: TProviderID; source: TProviderSource }>
  >({
    mutationFn: async ({ providerID, source }) => {
      ;(await client.providers.update(providerID, source)).unwrap()
    },
    onSettled: () => {
      queryClient.invalidateQueries(QueryKeys.PROVIDERS)
      queryClient.invalidateQueries(QueryKeys.providerUpdate(id))
      queryClient.invalidateQueries(QueryKeys.PROVIDERS_CHECK_UPDATE_ALL)
    },
  })
  const { modal: deleteProviderModal, open: openDeleteProviderModal } = useDeleteProviderModal(
    id,
    "provider",
    "delete",
    () => remove.run({ providerID: id })
  )

  const providerIcon = provider.config?.icon
  const isDefaultProvider = provider.default ?? false
  const providerVersion = provider.config?.version
  const providerRawSource = provider.config?.source?.raw
  const providerSource = provider.config?.source

  return (
    <>
      <Card variant="outline" width="72" height="96" overflow="hidden">
        <Box
          width="full"
          height="1"
          bgGradient={
            isDefaultProvider ? "linear(to-r, primary.400 30%, primary.500)" : "transparent"
          }
          position="absolute"
        />
        <CardHeader display="flex" justifyContent="center" padding="0">
          {exists(providerIcon) ? (
            <Image
              objectFit="cover"
              padding="4"
              borderRadius="md"
              height="44"
              src={providerIcon}
              alt="Provider Image"
            />
          ) : (
            <Center height="44">
              <ProviderPlaceholder boxSize={24} color="chakra-body-text" />
            </Center>
          )}
        </CardHeader>
        <CardBody>
          <Heading size="md">
            <Link
              as={RouterLink}
              color="var(--chakra-colors-chakra-body-text)"
              to={Routes.toProvider(id)}>
              {id}
            </Link>
          </Heading>
          {providerVersion && (
            <HStack spacing="0">
              <Text
                variant="muted"
                paddingY="1"
                fontFamily="monospace"
                fontSize="sm"
                fontWeight="regular">
                {providerVersion}
              </Text>
              {providerUpdate &&
                providerUpdate.updateAvailable &&
                providerSource &&
                !provider.isProxyProvider && (
                  <Tooltip
                    label={
                      providerUpdate.latestVersion
                        ? `Version ${providerUpdate.latestVersion} available`
                        : "New version available"
                    }>
                    <Button
                      marginLeft="2"
                      aria-label="Update provider"
                      colorScheme="orange"
                      size="xs"
                      leftIcon={<Icon as={HiArrowPath} boxSize="4" />}
                      onClick={() => updateProvider({ providerID: id, source: providerSource })}>
                      Update
                    </Button>
                  </Tooltip>
                )}
            </HStack>
          )}
          <HStack rowGap={2} marginTop={4} flexWrap="nowrap" alignItems="center">
            <IconTag
              icon={<Stack3D />}
              label={
                providerWorkspaces.length === 1
                  ? "1 workspace"
                  : providerWorkspaces.length > 0
                  ? providerWorkspaces.length + " workspaces"
                  : "No workspaces"
              }
              info={`This provider is used by ${providerWorkspaces.length} ${
                providerWorkspaces.length === 1 ? "workspace" : "workspaces"
              }`}
            />
          </HStack>
        </CardBody>
        <CardFooter
          display="flex"
          alignItems="flex-end"
          justify="space-between"
          paddingBottom="4"
          paddingTop="0"
          paddingX="4">
          <HStack>
            <Switch
              isDisabled={isDefaultProvider}
              isChecked={isDefaultProvider}
              onChange={(e) => {
                if (e.target.checked) {
                  updateDefaultProvider({ providerID: id })
                }
              }}
            />
            <Text fontSize="sm" variant="muted">
              Default
            </Text>
          </HStack>
          <ButtonGroup spacing="0">
            {providerRawSource && (
              <Tooltip label="Clone Provider">
                <IconButton
                  aria-label="Clone Provider"
                  variant="ghost"
                  onClick={() =>
                    showSetupProviderModal({
                      isStrict: false,
                      cloneProviderInfo: {
                        sourceProviderID: id,
                        sourceProvider: provider,
                        sourceProviderSource: providerRawSource,
                      },
                    })
                  }
                  icon={<Icon as={HiDuplicate} boxSize="4" />}
                  isDisabled={provider.isProxyProvider}
                />
              </Tooltip>
            )}
            <Tooltip label="Edit Provider">
              <IconButton
                aria-label="Edit Provider"
                variant="ghost"
                onClick={() => navigate(Routes.toProvider(id))}
                icon={<Icon as={HiPencil} boxSize="4" />}
              />
            </Tooltip>
            <Tooltip
              label={
                provider.isProxyProvider
                  ? "This provider is associated with a Pro instance. Disconnecting the Pro instance will automatically delete this provider"
                  : "Delete Provider"
              }>
              <IconButton
                aria-label="Delete Provider"
                variant="ghost"
                colorScheme="gray"
                icon={<Trash boxSize="4" />}
                onClick={openDeleteProviderModal}
                isLoading={remove.status === "loading" && remove.target?.providerID === id}
                isDisabled={provider.isProxyProvider}
              />
            </Tooltip>
          </ButtonGroup>
        </CardFooter>
      </Card>

      {setupProviderModal}
      {deleteProviderModal}
    </>
  )
}

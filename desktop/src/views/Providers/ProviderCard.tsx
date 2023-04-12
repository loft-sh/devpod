import {
  Button,
  ButtonGroup,
  Card,
  CardBody,
  CardFooter,
  CardHeader,
  Center,
  Heading,
  HStack,
  IconButton,
  Image,
  Link,
  Modal,
  ModalBody,
  ModalCloseButton,
  ModalContent,
  ModalFooter,
  ModalHeader,
  ModalOverlay,
  Tag,
  TagLabel,
  Tooltip,
  useColorModeValue,
  useDisclosure,
} from "@chakra-ui/react"
import { UseMutationResult } from "@tanstack/react-query"
import { useMemo } from "react"
import { useNavigate } from "react-router"
import { Link as RouterLink } from "react-router-dom"
import { useWorkspaces } from "../../contexts"
import { ProviderPlaceholder, Stack3D, Trash } from "../../icons"
import { exists } from "../../lib"
import { Routes } from "../../routes"
import { TProvider, TRunnable, TWithProviderID } from "../../types"

type TProviderCardProps = {
  id: string
  provider: TProvider | undefined
  remove: TRunnable<TWithProviderID> &
    Pick<UseMutationResult, "status" | "error"> & { target: TWithProviderID | undefined }
}

export function ProviderCard({ id, provider, remove }: TProviderCardProps) {
  const navigate = useNavigate()
  const workspaces = useWorkspaces()
  const { isOpen: isDeleteOpen, onOpen: onDeleteOpen, onClose: onDeleteClose } = useDisclosure()
  const providerWorkspaces = useMemo(
    () => workspaces.filter((workspace) => workspace.provider?.name === id),
    [id, workspaces]
  )
  const tagColor = useColorModeValue("gray.700", "gray.300")

  return (
    <>
      <Card variant="outline" width="72" height="96" key={id}>
        <CardHeader display="flex" justifyContent="center" padding="0">
          {exists(provider?.config?.icon) ? (
            <Image
              objectFit="cover"
              padding="8"
              borderRadius="md"
              height="48"
              src={provider?.config?.icon}
              alt="Project Image"
            />
          ) : (
            <Center height="48">
              <ProviderPlaceholder boxSize={24} color="chakra-body-text" />
            </Center>
          )}
        </CardHeader>
        <CardBody>
          <Heading size="md">
            <Link as={RouterLink} to={Routes.toProvider(id)}>
              {id}
            </Link>
          </Heading>
          <HStack rowGap={2} marginTop={4} flexWrap="nowrap" alignItems="center">
            <Tag borderRadius="full" color={tagColor}>
              <Stack3D boxSize={4} />
              <TagLabel marginLeft={2}>
                {providerWorkspaces.length === 1
                  ? "1 workspace"
                  : providerWorkspaces.length > 0
                  ? providerWorkspaces.length + " workspaces"
                  : "No workspaces"}
              </TagLabel>
            </Tag>
            {provider?.default && (
              <Tag borderRadius="full" color={tagColor}>
                <TagLabel>{"default"}</TagLabel>
              </Tag>
            )}
          </HStack>
        </CardBody>
        <CardFooter justify="end">
          <ButtonGroup>
            <Button onClick={() => navigate(Routes.toProvider(id))} isLoading={false}>
              Edit
            </Button>
            <Tooltip label={`Delete Provider`}>
              <IconButton
                aria-label="Delete Provider"
                variant="ghost"
                colorScheme="gray"
                icon={<Trash width={"16px"} />}
                onClick={() => {
                  onDeleteOpen()
                }}
                isLoading={remove.status === "loading" && remove.target?.providerID === id}
              />
            </Tooltip>
          </ButtonGroup>
        </CardFooter>
      </Card>
      <Modal onClose={onDeleteClose} isOpen={isDeleteOpen} isCentered>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>Delete Provider</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            {providerWorkspaces.length === 0 ? (
              <>
                Deleting the provider will erase all provider state. Make sure to delete provider
                workspaces before. Are you sure you want to delete provider {id}?
              </>
            ) : (
              <>
                Please make sure to delete all workspaces that use this provider, before deleting
                this provider itself
              </>
            )}
          </ModalBody>
          <ModalFooter>
            <HStack spacing={"2"}>
              <Button onClick={onDeleteClose}>Close</Button>
              {!providerWorkspaces.length && (
                <Button
                  colorScheme={"red"}
                  onClick={async () => {
                    remove.run({ providerID: id })
                    onDeleteClose()
                  }}>
                  Delete
                </Button>
              )}
            </HStack>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </>
  )
}

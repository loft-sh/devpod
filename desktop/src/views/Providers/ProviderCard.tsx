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
  Tooltip,
  useDisclosure,
} from "@chakra-ui/react"
import { UseMutationResult } from "@tanstack/react-query"
import { Link as RouterLink } from "react-router-dom"
import { ProviderPlaceholder, Trash } from "../../icons"
import { exists, noop } from "../../lib"
import { Routes } from "../../routes"
import { TProvider, TRunnable, TWithProviderID } from "../../types"

type TProviderCardProps = {
  id: string
  provider: TProvider | undefined
  remove: TRunnable<TWithProviderID> &
    Pick<UseMutationResult, "status" | "error"> & { target: TWithProviderID | undefined }
}

export function ProviderCard({ id, provider, remove }: TProviderCardProps) {
  const { isOpen: isDeleteOpen, onOpen: onDeleteOpen, onClose: onDeleteClose } = useDisclosure()

  return (
    <>
      <Card variant="outline" height="96" key={id}>
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
        </CardBody>
        <CardFooter justify="end">
          <ButtonGroup>
            <Button onClick={noop} isLoading={false}>
              Update
            </Button>
            <Tooltip label={`Delete Provider`}>
              <IconButton
                aria-label="Delete Provider"
                variant="ghost"
                colorScheme="gray"
                icon={<Trash width={"16px"} />}
                onClick={onDeleteOpen}
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
            Deleting the provider will erase all provider state. Make sure to delete provider
            workspaces before. Are you sure you want to delete provider {id}?
          </ModalBody>
          <ModalFooter>
            <HStack spacing={"2"}>
              <Button onClick={onDeleteClose}>Close</Button>
              <Button
                colorScheme={"red"}
                onClick={async () => {
                  remove.run({ providerID: id })
                  onDeleteClose()
                }}>
                Delete
              </Button>
            </HStack>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </>
  )
}

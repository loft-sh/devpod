import {
  Button,
  ButtonGroup,
  Card,
  CardBody,
  CardFooter,
  CardHeader,
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
import { Link as RouterLink } from "react-router-dom"
import { Routes } from "../../routes"
import { noop } from "../../lib"
import { TProvider, TRunnable, TWithProviderID } from "../../types"
import { UseMutationResult } from "@tanstack/react-query"
import CodeImage from "../../images/code.jpg"
import { Trash } from "../../icons"

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
      <Card variant="outline" width={"200px"} key={id}>
        <CardHeader>
          <Image
            objectFit="cover"
            maxW={{ base: "100%" }}
            src={provider?.config?.icon || CodeImage}
            alt="Project Image"
          />
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

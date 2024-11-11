import {
  Box,
  Button,
  HStack,
  Modal,
  ModalBody,
  ModalCloseButton,
  ModalContent,
  ModalFooter,
  ModalHeader,
  ModalOverlay,
  Portal,
  Text,
  useDisclosure,
} from "@chakra-ui/react"
import { useMemo } from "react"

export function useDeleteProviderModal(
  id: string,
  entityName: string,
  actionName: "delete" | "disconnect",
  onRemoveClicked: () => void
) {
  const { isOpen, onOpen, onClose } = useDisclosure()

  const modal = useMemo(() => {
    return (
      <Portal>
        <Modal onClose={onClose} isOpen={isOpen} isCentered>
          <ModalOverlay />
          <ModalContent>
            <ModalHeader>
              <Box as="span" textTransform="capitalize">
                {actionName}
              </Box>{" "}
              {id}
            </ModalHeader>
            <ModalCloseButton />
            <ModalBody>
              <Text>
                Please make sure to delete all workspaces that use this {entityName}, before{" "}
                {actionName}ing the {entityName}.
              </Text>
            </ModalBody>
            <ModalFooter>
              <HStack spacing={"2"}>
                <Button onClick={onClose}>Close</Button>
                <Button
                  textTransform="capitalize"
                  colorScheme="red"
                  onClick={() => {
                    onRemoveClicked()
                    onClose()
                  }}>
                  {actionName}
                </Button>
              </HStack>
            </ModalFooter>
          </ModalContent>
        </Modal>
      </Portal>
    )
  }, [actionName, entityName, id, isOpen, onClose, onRemoveClicked])

  return { modal, open: onOpen, isOpen }
}

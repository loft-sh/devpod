import {
  Button,
  HStack,
  Modal,
  ModalBody,
  ModalCloseButton,
  ModalContent,
  ModalFooter,
  ModalHeader,
  ModalOverlay,
  useDisclosure,
} from "@chakra-ui/react"
import { useMemo } from "react"

export function useStopWorkspaceModal(onClick: (closeModal: VoidFunction) => void) {
  const { isOpen, onOpen, onClose } = useDisclosure()

  const modal = useMemo(
    () => (
      <Modal onClose={onClose} isOpen={isOpen} isCentered>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>Stop Workspace</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            Stopping a workspace while it&apos;s not running may leave it in a corrupted state. Do
            you want to stop it regardless?
          </ModalBody>
          <ModalFooter>
            <HStack spacing={"2"}>
              <Button onClick={onClose}>Close</Button>
              <Button colorScheme={"red"} onClick={() => onClick(onClose)}>
                Stop
              </Button>
            </HStack>
          </ModalFooter>
        </ModalContent>
      </Modal>
    ),
    [isOpen, onClick, onClose]
  )

  return { modal, open: onOpen }
}

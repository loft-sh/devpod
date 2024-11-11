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

export function useRebuildWorkspaceModal(
  workspaceName: string,
  onClick: (closeModal: VoidFunction) => void
) {
  const { isOpen, onOpen, onClose } = useDisclosure()

  const modal = useMemo(
    () => (
      <Modal onClose={onClose} isOpen={isOpen} isCentered>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>Rebuild Workspace</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            Rebuilding the workspace will erase all state saved in the docker container overlay.
            This means you might need to reinstall or reconfigure certain applications. State in
            docker volumes is persisted. Are you sure you want to rebuild {workspaceName}?
          </ModalBody>
          <ModalFooter>
            <HStack spacing={"2"}>
              <Button onClick={onClose}>Close</Button>
              <Button colorScheme={"primary"} onClick={() => onClick(onClose)}>
                Rebuild
              </Button>
            </HStack>
          </ModalFooter>
        </ModalContent>
      </Modal>
    ),
    [isOpen, onClick, onClose, workspaceName]
  )

  return { modal, open: onOpen }
}

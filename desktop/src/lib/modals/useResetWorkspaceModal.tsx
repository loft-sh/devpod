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

export function useResetWorkspaceModal(
  workspaceName: string,
  onClick: (closeModal: VoidFunction) => void
) {
  const { isOpen, onOpen, onClose } = useDisclosure()

  const modal = useMemo(
    () => (
      <Modal onClose={onClose} isOpen={isOpen} isCentered>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>Reset Workspace</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            Reseting the workspace will erase all state saved in the docker container overlay and
            DELETE ALL UNCOMMITTED CODE. This means you might need to reinstall or reconfigure
            certain applications. You will start with a fresh clone of the repository. Are you sure
            you want to rebuild {workspaceName}?
          </ModalBody>
          <ModalFooter>
            <HStack spacing={"2"}>
              <Button onClick={onClose}>Close</Button>
              <Button colorScheme={"primary"} onClick={() => onClick(onClose)}>
                Reset
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

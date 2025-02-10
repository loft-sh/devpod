import {
  Box,
  Button,
  Checkbox,
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
import { useMemo, useState } from "react"

export function useDeleteWorkspaceModal(
  workspaceName: string,
  onClick: (forceDelete: boolean, closeModal: VoidFunction) => void,
  force?: boolean
) {
  const { isOpen, onOpen, onClose } = useDisclosure()
  const [forceDelete, setForceDelete] = useState<boolean>(false)

  const modal = useMemo(
    () => (
      <Modal onClose={onClose} isOpen={isOpen} isCentered>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>Delete Workspace</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            Deleting the workspace will erase all state. Are you sure you want to delete{" "}
            {workspaceName}?
            {force == null && (
              <Box marginTop={"2.5"}>
                <Checkbox checked={forceDelete} onChange={(e) => setForceDelete(e.target.checked)}>
                  Force Delete the Workspace
                </Checkbox>
              </Box>
            )}
          </ModalBody>
          <ModalFooter>
            <HStack spacing={"2"}>
              <Button onClick={onClose}>Close</Button>
              <Button colorScheme={"red"} onClick={() => onClick(forceDelete, onClose)}>
                Delete
              </Button>
            </HStack>
          </ModalFooter>
        </ModalContent>
      </Modal>
    ),
    [force, forceDelete, isOpen, onClick, onClose, workspaceName]
  )

  return { modal, open: onOpen }
}

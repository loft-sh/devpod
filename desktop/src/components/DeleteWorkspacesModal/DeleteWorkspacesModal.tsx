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
} from "@chakra-ui/react"
import React, { useCallback, useState } from "react"

export function DeleteWorkspacesModal({
  onDeleteRequested,
  onCloseRequested,
  amount,
  isOpen,
}: {
  isOpen: boolean
  onCloseRequested: () => void
  onDeleteRequested: (forceDelete: boolean) => void
  amount: number
}) {
  const [forceDelete, setForceDelete] = useState(false)

  const onDeleteClick = useCallback(() => {
    onCloseRequested()
    onDeleteRequested(forceDelete)
  }, [forceDelete, onDeleteRequested, onCloseRequested])

  const onForceDeleteChanged = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      setForceDelete(e.target.checked)
    },
    [setForceDelete]
  )

  return (
    <Modal onClose={onCloseRequested} isOpen={isOpen} isCentered>
      <ModalOverlay />
      <ModalContent>
        <ModalHeader>Delete {amount} Workspaces</ModalHeader>
        <ModalCloseButton />
        <ModalBody>
          Deleting the workspaces will erase all state. Are you sure you want to delete the selected
          workspaces?
          <Box marginTop={"2.5"}>
            <Checkbox checked={forceDelete} onChange={onForceDeleteChanged}>
              Force Delete
            </Checkbox>
          </Box>
        </ModalBody>
        <ModalFooter>
          <HStack spacing={"2"}>
            <Button onClick={onCloseRequested}>Close</Button>
            <Button colorScheme="red" onClick={onDeleteClick}>
              Delete
            </Button>
          </HStack>
        </ModalFooter>
      </ModalContent>
    </Modal>
  )
}

import {
  Modal,
  ModalBody,
  ModalCloseButton,
  ModalContent,
  ModalFooter,
  ModalHeader,
  ModalOverlay,
  useDisclosure,
  VStack,
} from "@chakra-ui/react"
import { useCallback, useMemo, useState } from "react"
import { useNavigate } from "react-router-dom"
import { WarningMessageBox } from "../../../components/Warning"
import { Routes } from "../../../routes"
import { SetupProviderSteps } from "../../Providers"

export function useSetupProviderModal() {
  const navigate = useNavigate()
  const { isOpen, onClose, onOpen } = useDisclosure()
  const [message, setMessage] = useState("")
  const [isStrict, setIsStrict] = useState(true)
  const [wasDismissed, setWasDismissed] = useState(false)

  const show = useCallback(
    ({ message: newMessage, isStrict }: Readonly<{ isStrict: boolean; message?: string }>) => {
      if (isOpen) {
        return
      }

      if (newMessage) {
        setMessage(newMessage)
      }
      setIsStrict(isStrict)
      onOpen()
    },
    [isOpen, onOpen]
  )

  const handleCloseClicked = useCallback(() => {
    if (isStrict) {
      navigate(Routes.WORKSPACES)

      return
    }

    setWasDismissed(true)
  }, [isStrict, navigate])

  const modal = useMemo(
    () => (
      <Modal
        onClose={onClose}
        isOpen={isOpen}
        isCentered
        size="6xl"
        scrollBehavior="inside"
        closeOnOverlayClick={false}>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>Configure Provider</ModalHeader>
          <ModalCloseButton onClick={handleCloseClicked} />
          <ModalBody>
            <VStack align="start" spacing="8">
              <WarningMessageBox warning={message} />
              <SetupProviderSteps onFinish={onClose} />
            </VStack>
          </ModalBody>
          <ModalFooter />
        </ModalContent>
      </Modal>
    ),
    [onClose, isOpen, handleCloseClicked, message]
  )

  return { modal, show, wasDismissed }
}

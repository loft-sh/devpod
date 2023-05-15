import {
  Modal,
  ModalBody,
  ModalCloseButton,
  ModalContent,
  ModalHeader,
  ModalOverlay,
  useDisclosure,
  VStack,
} from "@chakra-ui/react"
import { useCallback, useMemo, useState } from "react"
import { useNavigate } from "react-router-dom"
import { Routes } from "../../routes"
import { TProviderID } from "../../types"
import { SetupProviderSteps } from "../Providers"

export function useSetupProviderModal() {
  const navigate = useNavigate()
  const { isOpen, onClose, onOpen } = useDisclosure()
  const [isStrict, setIsStrict] = useState(true)
  const [suggestedProvider, setSuggestedProvider] = useState<TProviderID | undefined>(undefined)
  const [wasDismissed, setWasDismissed] = useState(false)
  const [currentProviderID, setCurrentProviderID] = useState<string | null>(null)

  const show = useCallback(
    ({
      isStrict,
      suggestedProvider,
    }: Readonly<{ isStrict: boolean; suggestedProvider?: TProviderID }>) => {
      if (isOpen) {
        return
      }

      if (suggestedProvider) {
        setSuggestedProvider(suggestedProvider)
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

  const title = useMemo(() => {
    if (currentProviderID !== null) {
      return `Configure Provider ${currentProviderID}`
    }

    if (isStrict) {
      return "Configure Provider before creating a workspace"
    }

    return "Configure Provider"
  }, [currentProviderID, isStrict])

  const modal = useMemo(
    () => (
      <Modal
        onClose={onClose}
        isOpen={isOpen}
        isCentered
        size="6xl"
        scrollBehavior="inside"
        closeOnOverlayClick={true}>
        <ModalOverlay />
        <ModalContent position="relative" overflow="hidden">
          <ModalHeader>{title}</ModalHeader>
          <ModalCloseButton onClick={handleCloseClicked} />
          <ModalBody overflowX="hidden" overflowY="auto" paddingBottom="0">
            <VStack align="start" spacing="8">
              <SetupProviderSteps
                suggestedProvider={suggestedProvider}
                onProviderIDChanged={setCurrentProviderID}
                onFinish={onClose}
                isModal
              />
            </VStack>
          </ModalBody>
        </ModalContent>
      </Modal>
    ),
    [onClose, isOpen, title, handleCloseClicked, suggestedProvider]
  )

  return { modal, show, wasDismissed }
}

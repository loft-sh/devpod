import { useEffect, useId, useMemo, useRef, useState } from "react"
import { useNavigate } from "react-router"
import { client } from "./client"
import { startWorkspaceAction } from "./contexts"
import { Routes } from "./routes"
import { appWindow } from "@tauri-apps/api/window"
import {
  Modal,
  ModalBody,
  ModalCloseButton,
  ModalContent,
  ModalFooter,
  ModalHeader,
  ModalOverlay,
  useDisclosure,
} from "@chakra-ui/react"
import { ErrorMessageBox } from "./components"

export function useAppReady() {
  const isReadyLockRef = useRef<boolean>(false)
  const viewID = useId()
  const navigate = useNavigate()
  const [openWorkspaceFailedMessage, setOpenWorkspaceFailedMessage] = useState<string | null>(null)
  const { isOpen, onClose, onOpen } = useDisclosure()
  const modal = useMemo(() => {
    return (
      <Modal
        onClose={onClose}
        isOpen={isOpen}
        onCloseComplete={() => setOpenWorkspaceFailedMessage(null)}
        isCentered>
        <ModalOverlay />
        <ModalContent>
          <ModalCloseButton />
          <ModalHeader>Failed to open workspace from URL</ModalHeader>
          <ModalBody>
            <ErrorMessageBox error={Error(openWorkspaceFailedMessage!)} />
          </ModalBody>
          <ModalFooter />
        </ModalContent>
      </Modal>
    )
  }, [isOpen, onClose, openWorkspaceFailedMessage])

  useEffect(() => {
    if (openWorkspaceFailedMessage !== null) {
      onOpen()
    } else {
      onClose()
    }
  }, [onClose, onOpen, openWorkspaceFailedMessage])

  // notifies underlying layer that ui is ready for communication
  useEffect(() => {
    if (!isReadyLockRef.current) {
      isReadyLockRef.current = true
      ;(async () => {
        const unsubscribe = await client.subscribe("event", async (event) => {
          await appWindow.setFocus()
          console.log("received event", event)
          if (event.type === "ShowDashboard") {
            navigate(Routes.WORKSPACES)

            return
          }

          if (event.type === "OpenWorkspaceFailed") {
            const message = Object.entries(event)
              .filter(([key]) => key !== "type")
              .map(([key, value]) => `${key}: ${value}`)
              .join("\n")
            setOpenWorkspaceFailedMessage(message)

            return
          }

          const workspacesResult = await client.workspaces.listAll()
          if (workspacesResult.err) {
            return
          }

          // Try to find workspace by source
          const maybeWorkspace = workspacesResult.val.find((w) => {
            if (w.source === null) {
              return false
            }

            // Check `repo@branch` combo first
            if (`${w.source.gitRepository ?? ""}@${w.source.gitBranch ?? ""}` === event.source) {
              return true
            }

            // Check Git repo
            if (w.source.gitRepository === event.source) {
              return true
            }

            // Check local folder
            if (w.source.localFolder === event.source) {
              return true
            }

            // Check Docker Image
            if (w.source.image === event.source) {
              return true
            }

            return false
          })

          if (maybeWorkspace !== undefined) {
            const actionID = startWorkspaceAction({
              workspaceID: maybeWorkspace.id,
              streamID: viewID,
              config: {
                id: maybeWorkspace.id,
                providerConfig: {
                  providerID: maybeWorkspace.provider?.name ?? undefined,
                },
                ideConfig: {
                  name: maybeWorkspace.ide?.name ?? null,
                },
              },
            })

            navigate(Routes.toAction(actionID))

            return
          }

          navigate(
            Routes.toWorkspaceCreate({
              workspaceID: event.workspace_id,
              providerID: event.provider_id,
              rawSource: event.source,
              ide: event.ide,
            })
          )
        })

        try {
          await client.ready()
        } catch (err) {
          console.error(err)
        }

        return unsubscribe
      })()
    }
  }, [navigate, viewID])

  return { modal }
}

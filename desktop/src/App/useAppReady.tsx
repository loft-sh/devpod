import { QueryKeys } from "@/queryKeys"
import { TProInstance } from "@/types"
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
  Text,
  useDisclosure,
  useToast,
} from "@chakra-ui/react"
import { useQuery } from "@tanstack/react-query"
import { getCurrentWebviewWindow } from "@tauri-apps/api/webviewWindow"
import { useCallback, useEffect, useId, useMemo, useRef, useState } from "react"
import { matchPath, useNavigate } from "react-router"
import { client } from "../client"
import { ErrorMessageBox } from "../components"
import { WORKSPACE_SOURCE_BRANCH_DELIMITER, WORKSPACE_SOURCE_COMMIT_DELIMITER } from "../constants"
import {
  startWorkspaceAction,
  useChangeSettings,
  useProInstances,
  useWorkspaceStore,
} from "../contexts"
import { exists, hasCapability, useLoginProModal } from "../lib"
import { Routes } from "../routes"
import { useChangelogModal } from "./useChangelogModal"

export function useAppReady() {
  const [[proInstances]] = useProInstances()
  const { store } = useWorkspaceStore()
  const isReadyLockRef = useRef<boolean>(false)
  const viewID = useId()
  const navigate = useNavigate()
  const toast = useToast()
  const { modal: errorModal, setFailedMessage } = useErrorModal()
  const { modal: changelogModal } = useChangelogModal(isReadyLockRef.current)
  const { modal: proLoginModal, handleOpenLogin: handleProLogin } = useLoginProModal()
  const { set: setSetting } = useChangeSettings()

  // auto-update pro providers in the background
  useQuery({
    queryKey: QueryKeys.proProviderUpdates(proInstances),
    queryFn: async () => {
      if (!proInstances || proInstances.length === 0) {
        return null
      }

      // let pro client check for updates without using the provider
      // we don't really care about the result in the context of the GUI, just need to make sure it's updating
      await Promise.allSettled(
        proInstances
          .filter(
            (instance) =>
              instance.provider && instance.host && hasCapability(instance, "update-provider")
          )
          .map(async (instance) => {
            const proClient = client.getProClient(instance)
            const checkUpdateRes = await proClient.checkUpdate()
            if (checkUpdateRes.err) {
              client.log(
                "error",
                `[${instance.host ?? ""}] Failed to check for upgrade: ${
                  checkUpdateRes.val.message
                }`
              )

              return null
            }

            const { available: updateAvailable, newVersion } = checkUpdateRes.val
            if (!updateAvailable || !newVersion) {
              return null
            }
            client.log(
              "info",
              `[${
                instance.host ?? ""
              }] New version available (${newVersion}). Attempting to update.`
            )

            const updateRes = await proClient.update(newVersion)
            if (updateRes.err) {
              client.log(
                "error",
                `[${instance.host ?? ""}] Failed to upgrade: ${updateRes.val.message}`
              )

              return null
            }

            client.log("info", `[${instance.host ?? ""}] Successfully updated to ${newVersion}`)
          })
      )

      return null
    },
    enabled: proInstances && proInstances.length > 0,
    refetchInterval: 1_000 * 60 * 5, // 5 minutes
  })

  useEffect(() => {
    window.addEventListener("contextmenu", (e) => {
      e.preventDefault()

      return false
    })
  }, [])

  const handleMessage: Parameters<typeof client.subscribe>[1] = useCallback(
    async (event) => {
      if (event.type === "ShowDashboard") {
        if (await getCurrentWebviewWindow().isMinimized()) {
          await getCurrentWebviewWindow().unminimize()
        }

        if (!(await getCurrentWebviewWindow().isVisible())) {
          await getCurrentWebviewWindow().show()
        }

        await getCurrentWebviewWindow().setFocus()

        return
      }

      if (event.type === "ShowToast") {
        await getCurrentWebviewWindow().setFocus()

        toast({
          title: event.title,
          description: event.message,
          status: event.status,
          duration: 5_000,
          isClosable: true,
        })

        return
      }

      if (event.type === "CommandFailed") {
        await getCurrentWebviewWindow().setFocus()
        const message = Object.entries(event)
          .filter(([key]) => key !== "type")
          .map(([key, value]) => `${key}: ${value}`)
          .join("\n")
        setFailedMessage(message)

        return
      }

      if (event.type === "LoginRequired") {
        const proInstances = await client.pro.listProInstances()
        if (proInstances.err) {
          return
        }
        const existingInstance = proInstances.val.find((i) => i.host === event.host)
        if (!existingInstance) {
          return
        }

        await getCurrentWebviewWindow().setFocus()
        const match = matchPath(Routes.toProInstance(event.host), location.pathname)
        if (match != null) {
          // only show toast if we're not on pro instance page anyway
          return
        }
        toast({
          title: "Login Required",
          description: (
            <HStack>
              <Text>You have been logged out. Please log back in.</Text>
              <Button
                ml="2"
                variant="ghost"
                onClick={() => navigate(Routes.toProInstance(event.host))}>
                Log in
              </Button>
            </HStack>
          ),
          status: "warning",
          duration: 5_000,
          isClosable: true,
        })

        return
      }

      if (event.type === "SetupPro") {
        // check if host is already taken. If not, set window to foreground and pass evnet to pro login handler
        const proInstances = await client.pro.listProInstances()
        if (proInstances.err) {
          return
        }

        const existingInstance = proInstances.val.find((i) => i.host === event.host)
        if (existingInstance) {
          // only warn in console, don't show modal
          console.warn("Pro instance already exists", existingInstance)

          return
        }

        const data: Parameters<typeof handleProLogin>[0] = {
          host: event.host,
          suggestedOptions: {},
        }
        if (event.accessKey) {
          data.accessKey = event.accessKey
        }
        if (event.options) {
          data.suggestedOptions = event.options
        }

        await getCurrentWebviewWindow().setFocus()
        // ensure pro is enabled
        setSetting("experimental_devPodPro", true)
        handleProLogin(data)

        return
      }

      if (event.type === "OpenProInstance") {
        const proInstances = await client.pro.listProInstances()
        if (proInstances.err) {
          return
        }

        const existingInstance = proInstances.val.find((i) => i.host === event.host)
        if (!existingInstance?.host) {
          return
        }

        await getCurrentWebviewWindow().setFocus()
        navigate(Routes.toProInstance(existingInstance.host))

        return
      }

      if (event.type === "ImportWorkspace") {
        await getCurrentWebviewWindow().setFocus()
        // Do we already know the workspace?
        let workspacesResult = await client.workspaces.listAll(false)
        if (workspacesResult.err) {
          const cleanedMsg = workspacesResult.val.message.split("\n").at(-1) ?? ""
          setFailedMessage("Failed to list workspaces: " + cleanedMsg)

          return
        }
        let maybeWorkspace = workspacesResult.val.find((w) => w.id === event.workspace_id)
        // Is it a pro workspace?
        if (maybeWorkspace && maybeWorkspace.provider?.name) {
          const proInstance = await findProInstance(maybeWorkspace.provider.name)
          if (proInstance && proInstance.host) {
            navigate(Routes.toProWorkspace(proInstance.host, maybeWorkspace.id))

            return
          }
        }

        // At this point it can't be a new pro workspace anymore,
        // we'll have to go through the old import flow
        const importResult = await client.pro.importWorkspace({
          workspaceID: event.workspace_id,
          workspaceUID: event.workspace_uid,
          devPodProHost: event.devpod_pro_host,
          project: event.project,
          options: event.options,
        })
        if (importResult.err) {
          const cleanedMsg = importResult.val.message.split("\n").at(-1) ?? ""
          setFailedMessage("Failed to import workspace: " + cleanedMsg)

          return
        }
        workspacesResult = await client.workspaces.listAll(false)
        if (workspacesResult.err) {
          return
        }
        maybeWorkspace = workspacesResult.val.find((w) => w.id === event.workspace_id)
        if (!maybeWorkspace) {
          setFailedMessage("Could not find workspace after import")

          return
        }

        const actionID = startWorkspaceAction({
          workspaceID: maybeWorkspace.id,
          streamID: viewID,
          config: {
            id: maybeWorkspace.id,
            providerConfig: {
              providerID: maybeWorkspace.provider?.name ?? undefined,
            },
            ideConfig: {
              name: maybeWorkspace.ide?.name,
            },
          },
          store,
        })
        navigate(Routes.toAction(actionID))

        return
      }

      // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
      if (event.type === "OpenWorkspace") {
        const workspacesResult = await client.workspaces.listAll(false)
        if (workspacesResult.err) {
          return
        }

        // Try to find workspace by source
        let maybeWorkspace = workspacesResult.val.find((w) => {
          if (!w.source) {
            return false
          }

          // Check `repo@sha256:commitHash`
          if (
            `${w.source.gitRepository ?? ""}${WORKSPACE_SOURCE_COMMIT_DELIMITER}${
              w.source.gitCommit ?? ""
            }` === event.source
          ) {
            return true
          }

          // Check `repo@branch`
          if (
            `${w.source.gitRepository ?? ""}${WORKSPACE_SOURCE_BRANCH_DELIMITER}${
              w.source.gitBranch ?? ""
            }` === event.source
          ) {
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

        // If we don't have a workspace by now, `source` isn't defined but `workspace_id` is, try to find workspace by ID
        // This happens for example if the message is triggered by a system tray item
        // WARN: `event.source` can be an empty string here, hence the falsy check
        if (maybeWorkspace === undefined && !event.source && exists(event.workspace_id)) {
          maybeWorkspace = workspacesResult.val.find((w) => w.id === event.workspace_id)
        }

        const ides = await client.ides.listAll()
        let defaultIDE = undefined
        if (ides.ok) {
          defaultIDE = ides.val.find((ide) => ide.default)?.name
        }

        const providerName = maybeWorkspace?.provider?.name
        if (maybeWorkspace !== undefined && providerName) {
          const proInstance = await findProInstance(providerName)
          if (proInstance && proInstance.host) {
            navigate(Routes.toProWorkspace(proInstance.host, maybeWorkspace.id))

            return
          }

          const actionID = startWorkspaceAction({
            workspaceID: maybeWorkspace.id,
            streamID: viewID,
            config: {
              id: maybeWorkspace.id,
              providerConfig: { providerID: providerName },
              ideConfig: { name: defaultIDE ?? maybeWorkspace.ide?.name ?? null },
            },
            store,
          })

          navigate(Routes.toAction(actionID))

          return
        }

        const match = matchPath(Routes.PRO_INSTANCE, location.pathname)
        if (match && match.params.host) {
          navigate(Routes.toProWorkspaceCreate(match.params.host))

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
      }
    },
    [handleProLogin, navigate, setFailedMessage, setSetting, store, toast, viewID]
  )

  // notifies underlying layer that ui is ready for communication
  useEffect(() => {
    const unsubscribePromise = client.subscribe("event", handleMessage)
    if (!isReadyLockRef.current) {
      isReadyLockRef.current = true

      unsubscribePromise.then(async () => {
        try {
          await client.ready()
        } catch (err) {
          return console.error(err)
        }
      })
    }

    return () => {
      unsubscribePromise.then((unsubscribe) => {
        unsubscribe()
      })
    }
  }, [handleMessage, navigate])

  return { errorModal, changelogModal, proLoginModal }
}

function useErrorModal() {
  const [failedMessage, setFailedMessage] = useState<string | null>(null)
  const { isOpen, onClose, onOpen } = useDisclosure()
  const modal = useMemo(() => {
    return (
      <Modal
        onClose={onClose}
        isOpen={isOpen}
        onCloseComplete={() => setFailedMessage(null)}
        isCentered>
        <ModalOverlay />
        <ModalContent>
          <ModalCloseButton />
          {/* todo: customize the header */}
          <ModalHeader>Failed to open workspace from URL</ModalHeader>
          <ModalBody>
            <ErrorMessageBox error={Error(failedMessage!)} />
          </ModalBody>
          <ModalFooter />
        </ModalContent>
      </Modal>
    )
  }, [isOpen, onClose, failedMessage])

  useEffect(() => {
    if (failedMessage !== null) {
      onOpen()
    } else {
      onClose()
    }
  }, [onClose, onOpen, failedMessage])

  return { modal, handleOpen: onOpen, setFailedMessage }
}

async function findProInstance(providerName: string): Promise<TProInstance | null> {
  const providersRes = await client.providers.listAll()
  if (providersRes.err) return null
  const provider = providersRes.val[providerName]
  if (!provider || !provider.isProxyProvider) return null

  // handle pro provider
  const proInstanceRes = await client.pro.listProInstances()
  if (proInstanceRes.err) return null
  const proInstance = proInstanceRes.val.find(
    (proInstance) => proInstance.provider === providerName
  )
  if (!proInstance?.host) return null

  return proInstance
}

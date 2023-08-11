import {
  Box,
  Button,
  Heading,
  Link,
  ListItem,
  Modal,
  ModalBody,
  ModalCloseButton,
  ModalContent,
  ModalFooter,
  ModalHeader,
  ModalOverlay,
  Text,
  UnorderedList,
  useDisclosure,
  useToast,
} from "@chakra-ui/react"
import { appWindow } from "@tauri-apps/api/window"
import Markdown from "markdown-to-jsx"
import { useEffect, useId, useMemo, useRef, useState } from "react"
import { useNavigate } from "react-router"
import { client } from "./client"
import { ErrorMessageBox } from "./components"
import { WORKSPACE_SOURCE_BRANCH_DELIMITER, WORKSPACE_SOURCE_COMMIT_DELIMITER } from "./constants"
import { startWorkspaceAction } from "./contexts"
import { Release } from "./gen"
import { exists, useReleases, useVersion } from "./lib"
import { Routes } from "./routes"

const LAST_INSTALLED_VERSION_KEY = "devpod-last-installed-version"
type TLinkClickEvent = React.MouseEvent<HTMLLinkElement> & { target: HTMLLinkElement }

export function useAppReady() {
  const isReadyLockRef = useRef<boolean>(false)
  const currentVersion = useVersion()
  const viewID = useId()
  const navigate = useNavigate()
  const [openWorkspaceFailedMessage, setOpenWorkspaceFailedMessage] = useState<string | null>(null)
  const { isOpen, onClose, onOpen } = useDisclosure()
  const toast = useToast()
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

  const releases = useReleases()
  const {
    isOpen: isChangelogModalOpen,
    onClose: onChangelogModalClose,
    onOpen: onChangelogModalOpen,
  } = useDisclosure()
  const [latestRelease, setLatestRelease] = useState<Release | null>(null)
  const changelogModal = useMemo(
    () =>
      latestRelease !== null ? (
        <Modal
          onClose={onChangelogModalClose}
          isOpen={isChangelogModalOpen}
          onCloseComplete={() => setOpenWorkspaceFailedMessage(null)}
          scrollBehavior="inside"
          size="3xl"
          isCentered>
          <ModalOverlay />
          <ModalContent>
            <ModalCloseButton />
            <ModalHeader>Installed Version {latestRelease.tag_name}</ModalHeader>
            <ModalBody>
              {latestRelease.body ? (
                <Changelog rawMarkdown={latestRelease.body} />
              ) : (
                <Text>This release doesn&apos;t have a changelog</Text>
              )}
            </ModalBody>
            <ModalFooter>
              <Button onClick={onChangelogModalClose}>Done</Button>
            </ModalFooter>
          </ModalContent>
        </Modal>
      ) : null,
    [isChangelogModalOpen, latestRelease, onChangelogModalClose]
  )

  useEffect(() => {
    if (!isReadyLockRef.current || !currentVersion || !releases) {
      return
    }

    const latestVersion = localStorage.getItem(LAST_INSTALLED_VERSION_KEY)
    const maybeRelease = releases.find((r) => r.tag_name === `v${currentVersion}`)

    if (latestVersion !== currentVersion) {
      localStorage.setItem(LAST_INSTALLED_VERSION_KEY, currentVersion)

      if (maybeRelease !== undefined) {
        setLatestRelease(maybeRelease)
        onChangelogModalOpen()
      }
    }
  }, [currentVersion, navigate, onChangelogModalOpen, releases])

  useEffect(() => {
    if (openWorkspaceFailedMessage !== null) {
      onOpen()
    } else {
      onClose()
    }
  }, [onClose, onOpen, openWorkspaceFailedMessage])

  useEffect(() => {
    window.addEventListener("contextmenu", (e) => {
      e.preventDefault()

      return false
    })
  }, [])

  // notifies underlying layer that ui is ready for communication
  useEffect(() => {
    if (!isReadyLockRef.current) {
      isReadyLockRef.current = true
      ;(async () => {
        const unsubscribe = await client.subscribe("event", async (event) => {
          await appWindow.setFocus()
          if (event.type === "ShowDashboard") {
            navigate(Routes.WORKSPACES)

            return
          }

          if (event.type === "ShowToast") {
            toast({
              title: event.title,
              description: event.message,
              status: event.status,
              duration: 5_000,
              isClosable: true,
            })

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
                  name: defaultIDE ?? maybeWorkspace.ide?.name ?? null,
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
  }, [navigate, toast, viewID])

  return { modal, changelogModal }
}

type TChangeLogProps = Readonly<{ rawMarkdown: string }>
function Changelog({ rawMarkdown }: TChangeLogProps) {
  return (
    <Box padding="2" marginBottom="4">
      <Markdown
        options={{
          overrides: {
            h2: {
              component: Heading,
              props: {
                size: "md",
                marginBottom: "2",
                marginTop: "4",
              },
            },
            a: {
              component: Link,
              props: {
                onClick: (e: TLinkClickEvent) => {
                  e.preventDefault()
                  client.openLink(e.target.href)
                },
              },
            },
            ul: {
              component: UnorderedList,
            },
            li: {
              component: ListItem,
            },
          },
        }}>
        {rawMarkdown}
      </Markdown>
    </Box>
  )
}

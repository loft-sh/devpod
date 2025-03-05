import { Template } from "@/icons"
import { useStoreTroubleshoot } from "@/lib/useStoreTroubleshoot"
import {
  Box,
  Card,
  CardHeader,
  Icon,
  List,
  ListItem,
  Modal,
  ModalBody,
  ModalCloseButton,
  ModalContent,
  ModalHeader,
  ModalOverlay,
  Text,
  useDisclosure,
} from "@chakra-ui/react"
import { useCallback, useMemo, useRef, useState } from "react"
import { HiServerStack } from "react-icons/hi2"
import { useNavigate } from "react-router"
import { IconTag, WorkspaceCardHeader } from "../../components"
import {
  TActionID,
  useProvider,
  useSettings,
  useWorkspace,
  useWorkspaceActions,
} from "../../contexts"
import {
  getIDEName,
  getWorkspaceSourceName,
  useDeleteWorkspaceModal,
  useRebuildWorkspaceModal,
  useResetWorkspaceModal,
  useStopWorkspaceModal,
} from "../../lib"
import { Routes } from "../../routes"
import { TProvider, TWorkspace, TWorkspaceID } from "../../types"
import { useIDEs } from "../../useIDEs"
import { ConfigureProviderOptionsForm } from "../Providers"
import { processDisplayOptions } from "../Providers/AddProvider/useProviderOptions"
import { TOptionWithID, mergeOptionDefinitions } from "../Providers/helpers"
import { WorkspaceControls } from "./WorkspaceControls"
import { WorkspaceStatusBadge } from "./WorkspaceStatusBadge"

type TWorkspaceCardProps = Readonly<{
  workspaceID: TWorkspaceID
  isSelected?: boolean
  onSelectionChange?: (isSelected: boolean) => void
}>

export function WorkspaceCard({ workspaceID, isSelected, onSelectionChange }: TWorkspaceCardProps) {
  const changeOptionsModalBodyRef = useRef<HTMLDivElement>(null)
  const settings = useSettings()
  const navigate = useNavigate()
  const { ides, defaultIDE } = useIDEs()
  const workspace = useWorkspace<TWorkspace>(workspaceID)
  const workspaceName = workspace.data?.id ?? ""
  const workspaceActions = useWorkspaceActions(workspaceName)
  const navigateToAction = useCallback(
    (actionID: TActionID | undefined) => {
      if (actionID !== undefined && actionID !== "") {
        navigate(Routes.toAction(actionID))
      }
    },
    [navigate]
  )
  const { modal: stopModal, open: openStopModal } = useStopWorkspaceModal(
    useCallback(
      (close) => {
        workspace.stop()
        close()
      },
      [workspace]
    )
  )
  const { modal: deleteModal, open: openDeleteModal } = useDeleteWorkspaceModal(
    workspaceName,
    useCallback(
      (force, close) => {
        workspace.remove(force)
        close()
      },
      [workspace]
    )
  )
  const { modal: rebuildModal, open: openRebuildModal } = useRebuildWorkspaceModal(
    workspaceName,
    useCallback(
      (close) => {
        const actionID = workspace.rebuild()
        close()
        navigateToAction(actionID)
      },
      [navigateToAction, workspace]
    )
  )
  const { modal: resetModal, open: openResetModal } = useResetWorkspaceModal(
    workspaceName,
    useCallback(
      (close) => {
        const actionID = workspace.reset()
        close()
        navigateToAction(actionID)
      },
      [navigateToAction, workspace]
    )
  )
  const {
    isOpen: isChangeOptionsOpen,
    onOpen: handleChangeOptionsClicked,
    onClose: onChangeOptionsClose,
  } = useDisclosure()
  const { store: storeTroubleshoot } = useStoreTroubleshoot()

  const [provider] = useProvider(workspace.data?.provider?.name)
  const [ideName, setIdeName] = useState<string | undefined>(() => {
    if (settings.fixedIDE && defaultIDE?.name) {
      return defaultIDE.name
    }

    return workspace.data?.ide?.name ?? undefined
  })
  const ideDisplayName =
    ideName !== undefined
      ? getIDEName({ name: ideName }, ides)
      : getIDEName(workspace.data?.ide, ides)

  const handleLogsClicked = useCallback(() => {
    let actionID = workspace.current?.id
    if (actionID === undefined) {
      actionID = workspace.checkStatus()
    }

    navigateToAction(actionID)
  }, [navigateToAction, workspace])

  const handleTroubleshootClicked = useCallback(() => {
    if (workspaceActions && workspace.data) {
      storeTroubleshoot({
        workspace: workspace.data,
        workspaceActions: workspaceActions,
      })
    }
  }, [storeTroubleshoot, workspace.data, workspaceActions])

  const hasError = useMemo<boolean>(() => {
    if (!workspaceActions?.length || workspaceActions[0]?.status !== "error") {
      return false
    }

    return true
  }, [workspaceActions])

  const handleBadgeClicked = useCallback(() => {
    if (workspace.current !== undefined) {
      navigateToAction(workspace.current.id)
    }

    if (workspace.data?.status === undefined || workspace.data.status === "NotFound") {
      const actionID = workspace.checkStatus()
      navigateToAction(actionID)
    }

    const maybeLastAction = workspaceActions?.[0]
    if (maybeLastAction) {
      navigateToAction(maybeLastAction.id)
    }

    return undefined
  }, [navigateToAction, workspace, workspaceActions])

  const handleChangeOptionsFinishClicked = (extraProviderOptions: Record<string, string>) => {
    // diff against current workspace options
    let changedOptions: Record<string, string> | undefined = undefined
    if (Object.keys(extraProviderOptions).length > 0) {
      changedOptions = {}
      const workspaceOptions = workspace.data?.provider?.options ?? {}
      for (const [k, v] of Object.entries(extraProviderOptions)) {
        // check if current workspace option doesn't contain option or it does but value is different
        if (!workspaceOptions[k] || workspaceOptions[k]?.value !== v) {
          changedOptions[k] = v
        }
      }
    }
    const actionID = workspace.start({
      id: workspaceID,
      providerConfig: { options: changedOptions },
    })
    onChangeOptionsClose()
    navigateToAction(actionID)
  }

  const isLoading = workspace.current?.status === "pending"

  if (workspace.data === undefined) {
    return null
  }

  const maybeRunnerName = getRunnerName(workspace.data, provider)
  const maybeTemplate = getTemplate(workspace.data, provider)
  const maybeTemplateOptions = getTemplateOptions(workspace.data, provider)

  return (
    <>
      <Card
        key={workspace.data.id}
        direction="row"
        width="full"
        maxWidth="60rem"
        variant="outline"
        marginBottom="3">
        <CardHeader overflow="hidden" w="full">
          <WorkspaceCardHeader
            id={workspace.data.id}
            source={
              workspace.data.source && (
                <Text
                  fontSize="sm"
                  color={"gray.600"}
                  _dark={{ color: "gray.300" }}
                  userSelect="text"
                  maxWidth="30rem"
                  overflow="hidden"
                  whiteSpace="nowrap"
                  textOverflow="ellipsis"
                  marginTop={-0.5}
                  _hover={{ overflow: "visible", cursor: "text" }}>
                  {getWorkspaceSourceName(workspace.data.source)}
                </Text>
              )
            }
            isSelected={isSelected}
            onSelectionChange={onSelectionChange}
            statusBadge={
              <WorkspaceStatusBadge
                status={workspace.data.status}
                isLoading={isLoading}
                hasError={hasError}
                onClick={handleBadgeClicked}
              />
            }
            controls={
              <WorkspaceControls
                id={workspace.data.id}
                workspace={workspace}
                provider={provider}
                isLoading={isLoading}
                isIDEFixed={settings.fixedIDE}
                ides={ides}
                ideName={ideName}
                setIdeName={setIdeName}
                navigateToAction={navigateToAction}
                onRebuildClicked={openRebuildModal}
                onResetClicked={openResetModal}
                onDeleteClicked={openDeleteModal}
                onStopClicked={openStopModal}
                onLogsClicked={handleLogsClicked}
                onTroubleshootClicked={handleTroubleshootClicked}
                onChangeOptionsClicked={handleChangeOptionsClicked}
              />
            }>
            <WorkspaceCardHeader.Provider name={workspace.data.provider?.name ?? undefined} />
            <WorkspaceCardHeader.IDE name={ideDisplayName} />
            <WorkspaceCardHeader.LastUsed timestamp={workspace.data.lastUsed} />
            {maybeTemplate && (
              <IconTag
                icon={<Template />}
                label={maybeTemplate}
                info={
                  <Box width="full">
                    Using {maybeTemplate} template with options: <br />
                    {maybeTemplateOptions.length > 0 ? (
                      <List mt="2" width="full">
                        {maybeTemplateOptions.map((opt) => (
                          <ListItem
                            key={opt.id}
                            width="full"
                            display="flex"
                            flexFlow="row nowrap"
                            alignItems="space-between">
                            <Text fontWeight="bold">{opt.value}</Text>
                            <Text ml="4">({opt.displayName || opt.id})</Text>
                          </ListItem>
                        ))}
                      </List>
                    ) : (
                      "No options configured"
                    )}
                  </Box>
                }
              />
            )}
            {maybeRunnerName && (
              <IconTag
                icon={<Icon as={HiServerStack} />}
                label={maybeRunnerName}
                info={`Running on ${maybeRunnerName}`}
              />
            )}
          </WorkspaceCardHeader>
        </CardHeader>
      </Card>

      {resetModal}
      {rebuildModal}
      {deleteModal}
      {stopModal}

      <Modal
        onClose={onChangeOptionsClose}
        isOpen={isChangeOptionsOpen}
        isCentered
        size="4xl"
        scrollBehavior="inside">
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>Change Options</ModalHeader>
          <ModalCloseButton />
          <ModalBody
            ref={changeOptionsModalBodyRef}
            overflowX="hidden"
            overflowY="auto"
            paddingBottom="0">
            {workspace.data.provider?.name ? (
              <ConfigureProviderOptionsForm
                workspace={workspace.data}
                showBottomActionBar
                isModal
                submitTitle="Update &amp; Open"
                containerRef={changeOptionsModalBodyRef}
                reuseMachine={false}
                providerID={workspace.data.provider.name}
                onFinish={handleChangeOptionsFinishClicked}
              />
            ) : (
              <>Unable to find provider for this workspace</>
            )}
          </ModalBody>
        </ModalContent>
      </Modal>
    </>
  )
}

function getRunnerName(workspace: TWorkspace, provider: TProvider | undefined): string | undefined {
  const options = mergeOptionDefinitions(
    workspace.provider?.options ?? {},
    provider?.config?.options ?? {}
  )
  const maybeRunnerOption = options["LOFT_RUNNER"]
  if (!maybeRunnerOption) {
    return undefined
  }
  const value = maybeRunnerOption.value

  return maybeRunnerOption.enum?.find((e) => e.value === value)?.displayName ?? value ?? undefined
}

function getTemplate(workspace: TWorkspace, provider: TProvider | undefined): string | undefined {
  const options = mergeOptionDefinitions(
    workspace.provider?.options ?? {},
    provider?.config?.options ?? {}
  )
  const maybeTemplateOption = options["LOFT_TEMPLATE"]
  if (!maybeTemplateOption) {
    return undefined
  }
  const value = maybeTemplateOption.value

  return maybeTemplateOption.enum?.find((e) => e.value === value)?.displayName ?? value ?? undefined
}

function getTemplateOptions(
  workspace: TWorkspace,
  provider: TProvider | undefined
): readonly TOptionWithID[] {
  const options = mergeOptionDefinitions(
    workspace.provider?.options ?? {},
    provider?.config?.options ?? {}
  )
  const displayOptions = processDisplayOptions(options, [], true)

  // shouldn't have groups here as we passed in empty array earlier
  return [...displayOptions.required, ...displayOptions.other].filter(
    (opt) => opt.id !== "LOFT_TEMPLATE"
  )
}

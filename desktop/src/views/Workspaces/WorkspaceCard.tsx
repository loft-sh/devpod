import { Card, CardHeader, Text } from "@chakra-ui/react"
import { useCallback, useMemo, useState } from "react"
import { useNavigate } from "react-router"
import { WorkspaceCardHeader } from "../../components"
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
import { TWorkspace, TWorkspaceID } from "../../types"
import { useIDEs } from "../../useIDEs"
import { WorkspaceControls } from "./WorkspaceControls"
import { WorkspaceStatusBadge } from "./WorkspaceStatusBadge"

type TWorkspaceCardProps = Readonly<{
  workspaceID: TWorkspaceID
  isSelected?: boolean
  onSelectionChange?: (isSelected: boolean) => void
}>

export function WorkspaceCard({ workspaceID, isSelected, onSelectionChange }: TWorkspaceCardProps) {
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

  const isLoading = workspace.current?.status === "pending"

  if (workspace.data === undefined) {
    return null
  }

  return (
    <>
      <Card
        key={workspace.data.id}
        direction="row"
        width="full"
        maxWidth="60rem"
        variant="outline"
        backgroundColor={isSelected ? "gray.50" : "transparent"}
        marginBottom="3">
        <CardHeader overflow="hidden" w="full">
          <WorkspaceCardHeader
            id={workspace.data.id}
            source={
              workspace.data.source && (
                <Text
                  fontSize="sm"
                  color="gray.500"
                  userSelect="auto"
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
              />
            }>
            <WorkspaceCardHeader.Provider name={workspace.data.provider?.name ?? undefined} />
            <WorkspaceCardHeader.IDE name={ideDisplayName} />
            <WorkspaceCardHeader.LastUsed timestamp={workspace.data.lastUsed} />
          </WorkspaceCardHeader>
        </CardHeader>
      </Card>

      {resetModal}
      {rebuildModal}
      {deleteModal}
      {stopModal}
    </>
  )
}

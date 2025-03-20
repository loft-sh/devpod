import {
  ProWorkspaceInstance,
  useProjectClusters,
  useTemplates,
  useWorkspace,
  useWorkspaceActions,
} from "@/contexts"
import {
  getDisplayName,
  useDeleteWorkspaceModal,
  useRebuildWorkspaceModal,
  useResetWorkspaceModal,
  useStopWorkspaceModal,
} from "@/lib"
import { useStoreTroubleshoot } from "@/lib/useStoreTroubleshoot"
import { Routes } from "@/routes"
import { Card, CardBody, CardHeader, useColorModeValue } from "@chakra-ui/react"
import { useCallback, useMemo } from "react"
import { useNavigate } from "react-router"
import { WorkspaceCardHeader } from "./WorkspaceCardHeader"
import { WorkspaceDetails } from "./WorkspaceDetails"
import { useTemplate } from "./useTemplate"

type TWorkspaceInstanceCardProps = Readonly<{
  host: string
  instanceName: string
  isSelected?: boolean
  onSelectionChange?: (isSelected: boolean) => void
}>

export function WorkspaceInstanceCard({
  instanceName,
  host,
  isSelected,
  onSelectionChange,
}: TWorkspaceInstanceCardProps) {
  const bgColor = useColorModeValue("white", "gray.900")
  const hoverBorderColor = useColorModeValue("primary.600", "primary.300")
  const workspace = useWorkspace<ProWorkspaceInstance>(instanceName)
  const instance = workspace.data
  const instanceDisplayName = getDisplayName(instance)
  const workspaceActions = useWorkspaceActions(instance?.id)
  const { data: projectClusters } = useProjectClusters()

  const navigate = useNavigate()

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
    instanceDisplayName,
    useCallback(
      (_, close) => {
        workspace.remove(true)
        close()
      },
      [workspace]
    ),
    true
  )

  const { modal: rebuildModal, open: openRebuildModal } = useRebuildWorkspaceModal(
    instanceDisplayName,
    useCallback(
      (close) => {
        workspace.rebuild()
        close()
        instance?.id && navigate(Routes.toProWorkspace(host, instance.id))
      },
      [workspace, navigate, host, instance]
    )
  )

  const { modal: resetModal, open: openResetModal } = useResetWorkspaceModal(
    instanceDisplayName,
    useCallback(
      (close) => {
        workspace.reset()
        close()
        instance?.id && navigate(Routes.toProWorkspace(host, instance.id))
      },
      [workspace, navigate, host, instance]
    )
  )

  const { store: storeTroubleshoot } = useStoreTroubleshoot()

  const handleTroubleshootClicked = useCallback(() => {
    if (instance && workspaceActions) {
      storeTroubleshoot({
        workspace: instance,
        workspaceActions: workspaceActions,
      })
    }
  }, [storeTroubleshoot, instance, workspaceActions])

  if (!instance) {
    return null
  }

  const handleOpenClicked = (ideName: string) => {
    workspace.start({ id: instance.id, ideConfig: { name: ideName } })
    navigate(Routes.toProWorkspace(host, instance.id))
  }

  const cluster = useMemo(() => {
    if (instance?.spec?.runnerRef?.runner) {
      return projectClusters?.runners?.find(
        (runner) => runner.metadata?.name === instance?.spec?.runnerRef?.runner
      )
    }

    return projectClusters?.clusters?.find(
      (cluster) => cluster.metadata?.name === instance?.spec?.target?.cluster?.name
    )
  }, [projectClusters, instance])

  const { template, parameters } = useTemplate(instance)

  const isRunning = instance.status?.lastWorkspaceStatus === "Running"

  return (
    <>
      <Card
        direction="column"
        width="full"
        variant="outline"
        marginBottom="3"
        paddingLeft="2"
        bg={bgColor}
        _hover={{ borderColor: hoverBorderColor, cursor: "pointer" }}
        boxShadow="0px 2px 4px 0px rgba(0, 0, 0, 0.07)"
        onClick={() => navigate(Routes.toProWorkspace(host, instance.id))}>
        <CardHeader overflow="hidden" w="full" pb="2">
          <WorkspaceCardHeader
            showSelection={true}
            isSelected={isSelected}
            onSelectionChange={onSelectionChange}
            instance={instance}>
            <WorkspaceCardHeader.Controls
              onOpenClicked={handleOpenClicked}
              onDeleteClicked={openDeleteModal}
              onRebuildClicked={openRebuildModal}
              onResetClicked={openResetModal}
              onStopClicked={!isRunning ? openStopModal : workspace.stop}
              onTroubleshootClicked={handleTroubleshootClicked}
            />
          </WorkspaceCardHeader>
        </CardHeader>
        <CardBody py="0">
          <WorkspaceDetails
            instance={instance}
            template={template}
            cluster={cluster}
            parameters={parameters}
            showDetails={false}
          />
        </CardBody>
      </Card>

      {resetModal}
      {rebuildModal}
      {deleteModal}
      {stopModal}
    </>
  )
}

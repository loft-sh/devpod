import { WarningMessageBox } from "@/components"
import {
  ProWorkspaceInstance,
  useProContext,
  useProjectClusters,
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
import { Box, Center, Spinner, Text, VStack } from "@chakra-ui/react"
import { useCallback, useEffect, useMemo } from "react"
import { useNavigate, useParams } from "react-router-dom"
import { BackToWorkspaces } from "../BackToWorkspaces"
import { WorkspaceTabs } from "./Tabs"
import { WorkspaceCardHeader } from "./WorkspaceCardHeader"
import { WorkspaceDetails } from "./WorkspaceDetails"
import { useTemplate } from "./useTemplate"
import { useMutation } from "@tanstack/react-query"

export function Workspace() {
  const params = useParams<{ workspace: string }>()
  const { data: projectClusters } = useProjectClusters()
  const { host, isLoadingWorkspaces, client } = useProContext()
  const navigate = useNavigate()
  const workspace = useWorkspace<ProWorkspaceInstance>(params.workspace)
  const instance = workspace.data
  const instanceDisplayName = getDisplayName(instance)
  const workspaceActions = useWorkspaceActions(instance?.id)

  const { mutate: updateWorkspaceDisplayName } = useMutation({
    mutationFn: async ({ newName }: Readonly<{ newName: string }>) => {
      if (!instance) {
        return
      }
      const updatedWorkspace = new ProWorkspaceInstance(instance)
      if (!updatedWorkspace.spec) {
        updatedWorkspace.spec = {}
      }
      updatedWorkspace.spec.displayName = newName

      return (await client.updateWorkspace(updatedWorkspace)).unwrap()
    },
  })

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
  const { template, parameters } = useTemplate(instance)

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

  // navigate to pro instance view after successfully deleting the workspace
  useEffect(() => {
    if (workspace.current?.name === "remove" && workspace.current.status === "success") {
      navigate(Routes.toProInstance(host))
    }
  }, [host, navigate, workspace])

  const { store: storeTroubleshoot } = useStoreTroubleshoot()

  const handleTroubleshootClicked = useCallback(() => {
    if (workspace.data && workspaceActions) {
      storeTroubleshoot({
        workspace: workspace.data,
        workspaceActions: workspaceActions,
      })
    }
  }, [storeTroubleshoot, workspace.data, workspaceActions])

  if (!instance) {
    if (isLoadingWorkspaces) {
      return (
        <Center w="full" h="60%" flexFlow="column nowrap">
          <Spinner size="xl" thickness="4px" speed="1s" color="gray.600" />
          <Text mt="4">Loading Workspaces...</Text>
        </Center>
      )
    }

    return (
      <VStack align="start" gap="4">
        <BackToWorkspaces />
        <WarningMessageBox
          warning={
            <>
              Instance <b>{params.workspace}</b> not found
            </>
          }
        />
      </VStack>
    )
  }
  const canStop =
    instance.status?.lastWorkspaceStatus != "Busy" &&
    instance.status?.lastWorkspaceStatus != "Stopped"

  const handleOpenClicked = (ideName: string) => {
    workspace.start({ id: instance.id, ideConfig: { name: ideName } })
    navigate(Routes.toProWorkspace(host, instance.id))
  }

  const handleWorkspaceDisplayNameChanged = (newName: string) => {
    updateWorkspaceDisplayName({ newName })
  }

  return (
    <>
      <VStack align="start" width="full" height="full">
        <BackToWorkspaces />
        <VStack align="start" width="full" pl="4" px="4" paddingInlineEnd="0">
          <Box w="full">
            <WorkspaceCardHeader
              instance={instance}
              showSource={false}
              onDisplayNameChange={handleWorkspaceDisplayNameChanged}>
              <WorkspaceCardHeader.Controls
                onOpenClicked={handleOpenClicked}
                onDeleteClicked={openDeleteModal}
                onRebuildClicked={openRebuildModal}
                onResetClicked={openResetModal}
                onStopClicked={!canStop ? openStopModal : workspace.stop}
                onTroubleshootClicked={handleTroubleshootClicked}
              />
            </WorkspaceCardHeader>
          </Box>

          <WorkspaceDetails
            instance={instance}
            template={template}
            cluster={cluster}
            parameters={parameters}
          />
        </VStack>
        <Box height="full">
          <WorkspaceTabs
            host={host}
            workspace={workspace}
            instance={instance}
            template={template}
          />
        </Box>
      </VStack>

      {stopModal}
      {rebuildModal}
      {resetModal}
      {deleteModal}
    </>
  )
}

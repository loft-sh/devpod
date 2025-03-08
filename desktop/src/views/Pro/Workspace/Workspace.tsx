import { WarningMessageBox } from "@/components"
import {
  ProWorkspaceInstance,
  useProContext,
  useProjectClusters,
  useTemplates,
  useWorkspace,
  useWorkspaceActions,
} from "@/contexts"
import { Clock, Folder, Git, Globe, Image, Status } from "@/icons"
import {
  Annotations,
  Source,
  getDisplayName,
  getLastActivity,
  useDeleteWorkspaceModal,
  useRebuildWorkspaceModal,
  useResetWorkspaceModal,
  useStopWorkspaceModal,
} from "@/lib"
import { Routes } from "@/routes"
import {
  Box,
  Center,
  ComponentWithAs,
  HStack,
  IconProps,
  Spinner,
  Text,
  VStack,
  useColorModeValue,
} from "@chakra-ui/react"
import { ManagementV1DevPodWorkspaceTemplate } from "@loft-enterprise/client/gen/models/managementV1DevPodWorkspaceTemplate"
import dayjs from "dayjs"
import { ReactElement, cloneElement, useCallback, useEffect, useMemo } from "react"
import { useNavigate, useParams } from "react-router-dom"
import { BackToWorkspaces } from "../BackToWorkspaces"
import { WorkspaceTabs } from "./Tabs"
import { WorkspaceCardHeader } from "./WorkspaceCardHeader"
import { WorkspaceStatus } from "./WorkspaceStatus"
import { useStoreTroubleshoot } from "@/lib/useStoreTroubleshoot"

export function Workspace() {
  const params = useParams<{ workspace: string }>()
  const { data: templates } = useTemplates()
  const { data: projectClusters } = useProjectClusters()
  const { host, isLoadingWorkspaces } = useProContext()
  const navigate = useNavigate()
  const workspace = useWorkspace<ProWorkspaceInstance>(params.workspace)
  const instance = workspace.data
  const instanceDisplayName = getDisplayName(instance)
  const workspaceActions = useWorkspaceActions(instance?.id)

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
      },
      [workspace]
    )
  )
  const { modal: resetModal, open: openResetModal } = useResetWorkspaceModal(
    instanceDisplayName,
    useCallback(
      (close) => {
        workspace.reset()
        close()
      },
      [workspace]
    )
  )
  const template = useMemo(
    () =>
      templates?.workspace.find(
        (template) => template.metadata?.name === instance?.spec?.templateRef?.name
      ),
    [instance, templates]
  )
  const cluster = useMemo(() => {
    return projectClusters?.clusters.find(
      // @ts-ignore FIXME: after updating types
      (cluster) => cluster.metadata?.name === instance?.spec?.clusterRef?.cluster
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

  const sourceInfo = getSourceInfo(
    Source.fromRaw(instance.metadata?.annotations?.[Annotations.WorkspaceSource])
  )

  const lastActivity = getLastActivity(instance)

  return (
    <>
      <VStack align="start" width="full" height="full">
        <BackToWorkspaces />
        <VStack align="start" width="full" pl="4" px="4" paddingInlineEnd="0">
          <Box w="full">
            <WorkspaceCardHeader instance={instance} showSource={false}>
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

          <HStack mt="4" gap="6" flexWrap="wrap">
            <WorkspaceInfoDetail
              label={
                <WorkspaceStatus
                  status={instance.status}
                  deletionTimestamp={instance.metadata?.deletionTimestamp}
                />
              }
            />
            <WorkspaceInfoDetail
              icon={Status}
              label={
                <HStack whiteSpace="nowrap" wordBreak={"keep-all"}>
                  <Text>ID: {instance.id}</Text>
                </HStack>
              }
            />
            {sourceInfo && <WorkspaceInfoDetail icon={sourceInfo.icon} label={sourceInfo.label} />}
            <WorkspaceInfoDetail icon={Status} label={formatTemplateDetail(instance, template)} />
            <WorkspaceInfoDetail icon={Globe} label={<Text>{getDisplayName(cluster)}</Text>} />
            {lastActivity && (
              <WorkspaceInfoDetail
                icon={Clock}
                label={<Text>{dayjs(lastActivity).from(Date.now())}</Text>}
              />
            )}
          </HStack>
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

type TWorkspaceInfoDetailProps = Readonly<{
  icon?: ComponentWithAs<"svg", IconProps>
  label: ReactElement
}>
function WorkspaceInfoDetail({ icon: Icon, label }: TWorkspaceInfoDetailProps) {
  const color = useColorModeValue("gray.600", "gray.400")
  const l = cloneElement(label, { color })

  return (
    <HStack gap="1" whiteSpace="nowrap" userSelect="text" cursor="text">
      {Icon && <Icon boxSize="5" color="gray.500" />}
      {l}
    </HStack>
  )
}

function getSourceInfo(
  source: Source | undefined
): Readonly<{ icon: ComponentWithAs<"svg", IconProps>; label: ReactElement }> | undefined {
  if (!source) {
    return undefined
  }

  switch (source.type) {
    case "git":
      return {
        icon: Git,
        label: <Text>{source.value}</Text>,
      }
    case "image":
      return {
        icon: Image,
        label: <Text>{source.value}</Text>,
      }
    case "local":
      return {
        icon: Folder,
        label: <Text>{source.value}</Text>,
      }
  }
}

function formatTemplateDetail(
  instance: ProWorkspaceInstance,
  template: ManagementV1DevPodWorkspaceTemplate | undefined
): ReactElement {
  const templateName = instance.spec?.templateRef?.name
  const templateDisplayName = getDisplayName(template, templateName)
  let templateVersion = instance.spec?.templateRef?.version
  if (!templateVersion) {
    templateVersion = "latest"
  }

  return (
    <Text>
      {templateDisplayName}/{templateVersion}
    </Text>
  )
}

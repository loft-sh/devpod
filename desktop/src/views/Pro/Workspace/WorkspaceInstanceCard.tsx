import { ProWorkspaceInstance, useTemplates, useWorkspace, useWorkspaceActions } from "@/contexts"
import { CogOutlined, Status } from "@/icons"
import {
  TParameterWithValue,
  getDisplayName,
  getParametersWithValues,
  useDeleteWorkspaceModal,
  useRebuildWorkspaceModal,
  useResetWorkspaceModal,
  useStopWorkspaceModal,
} from "@/lib"
import { Routes } from "@/routes"
import {
  Card,
  CardBody,
  CardHeader,
  Divider,
  HStack,
  Text,
  ComponentWithAs,
  IconProps,
  VStack,
  useColorModeValue,
} from "@chakra-ui/react"
import { ManagementV1DevPodWorkspaceTemplate } from "@loft-enterprise/client/gen/models/managementV1DevPodWorkspaceTemplate"
import { useCallback, useMemo, ReactElement, ReactNode, cloneElement } from "react"
import { useNavigate } from "react-router"
import { WorkspaceCardHeader } from "./WorkspaceCardHeader"
import { WorkspaceStatus } from "./WorkspaceStatus"
import { useStoreTroubleshoot } from "@/lib/useStoreTroubleshoot"

type TWorkspaceInstanceCardProps = Readonly<{
  host: string
  instanceName: string
}>

export function WorkspaceInstanceCard({ instanceName, host }: TWorkspaceInstanceCardProps) {
  const hoverColor = useColorModeValue("gray.50", "gray.800")
  const { data: templates } = useTemplates()
  const workspace = useWorkspace<ProWorkspaceInstance>(instanceName)
  const instance = workspace.data
  const instanceDisplayName = getDisplayName(instance)
  const workspaceActions = useWorkspaceActions(instance?.id)

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

  const { store: storeTroubleshoot } = useStoreTroubleshoot()

  const { parameters, template } = useMemo<{
    parameters: readonly TParameterWithValue[]
    template: ManagementV1DevPodWorkspaceTemplate | undefined
  }>(() => {
    // find template for workspace
    const currentTemplate = templates?.workspace.find(
      (template) => instance?.spec?.templateRef?.name === template.metadata?.name
    )
    const empty = { parameters: [], template: undefined }
    if (!currentTemplate || !instance) {
      return empty
    }

    const parameters = getParametersWithValues(instance, currentTemplate)
    if (!parameters) {
      return empty
    }

    return { parameters, template: currentTemplate }
  }, [instance, templates])

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

  const templateRef = instance.spec?.templateRef
  const isRunning = instance.status?.lastWorkspaceStatus === "Running" // TODO: Types

  return (
    <>
      <Card
        direction="column"
        width="full"
        variant="outline"
        marginBottom="3"
        paddingLeft="2"
        _hover={{ bgColor: hoverColor, cursor: "pointer" }}
        boxShadow="0px 2px 4px 0px rgba(0, 0, 0, 0.07)"
        onClick={() => navigate(Routes.toProWorkspace(host, instance.id))}>
        <CardHeader overflow="hidden" w="full">
          <WorkspaceCardHeader instance={instance}>
            <WorkspaceCardHeader.Controls
              onOpenClicked={handleOpenClicked}
              onDeleteClicked={openDeleteModal}
              onRebuildClicked={openRebuildModal}
              onResetClicked={openResetModal}
              onStopClicked={isRunning ? openStopModal : workspace.stop}
              onTroubleshootClicked={handleTroubleshootClicked}
            />
          </WorkspaceCardHeader>
        </CardHeader>
        <CardBody pt="0">
          <HStack gap="6" align="start">
            <WorkspaceInfoDetail icon={Status} label={<Text>Status</Text>}>
              <WorkspaceStatus status={instance.status} />
            </WorkspaceInfoDetail>

            <WorkspaceInfoDetail icon={Status} label={<Text>Template</Text>}>
              <Text>
                {getDisplayName(template, templateRef?.name)}/{templateRef?.version || "latest"}
              </Text>
            </WorkspaceInfoDetail>

            {parameters.length > 0 && (
              <>
                <Divider orientation="vertical" mx="2" h="12" borderColor="gray.400" />

                {parameters.map((param) => {
                  let label = param.label
                  if (!label) {
                    label = param.variable
                  }

                  let value = param.value ?? param.defaultValue ?? ""
                  if (param.type === "boolean") {
                    if (value) {
                      value = "true"
                    } else {
                      value = "false"
                    }
                  }

                  return (
                    <WorkspaceInfoDetail
                      key={param.variable}
                      icon={CogOutlined}
                      label={<Text>{label}</Text>}>
                      <Text>{value}</Text>
                    </WorkspaceInfoDetail>
                  )
                })}
              </>
            )}
          </HStack>
        </CardBody>
      </Card>

      {resetModal}
      {rebuildModal}
      {deleteModal}
      {stopModal}
    </>
  )
}

type TWorkspaceInfoDetailProps = Readonly<{
  icon: ComponentWithAs<"svg", IconProps>
  label: ReactElement
  children: ReactNode
}>
function WorkspaceInfoDetail({ icon: Icon, label, children }: TWorkspaceInfoDetailProps) {
  const l = cloneElement(label, { color: "gray.500", fontWeight: "medium", fontSize: "sm" })

  return (
    <VStack align="start" gap="1" color="gray.700">
      <HStack gap="1">
        <Icon boxSize={4} color="gray.500" />
        {l}
      </HStack>
      {children}
    </VStack>
  )
}

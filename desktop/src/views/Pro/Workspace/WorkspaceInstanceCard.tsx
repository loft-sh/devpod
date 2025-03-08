import { ProWorkspaceInstance, useTemplates, useWorkspace, useWorkspaceActions } from "@/contexts"
import { Clock, CogOutlined, Status, User } from "@/icons"
import {
  TParameterWithValue,
  getDisplayName,
  getLastActivity,
  getParametersWithValues,
  useDeleteWorkspaceModal,
  useRebuildWorkspaceModal,
  useResetWorkspaceModal,
  useStopWorkspaceModal,
} from "@/lib"
import { useStoreTroubleshoot } from "@/lib/useStoreTroubleshoot"
import { Routes } from "@/routes"
import {
  Card,
  CardBody,
  CardHeader,
  ComponentWithAs,
  HStack,
  IconProps,
  Text,
  Tooltip,
  VStack,
  useColorModeValue,
} from "@chakra-ui/react"
import { ManagementV1DevPodWorkspaceTemplate } from "@loft-enterprise/client/gen/models/managementV1DevPodWorkspaceTemplate"
import dayjs from "dayjs"
import { ReactElement, ReactNode, cloneElement, useCallback, useMemo } from "react"
import { useNavigate } from "react-router"
import { WorkspaceCardHeader } from "./WorkspaceCardHeader"
import { WorkspaceStatus } from "./WorkspaceStatus"

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
      return { parameters: [], template: currentTemplate }
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

  // Format timespan labels.
  const [lastActivity, created] = useMemo(() => {
    if (!instance) {
      return [undefined, undefined]
    }

    const lastActivityDate = getLastActivity(instance)
    const lastActivityFormatted = lastActivityDate
      ? dayjs(lastActivityDate).from(Date.now())
      : undefined

    const createdFormatted = instance.metadata?.creationTimestamp
      ? dayjs(instance.metadata.creationTimestamp).from(Date.now())
      : undefined

    return [
      lastActivityFormatted
        ? { formatted: lastActivityFormatted, date: lastActivityDate }
        : undefined,
      createdFormatted
        ? { formatted: createdFormatted, date: instance.metadata?.creationTimestamp }
        : undefined,
    ]
  }, [instance])

  const owner = useMemo(() => {
    return instance?.spec?.owner?.user ?? instance?.spec?.owner?.team ?? "unknown"
  }, [instance])

  if (!instance) {
    return null
  }

  const handleOpenClicked = (ideName: string) => {
    workspace.start({ id: instance.id, ideConfig: { name: ideName } })
    navigate(Routes.toProWorkspace(host, instance.id))
  }

  const templateRef = instance.spec?.templateRef
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
        _hover={{ bgColor: hoverColor, cursor: "pointer" }}
        boxShadow="0px 2px 4px 0px rgba(0, 0, 0, 0.07)"
        onClick={() => navigate(Routes.toProWorkspace(host, instance.id))}>
        <CardHeader overflow="hidden" w="full">
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
          <VStack gap="6" align="start">
            <HStack gap="6">
              <WorkspaceInfoDetail icon={Status} label={<Text>Status</Text>}>
                <WorkspaceStatus
                  status={instance.status}
                  deletionTimestamp={instance.metadata?.deletionTimestamp}
                />
              </WorkspaceInfoDetail>

              <WorkspaceInfoDetail icon={Status} label={<Text>Template</Text>}>
                <Text>
                  {getDisplayName(template, templateRef?.name)}/{templateRef?.version || "latest"}
                </Text>
              </WorkspaceInfoDetail>

              <WorkspaceInfoDetail icon={User} label={<Text>Owner</Text>}>
                <Text>{owner}</Text>
              </WorkspaceInfoDetail>

              {lastActivity && (
                <WorkspaceInfoDetail icon={Clock} label={<Text>Last activity</Text>}>
                  <Tooltip
                    label={lastActivity.date ? lastActivity.date.toLocaleString() : undefined}>
                    <Text>{lastActivity.formatted}</Text>
                  </Tooltip>
                </WorkspaceInfoDetail>
              )}

              {created && (
                <WorkspaceInfoDetail icon={Clock} label={<Text>Created</Text>}>
                  <Tooltip
                    label={created.date ? new Date(created.date).toLocaleString() : undefined}>
                    <Text>{created.formatted}</Text>
                  </Tooltip>
                </WorkspaceInfoDetail>
              )}
            </HStack>

            <HStack gap="6" wrap="wrap">
              {parameters.length > 0 &&
                parameters.map((param) => {
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
            </HStack>
          </VStack>
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
  const color = useColorModeValue("gray.700", "gray.300")
  const labelColor = useColorModeValue("gray.500", "gray.400")
  const l = cloneElement(label, { color: labelColor, fontWeight: "medium", fontSize: "sm" })

  return (
    <VStack align="start" gap="1" color={color}>
      <HStack gap="1">
        <Icon boxSize={4} color={labelColor} />
        {l}
      </HStack>
      {children}
    </VStack>
  )
}

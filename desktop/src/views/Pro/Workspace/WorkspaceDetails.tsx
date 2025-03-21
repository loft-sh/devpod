import { ProWorkspaceInstance } from "@/contexts"
import {
  CPU,
  Clock,
  CogOutlined,
  Dashboard,
  Database,
  Folder,
  Git,
  Globe,
  History as HistoryIcon,
  Image,
  Memory,
  Status,
  User,
} from "@/icons"
import { Annotations, Source, TParameterWithValue, getDisplayName, getLastActivity } from "@/lib"
import { ManagementV1Runner } from "@/runner"
import {
  ComponentWithAs,
  HStack,
  IconProps,
  Text,
  Tooltip,
  VStack,
  useColorModeValue,
} from "@chakra-ui/react"
import { ManagementV1Cluster } from "@loft-enterprise/client/gen/models/managementV1Cluster"
import { ManagementV1DevPodWorkspaceTemplate } from "@loft-enterprise/client/gen/models/managementV1DevPodWorkspaceTemplate"
import dayjs from "dayjs"
import { ReactElement, ReactNode, cloneElement, useMemo } from "react"
import { WorkspaceStatus } from "./WorkspaceStatus"
import { ManagementV1DevPodWorkspaceInstanceKubernetesStatus } from "@loft-enterprise/client/gen/models/managementV1DevPodWorkspaceInstanceKubernetesStatus"
import { ManagementV1DevPodWorkspaceInstancePodStatus, ManagementV1DevPodWorkspaceInstancePodStatusPhaseEnum } from "@loft-enterprise/client/gen/models/managementV1DevPodWorkspaceInstancePodStatus"
import { ManagementV1DevPodWorkspaceInstancePersistentVolumeClaimStatus, ManagementV1DevPodWorkspaceInstancePersistentVolumeClaimStatusPhaseEnum } from "@loft-enterprise/client/gen/models/managementV1DevPodWorkspaceInstancePersistentVolumeClaimStatus"
import { quantityToScalar } from "@kubernetes/client-node/dist/util"

type TWorkspaceDetailsProps = Readonly<{
  instance: ProWorkspaceInstance
  template: ManagementV1DevPodWorkspaceTemplate | undefined
  cluster: ManagementV1Cluster | ManagementV1Runner | undefined
  parameters: readonly TParameterWithValue[]
  showDetails?: boolean
}>
export function WorkspaceDetails({
  instance,
  cluster,
  template,
  parameters,
  showDetails = true,
}: TWorkspaceDetailsProps) {
  const sourceInfo = getSourceInfo(
    Source.fromRaw(instance.metadata?.annotations?.[Annotations.WorkspaceSource])
  )
  const owner = useMemo(() => {
    return instance?.spec?.owner?.user ?? instance?.spec?.owner?.team ?? "unknown"
  }, [instance])

  const mainContainerImage = useMemo(
    () =>
      instance?.status?.kubernetes?.podStatus?.containerStatuses?.find(
        ({ name }) => name === "devpod"
      )?.image,
    [instance.status?.kubernetes]
  )

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

  return (
    <VStack align="start" w="full" gap="4" pb="4">
      <HStack w="full" flexWrap="wrap">
        {showDetails && (
          <>
            {sourceInfo && (
              <WorkspaceInfoDetail
                icon={sourceInfo.icon}
                label={sourceInfo.label}
                tooltipLabel={"Source"}
              />
            )}

            <WorkspaceInfoDetail
              icon={Status}
              label={
                <HStack whiteSpace="nowrap" wordBreak={"keep-all"}>
                  <Text>{instance.id}</Text>
                </HStack>
              }
              tooltipLabel={"ID"}
            />
          </>
        )}
        <WorkspaceInfoDetail
          icon={Status}
          label={formatTemplateDetail(instance, template)}
          tooltipLabel={"Template"}
        />
        <WorkspaceInfoDetail
          icon={Globe}
          label={<Text>{getDisplayName(cluster)}</Text>}
          tooltipLabel={"Cluster"}
        />
        <WorkspaceInfoDetail icon={User} label={<Text>{owner}</Text>} tooltipLabel={"Owner"} />

        {lastActivity && (
          <WorkspaceInfoDetail
            icon={Clock}
            label={<Text>{lastActivity.formatted}</Text>}
            tooltipLabel={
              "Last activity: " + (lastActivity.date ? lastActivity.date.toLocaleString() : "")
            }
          />
        )}

        {created && (
          <WorkspaceInfoDetail
            icon={HistoryIcon}
            label={<Text>{created.formatted}</Text>}
            tooltipLabel={
              "Created: " + (created.date ? new Date(created.date).toLocaleString() : "")
            }
          />
        )}
        {mainContainerImage && (
          <WorkspaceInfoDetail
            icon={Image}
            label={
              <Text maxW="56" overflow="hidden" textOverflow={"ellipsis"}>
                {formatContainerImage(mainContainerImage)}
              </Text>
            }
            tooltipLabel={`Image: ${mainContainerImage}`}
          />
        )}
      </HStack>

      <HStack w="full" flexWrap={"wrap"}>
        <HStack gap="6" mr="8">
          <StackedWorkspaceInfoDetail icon={Status} label={<Text>Status</Text>}>
            <WorkspaceStatus
              status={instance.status}
              deletionTimestamp={instance.metadata?.deletionTimestamp}
            />
          </StackedWorkspaceInfoDetail>

          <StackedWorkspaceInfoDetail icon={Dashboard} label={<Text>Avg. Latency</Text>}>
            <Text>
              {instance.status?.metrics?.latencyMs
                ? instance.status.metrics.latencyMs?.toFixed(2) + "ms"
                : "-"}
            </Text>
          </StackedWorkspaceInfoDetail>

          {instance.status?.kubernetes && <KubernetesDetails status={instance.status.kubernetes} />}
        </HStack>

        {parameters.length > 0 && (
          <HStack
            py="2"
            px="4"
            gap="6"
            wrap="wrap"
            bg="gray.50"
            borderRadius="md"
            borderWidth={"thin"}
            borderColor={"gray.200"}
            _dark={{ bg: "whiteAlpha.50", borderColor: "gray.700" }}>
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
                <ParameterDetail
                  key={param.variable}
                  icon={CogOutlined}
                  label={<Text>{label}</Text>}>
                  <Text>{value}</Text>
                </ParameterDetail>
              )
            })}
          </HStack>
        )}
      </HStack>
    </VStack>
  )
}

type TWorkspaceInfoDetailProps = Readonly<{
  icon?: ComponentWithAs<"svg", IconProps>
  label: ReactElement
  tooltipLabel?: ReactNode
}>
function WorkspaceInfoDetail({ icon: Icon, label, tooltipLabel }: TWorkspaceInfoDetailProps) {
  const color = useColorModeValue("gray.600", "gray.200")
  const l = cloneElement(label, { color })

  const content = (
    <HStack gap="1" whiteSpace="nowrap" userSelect="text" cursor="text">
      {Icon && <Icon boxSize="5" color="gray.500" />}
      {l}
    </HStack>
  )
  if (tooltipLabel) {
    return <Tooltip label={tooltipLabel}>{content}</Tooltip>
  }

  return content
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

type TStackedWorkspaceInfoDetailProps = Readonly<{
  icon: ComponentWithAs<"svg", IconProps>
  label: ReactElement
  children: ReactNode
  tooltipLabel?: ReactNode
}>
function StackedWorkspaceInfoDetail({
  icon: Icon,
  label,
  children,
  tooltipLabel,
}: TStackedWorkspaceInfoDetailProps) {
  const color = useColorModeValue("gray.700", "gray.200")
  const labelColor = useColorModeValue("gray.500", "gray.300")
  const l = cloneElement(label, { color: labelColor, fontWeight: "medium", fontSize: "sm" })

  const content = (
    <VStack align="start" gap="1" color={color}>
      <HStack gap="1">
        <Icon boxSize={4} color={labelColor} />
        {l}
      </HStack>
      {children}
    </VStack>
  )
  if (tooltipLabel) {
    return <Tooltip label={tooltipLabel}>{content}</Tooltip>
  }

  return content
}

type TParameterDetailProps = Readonly<{
  icon: ComponentWithAs<"svg", IconProps>
  label: ReactElement
  children: ReactNode
}>
function ParameterDetail({ icon: Icon, label, children }: TParameterDetailProps) {
  const color = useColorModeValue("gray.800", "gray.200")
  const labelColor = useColorModeValue("gray.600", "gray.300")
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

type TKubernetesDetailsProps = Readonly<{
  status: ManagementV1DevPodWorkspaceInstanceKubernetesStatus
}>
function KubernetesDetails({ status }: TKubernetesDetailsProps) {
  const storageCapacity = status.persistentVolumeClaimStatus?.capacity?.["storage"]
  const resources = useMemo(() => {
    const mainContainerResources = status.podStatus?.containerResources?.find(
      ({ name }) => name === "devpod"
    )
    if (!mainContainerResources) {
      return []
    }

    const mainContainerMetrics = status.podStatus?.containerMetrics?.find(
      ({ name }) => name === "devpod"
    )
    const indexedMetrics: Record<string, string> = {}
    if (mainContainerMetrics?.usage) {
      for (const [k, v] of Object.entries(mainContainerMetrics.usage)) {
        indexedMetrics[k] = v
      }
    }

    if (!mainContainerResources.resources?.limits) {
      return Object.entries(mainContainerMetrics?.usage ?? {}).map(([type, quantity]) => {
        return getResourceDetails(type, undefined, quantity, undefined)
      })
    }

    return Object.entries(mainContainerResources.resources?.limits ?? {}).map(
      ([type, quantity]) => {
        const used = indexedMetrics[type]
        let usagePercentage = calculateUsagePercentage(
          quantityToScalarBigInt(used),
          quantityToScalarBigInt(quantity)
        )

        return getResourceDetails(type, quantity, used, usagePercentage)
      }
    )
  }, [status.podStatus])

  return (
    <>
      {storageCapacity && (
        <StackedWorkspaceInfoDetail icon={Dashboard} label={<Text>Disk Size</Text>}>
          <Text>{storageCapacity}</Text>
        </StackedWorkspaceInfoDetail>
      )}

      {status.podStatus && <PodStatus podStatus={status.podStatus} />}
      {status.persistentVolumeClaimStatus && <PvcStatus pvcStatus={status.persistentVolumeClaimStatus} />}

      {resources.map((resource) => {
        return (
          <StackedWorkspaceInfoDetail
            key={resource.type}
            icon={resource.icon ?? Database}
            label={<Text textTransform={"capitalize"}>{resource.label ?? resource.type}</Text>}
            tooltipLabel={
              <Text>
                {resource.type}: {resource.used ? resource.used : "-"}/
                {resource.total ? resource.total : "-"}
                {resource.unit ? resource.unit : ""}
              </Text>
            }>
            <Text>
              {resource.usagePercentage != null
                ? resource.usagePercentage != invalidQuantity
                  ? resource.usagePercentage + "%"
                  : "-"
                : resource.used != null
                ? resource.used + (resource.unit ?? "")
                : "-"}
            </Text>
          </StackedWorkspaceInfoDetail>
        )
      })}
    </>
  )
}

function PodStatus({ podStatus }: { podStatus: ManagementV1DevPodWorkspaceInstancePodStatus }) {
  const phase = podStatus.phase
  const phaseColor = {
    [ManagementV1DevPodWorkspaceInstancePodStatusPhaseEnum.Pending]: "yellow.500",
    [ManagementV1DevPodWorkspaceInstancePodStatusPhaseEnum.Running]: "",
    [ManagementV1DevPodWorkspaceInstancePodStatusPhaseEnum.Succeeded]: "red.400",
    [ManagementV1DevPodWorkspaceInstancePodStatusPhaseEnum.Failed]: "red.400",
    [ManagementV1DevPodWorkspaceInstancePodStatusPhaseEnum.Unknown]: "red.400",
  }

  let reason = podStatus.reason
  let message = podStatus.message
  if (phase !== ManagementV1DevPodWorkspaceInstancePodStatusPhaseEnum.Running) {
    // check container status first
    const containerStatus = podStatus.containerStatuses?.find((container) => container.name === "devpod" && (container.state?.waiting?.reason || container.state?.terminated?.reason))
    if (containerStatus) {
      if (containerStatus.state?.waiting) {
        reason = containerStatus.state.waiting.reason
        message = containerStatus.state.waiting.message
      } else if (containerStatus.state?.terminated) {
        reason = containerStatus.state.terminated.reason
        message = containerStatus.state.terminated.message
        if (!containerStatus.state.terminated.message && containerStatus.state.terminated.exitCode != 0) {
          message = "Exit code: " + containerStatus.state.terminated.exitCode
        }
      }
    }

    // check pod conditions
    if (!reason && !message) {
      const podCondition = podStatus.conditions?.find((condition) => condition.status === "False" && condition.reason)
      if (podCondition) {
        reason = podCondition.reason
        message = podCondition.message
      }
    }

    // try to find warning event
    if (!reason && !message) {
      const warningEvent = podStatus.events?.find((event) => event.type === "Warning")
      if (warningEvent) {
        reason = warningEvent.reason
        message = warningEvent.message
      }
    }

    // try to find normal event
    if (!reason && !message) {
      const normalEvent = podStatus.events?.find((event) => event.type === "Normal")
      if (normalEvent) {
            reason = normalEvent.reason
            message = normalEvent.message
      }
    }
  }

  return (
    <StackedWorkspaceInfoDetail icon={Dashboard} label={<Text>Pod</Text>}>
      <Text color={phase ? phaseColor[phase] : "gray.500"}>
        {phase === ManagementV1DevPodWorkspaceInstancePodStatusPhaseEnum.Running ? podStatus.phase : (
          (reason && message) ? <Tooltip label={message}>
            <Text>{podStatus.phase} ({reason})</Text>
          </Tooltip> : (reason ? <Text>{podStatus.phase} ({reason})</Text> : podStatus.phase)
        )}
      </Text>
    </StackedWorkspaceInfoDetail>
  )
}


function PvcStatus({ pvcStatus }: { pvcStatus: ManagementV1DevPodWorkspaceInstancePersistentVolumeClaimStatus }) {
  const phase = pvcStatus.phase
  const phaseColor = {
    [ManagementV1DevPodWorkspaceInstancePersistentVolumeClaimStatusPhaseEnum.Pending]: "yellow.500",
    [ManagementV1DevPodWorkspaceInstancePersistentVolumeClaimStatusPhaseEnum.Bound]: "",
    [ManagementV1DevPodWorkspaceInstancePersistentVolumeClaimStatusPhaseEnum.Lost]: "red.400",
  } 

  let reason: string | undefined = ""
  let message: string | undefined = ""
  if (phase !== ManagementV1DevPodWorkspaceInstancePersistentVolumeClaimStatusPhaseEnum.Bound) {
    reason = pvcStatus.conditions?.find((condition) => condition.status === "False")?.reason
    message = pvcStatus.conditions?.find((condition) => condition.status === "False")?.message

    // try to find warning event
    if (!reason && !message) {
      const warningEvent = pvcStatus.events?.find((event) => event.type === "Warning")
      if (warningEvent) {
        reason = warningEvent.reason
        message = warningEvent.message
      }
    }

    // try to find normal event
    if (!reason && !message) {
      const normalEvent = pvcStatus.events?.find((event) => event.type === "Normal")
      if (normalEvent) {
        reason = normalEvent.reason
        message = normalEvent.message
      }
    }
  }

  return (
    <StackedWorkspaceInfoDetail icon={Dashboard} label={<Text>Volume</Text>}>
      <Text color={phase ? phaseColor[phase] : "gray.500"}>
        {phase === ManagementV1DevPodWorkspaceInstancePersistentVolumeClaimStatusPhaseEnum.Bound ? pvcStatus.phase : ((reason && message) ? <Tooltip label={message}>
          <Text>{pvcStatus.phase} ({reason})</Text>
        </Tooltip> : reason ? <Text>{pvcStatus.phase} ({reason})</Text> : pvcStatus.phase) }
      </Text>
    </StackedWorkspaceInfoDetail>
  )
}

const invalidQuantity = -1

function quantityToScalarBigInt(quantity: string | number | undefined): bigint | number {
  if (!quantity) {
    return invalidQuantity
  }

  try {
    return quantityToScalar(String(quantity))
  } catch {
    return invalidQuantity
  }
}

function calculateUsagePercentage(current: number | bigint, total: number | bigint) {
  if (current === invalidQuantity || total === invalidQuantity) {
    return invalidQuantity
  }

  try {
    if (typeof current === "bigint" || typeof total === "bigint") {
      return Number((BigInt(current) * 100n) / BigInt(total))
    }

    return Math.floor((current / total) * 100)
  } catch {
    return invalidQuantity
  }
}

function formatCPU(input: string | undefined): string | undefined {
  if (!input) {
    return undefined
  }
  const nanoMatch = input.match(/^([0-9]+)n$/)?.[1]
  const f = (num: number): string => {
    if (Number.isInteger(num)) {
      return num.toString()
    } else {
      return num.toFixed(2)
    }
  }
  if (nanoMatch !== undefined) {
    return f(parseInt(nanoMatch) / 1000 / 1000 / 1000)
  }

  const uMatch = input.match(/^([0-9]+)u$/)?.[1]
  if (uMatch !== undefined) {
    return f(parseInt(uMatch) / 1000 / 1000)
  }

  const milliMatch = input.match(/^([0-9]+)m$/)?.[1]
  if (milliMatch !== undefined) {
    return f(parseInt(milliMatch) / 1000)
  }

  return f(parseFloat(input))
}

const memoryMultipliers = {
  k: 1000,
  M: 1000 ** 2,
  G: 1000 ** 3,
  T: 1000 ** 4,
  P: 1000 ** 5,
  E: 1000 ** 6,
  Ki: 1024,
  Mi: 1024 ** 2,
  Gi: 1024 ** 3,
  Ti: 1024 ** 4,
  Pi: 1024 ** 5,
  Ei: 1024 ** 6,
} as const

function formatMemoryGi(input: string | undefined): string | undefined {
  if (!input) {
    return undefined
  }
  const unitMatch = input.match(/^([0-9]+)([A-Za-z]{1,2})$/)
  if (unitMatch) {
    const num =
      (parseInt(unitMatch[1]!, 10) *
        memoryMultipliers[unitMatch[2]! as keyof typeof memoryMultipliers]) /
      1024 /
      1024 /
      1024

    if (Number.isInteger(num)) {
      return num.toString()
    } else {
      return num.toFixed(2)
    }
  }

  return parseInt(input, 10).toFixed(2)
}

function formatContainerImage(fullImageName: string): string {
  try {
    const url = new URL("https://" + fullImageName)
    const path = url.pathname
    if (path.startsWith("/")) {
      return path.substring(1, path.length)
    }
    return path
  } catch {
    return fullImageName
  }
}

function getResourceDetails(
  type: string,
  total: string | undefined,
  used: string | undefined,
  usagePercentage: number | undefined
) {
  if (type === "memory") {
    return {
      type,
      usagePercentage,
      icon: Memory,
      used: formatMemoryGi(used),
      total: formatMemoryGi(total),
      unit: "Gi",
    }
  }

  if (type === "cpu") {
    return {
      type,
      usagePercentage,
      icon: CPU,
      label: "CPU",
      used: formatCPU(used),
      total: formatCPU(total),
    }
  }
  if (type.endsWith("/gpu")) {
    return {
      type,
      used,
      icon: CPU,
      label: "GPU",
      total: total,
      usagePercentage: 100, // need to adjust if fractional GPUs and GPU monitoring are supported by the k8s metrics server
    }
  }

  return {
    type,
    used,
    total: total,
    usagePercentage,
  }
}

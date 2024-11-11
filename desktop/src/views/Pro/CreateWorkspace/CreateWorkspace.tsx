import { client as globalClient } from "@/client"
import {
  ProWorkspaceInstance,
  ProWorkspaceStore,
  useProContext,
  useWorkspace,
  useWorkspaceStore,
} from "@/contexts"
import {
  Annotations,
  Failed,
  Labels,
  Result,
  Return,
  Source,
  randomString,
  safeMaxName,
} from "@/lib"
import { Routes } from "@/routes"
import { Box, HStack, Heading, VStack } from "@chakra-ui/react"
import { NewResource, Resources, getProjectNamespace } from "@loft-enterprise/client"
import { ManagementV1DevPodWorkspaceInstance } from "@loft-enterprise/client/gen/models/managementV1DevPodWorkspaceInstance"
import * as jsyaml from "js-yaml"
import { useState } from "react"
import { useNavigate } from "react-router-dom"
import { BackToWorkspaces } from "../BackToWorkspaces"
import { CreateWorkspaceForm } from "./CreateWorkspaceForm"
import { TFormValues } from "./types"

export function CreateWorkspace() {
  const workspace = useWorkspace<ProWorkspaceInstance>(undefined)
  const { store } = useWorkspaceStore<ProWorkspaceStore>()
  const [globalError, setGlobalError] = useState<Failed | null>(null)
  const { host, currentProject, managementSelfQuery, client } = useProContext()
  const navigate = useNavigate()

  const handleReset = () => {
    setGlobalError(null)
    navigate(Routes.toProInstance(host))
  }

  const handleSubmit = async (values: TFormValues) => {
    setGlobalError(null)
    const instanceRes = await buildWorkspaceInstance(
      values,
      currentProject?.metadata?.name,
      managementSelfQuery.data?.status?.projectNamespacePrefix
    )
    if (instanceRes.err) {
      setGlobalError(instanceRes.val)

      return
    }

    const createRes = await client.createWorkspace(instanceRes.val.instance)
    if (createRes.err) {
      setGlobalError(createRes.val)

      return
    }
    // update workspace store immediately
    const instance = new ProWorkspaceInstance(createRes.val)
    store.setWorkspace(instance.id, instance)

    workspace.create({
      id: instanceRes.val.workspaceID,
      workspaceKey: instance.id,
      ideConfig: {
        name: values.defaultIDE,
      },
    })

    navigate(Routes.toProWorkspace(host, instance.id))
  }

  return (
    <Box h="full" mb="40">
      <VStack align="start">
        <BackToWorkspaces />
        <HStack align="center" justify="space-between" mb="8">
          <Heading fontWeight="thin">Create Workspace</Heading>
        </HStack>
      </VStack>
      <CreateWorkspaceForm onReset={handleReset} onSubmit={handleSubmit} error={globalError} />
    </Box>
  )
}

async function buildWorkspaceInstance(
  values: TFormValues,
  currentProject: string | undefined,
  projectNamespacePrefix: string | undefined
): Promise<Result<{ workspaceID: string; instance: ManagementV1DevPodWorkspaceInstance }>> {
  const instance = NewResource(Resources.ManagementV1DevPodWorkspaceInstance)
  const workspaceSource = new Source(values.sourceType, values.source)

  // Workspace name
  const sourceIDRes = await globalClient.workspaces.newID(workspaceSource.stringify())
  if (sourceIDRes.err) {
    return sourceIDRes
  }
  const id = getID(sourceIDRes.val)

  // Kubernetes name
  const kubeNameRes = await getKubeName(values.name || id)
  if (kubeNameRes.err) {
    return kubeNameRes
  }
  const kubeName = kubeNameRes.val

  // ID/UID
  const uidRes = await globalClient.workspaces.newUID()
  if (uidRes.err) {
    return uidRes
  }
  const uid = uidRes.val
  const displayName = values.name
  const ns = getProjectNamespace(currentProject, projectNamespacePrefix)

  if (!instance.metadata) {
    instance.metadata = {}
  }
  if (!instance.metadata.labels) {
    instance.metadata.labels = {}
  }
  if (!instance.metadata.annotations) {
    instance.metadata.annotations = {}
  }
  if (!instance.spec) {
    instance.spec = {}
  }
  instance.metadata.generateName = `${kubeName}-`
  instance.metadata.namespace = ns
  instance.metadata.labels[Labels.WorkspaceID] = id
  instance.metadata.labels[Labels.WorkspaceUID] = uid
  instance.metadata.annotations[Annotations.WorkspaceSource] = workspaceSource.stringify()
  instance.spec.displayName = displayName

  // Template, version and parameters
  const { workspaceTemplate: template, workspaceTemplateVersion, ...parameters } = values.options
  let templateVersion = workspaceTemplateVersion
  if (templateVersion === "latest") {
    templateVersion = ""
  }
  instance.spec.templateRef = {
    name: template,
    version: templateVersion,
  }
  try {
    instance.spec.parameters = jsyaml.dump(parameters)
  } catch (err) {
    return Return.Failed(err as any)
  }

  // Environment template
  if (values.devcontainerType === "external") {
    instance.spec.environmentRef = {
      name: values.devcontainerJSON,
    }
  }

  return Return.Value({ workspaceID: id, instance })
}

async function getKubeName(name: string): Promise<Result<string>> {
  try {
    const kubeName = await safeMaxName(
      name
        .toLowerCase()
        .replace(/[^a-z0-9]/g, "-")
        .replace(/--+/g, "-")
        .replace(/(^[^a-z0-9])|([^a-z0-9]$)/, ""),
      39
    )

    return Return.Value(kubeName)
  } catch (err) {
    return Return.Failed(`Failed to get kubernetes name from ${name}: ${err}`)
  }
}

function getID(name: string): string {
  if (name.length <= 48 - 6) {
    return `${name}-${randomString(5)}`
  }
  const start = name.substring(0, 48 - 6)

  return `${start}-${randomString(5)}`
}

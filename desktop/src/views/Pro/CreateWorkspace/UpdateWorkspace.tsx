import {
  ProWorkspaceInstance,
  ProWorkspaceStore,
  useProContext,
  useWorkspace,
  useWorkspaceStore,
} from "@/contexts"
import { Failed, Result, Return } from "@/lib"
import { Routes } from "@/routes"
import { ManagementV1DevPodWorkspaceTemplate } from "@loft-enterprise/client/gen/models/managementV1DevPodWorkspaceTemplate"
import jsyaml from "js-yaml"
import { useState } from "react"
import { useNavigate } from "react-router"
import { CreateWorkspaceForm } from "./CreateWorkspaceForm"
import { TFormValues } from "./types"
import { Box } from "@chakra-ui/react"

type TUpdateWorkspaceProps = Readonly<{
  instance: ProWorkspaceInstance
  template: ManagementV1DevPodWorkspaceTemplate | undefined
}>
export function UpdateWorkspace({ instance, template }: TUpdateWorkspaceProps) {
  const navigate = useNavigate()
  const workspace = useWorkspace<ProWorkspaceInstance>(instance.id)
  const { store } = useWorkspaceStore<ProWorkspaceStore>()
  const { host, client } = useProContext()
  const [globalError, setGlobalError] = useState<Failed | null>(null)

  const handleSubmit = async (values: TFormValues) => {
    setGlobalError(null)

    const res = updateWorkspaceInstance(instance, values)
    if (res.err) {
      setGlobalError(res.val)

      return
    }

    const updateRes = await client.updateWorkspace(res.val)
    if (updateRes.err) {
      setGlobalError(updateRes.val)

      return
    }
    // update workspace store immediately
    const updatedInstance = new ProWorkspaceInstance(updateRes.val)
    store.setWorkspace(updatedInstance.id, updatedInstance)

    workspace.start({ id: updatedInstance.id, ideConfig: { name: values.defaultIDE } })

    navigate(Routes.toProWorkspaceDetail(host, instance.id, "logs"))
  }

  const handleReset = () => {
    setGlobalError(null)
  }

  return (
    <Box mb="40">
      <CreateWorkspaceForm
        instance={instance}
        template={template}
        onSubmit={handleSubmit}
        onReset={handleReset}
        error={globalError}
      />
    </Box>
  )
}

function updateWorkspaceInstance(
  instance: ProWorkspaceInstance,
  values: TFormValues
): Result<ProWorkspaceInstance> {
  const newInstance = new ProWorkspaceInstance(instance)
  if (!newInstance.spec) {
    newInstance.spec = {}
  }

  // source can't be updated

  // template
  const { workspaceTemplate: template, workspaceTemplateVersion, ...parameters } = values.options
  let templateVersion = workspaceTemplateVersion
  if (templateVersion === "latest") {
    templateVersion = ""
  }
  if (
    newInstance.spec.templateRef?.name !== template ||
    newInstance.spec.templateRef?.version !== workspaceTemplateVersion
  ) {
    newInstance.spec.templateRef = {
      name: template,
      version: templateVersion,
    }
  }

  // parameters
  try {
    const newParameters = jsyaml.dump(parameters)
    if (newInstance.spec.parameters !== newParameters) {
      newInstance.spec.parameters = newParameters
    }
  } catch (err) {
    return Return.Failed(err as any)
  }

  // name
  if (newInstance.spec.displayName !== values.name) {
    newInstance.spec.displayName = values.name
  }

  // devcontainer.json can't be updated

  return Return.Value(newInstance)
}

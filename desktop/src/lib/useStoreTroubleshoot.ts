import { client } from "@/client"
import { TActionObj } from "@/contexts/DevPodContext/action"
import { TWorkspace } from "@/types"
import { useToast } from "@chakra-ui/react"
import { useMutation } from "@tanstack/react-query"
import { ProWorkspaceInstance } from "@/contexts"

export function useStoreTroubleshoot() {
  const toast = useToast()
  const { mutate, isLoading: isStoring } = useMutation({
    mutationFn: async ({
      workspace,
      workspaceActions,
    }: {
      workspace: TWorkspace | ProWorkspaceInstance
      workspaceActions: TActionObj[]
    }) => {
      const logFiles = await Promise.all(
        workspaceActions.map((action) => client.workspaces.getActionLogFile(action.id))
      )

      const targetFolder = await client.selectFromDir("Save Troubleshooting Data")

      // user cancelled "save file" dialog
      if (targetFolder === null) {
        return
      }

      const unwrappedLogFiles = logFiles
        .filter((f) => f.ok)
        .map((f) => f.unwrap() ?? "")
        .map((f) => [[f], [targetFolder, f.split("/").pop() ?? ""]])
      // poor mans zip
      await Promise.all(
        unwrappedLogFiles.map(([src, target]) => client.copyFilePaths(src ?? [], target ?? []))
      )

      await client.writeTextFile(
        [targetFolder, "workspace_actions.json"],
        JSON.stringify(workspaceActions, null, 2)
      )

      await client.writeTextFile(
        [targetFolder, "workspace.json"],
        JSON.stringify(workspace, null, 2)
      )

      const troubleshootOutput = await client.workspaces.troubleshoot({
        id: workspace.id,
        actionID: "",
        streamID: "",
      })
      if (troubleshootOutput.ok) {
        await client.writeTextFile(
          [targetFolder, "cli_troubleshoot.json"],
          troubleshootOutput.unwrap().stdout
        )
      }

      client.open(targetFolder)
    },
    onError(error) {
      toast({
        title: `Failed to save logs: ${error}`,
        status: "error",
        isClosable: true,
        duration: 30_000, // 30 sec
      })
    },
  })

  return { store: mutate, isStoring }
}

import { client } from "@/client"
import { TActionID } from "@/contexts"
import { useToast } from "@chakra-ui/react"
import { useMutation } from "@tanstack/react-query"
import * as dialog from "@tauri-apps/plugin-dialog"

export function useDownloadLogs() {
  const toast = useToast()
  const { mutate, isLoading: isDownloading } = useMutation({
    mutationFn: async ({ actionID }: { actionID: TActionID }) => {
      const actionLogFile = (await client.workspaces.getActionLogFile(actionID)).unwrap()

      if (actionLogFile === undefined) {
        throw new Error(`Unable to retrieve file for action ${actionID}`)
      }

      const targetFile = await dialog.save({
        title: "Save Logs",
        filters: [{ name: "format", extensions: ["log", "txt"] }],
      })

      // user cancelled "save file" dialog
      if (targetFile === null) {
        return
      }

      await client.copyFile(actionLogFile, targetFile)
      client.open(targetFile)
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

  return { download: mutate, isDownloading }
}

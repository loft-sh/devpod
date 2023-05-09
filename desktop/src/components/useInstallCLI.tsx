import { QuestionIcon } from "@chakra-ui/icons"
import { Button, Code, Tooltip } from "@chakra-ui/react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useMemo } from "react"
import { CheckCircle, ExclamationCircle } from "../icons"
import { client } from "../client"
import { QueryKeys } from "../queryKeys"
import { isError, isWindows } from "../lib"
import { ErrorMessageBox } from "./Error"

export function useInstallCLI() {
  const { data: isCLIInstalled } = useQuery<boolean>({
    queryKey: QueryKeys.IS_CLI_INSTALLED,
    queryFn: async () => {
      return (await client.isCLIInstalled()).unwrap()!
    },
  })
  const queryClient = useQueryClient()
  const {
    mutate: addBinaryToPath,
    isLoading,
    error,
    status,
  } = useMutation({
    mutationFn: async () => {
      ;(await client.installCLI()).unwrap()
    },
    onSettled: () => {
      queryClient.invalidateQueries(QueryKeys.IS_CLI_INSTALLED)
    },
  })

  const badge = useMemo(() => {
    if (isCLIInstalled === undefined) {
      return (
        <Tooltip label="No information available">
          <QuestionIcon boxSize={5} color="gray.400" />
        </Tooltip>
      )
    }

    return isCLIInstalled ? (
      <Tooltip label="Installed">
        <CheckCircle boxSize={5} color="green.500" />
      </Tooltip>
    ) : (
      <Tooltip label="Not Installed">
        <ExclamationCircle boxSize={5} color="red.500" />
      </Tooltip>
    )
  }, [isCLIInstalled])

  const button = useMemo(() => {
    return (
      <Button
        variant="outline"
        isLoading={isLoading}
        onClick={() => addBinaryToPath()}
        isDisabled={status === "success"}>
        Add CLI to Path
      </Button>
    )
  }, [addBinaryToPath, isLoading, status])

  const helpText = useMemo(() => {
    return (
      <>
        Adds the DevPod CLI to your <Code>$PATH</Code>.
        <br />
        {isWindows ? (
          <>
            It will be placed in <Code>%APP_DATA%\sh.loft.devpod\bin</Code>
          </>
        ) : (
          <>
            It will be placed in either <Code>/usr/local/bin</Code>,<Code>$HOME/.local/bin</Code> or{" "}
            <Code>$HOME/bin</Code> depending on your permissions
          </>
        )}
      </>
    )
  }, [])

  const errorMessage = useMemo(() => {
    return isError(error) && <ErrorMessageBox error={error} />
  }, [error])

  return {
    isInstalled: isCLIInstalled,
    install: addBinaryToPath,
    isLoading,
    error,
    status,
    badge,
    button,
    helpText,
    errorMessage,
  }
}

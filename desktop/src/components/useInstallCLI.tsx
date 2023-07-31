import { QuestionIcon } from "@chakra-ui/icons"
import {
  AlertDialog,
  AlertDialogBody,
  AlertDialogContent,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogOverlay,
  Button,
  ButtonGroup,
  Code,
  Tooltip,
  useDisclosure,
} from "@chakra-ui/react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useMemo, useRef } from "react"
import { client } from "../client"
import { CheckCircle, ExclamationCircle } from "../icons"
import { Err, Failed, isError, isMacOS, isWindows } from "../lib"
import { QueryKeys } from "../queryKeys"
import { ErrorMessageBox } from "./Error"

export function useInstallCLI() {
  const { isOpen, onOpen: showAlertDialog, onClose } = useDisclosure()
  const cancelRef = useRef<HTMLButtonElement>(null)
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
  } = useMutation<void, Err<Failed>, { force?: boolean }>({
    mutationFn: async ({ force = false }) => {
      ;(await client.installCLI(force)).unwrap()
      // throw Return.Failed("Did not work")
    },
    onSettled: () => {
      queryClient.invalidateQueries(QueryKeys.IS_CLI_INSTALLED)
    },
    onError: (_, { force }) => {
      if (isMacOS && !force) {
        showAlertDialog()
      }
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
      <>
        <Button
          variant="outline"
          isLoading={isLoading}
          onClick={() => addBinaryToPath({})}
          isDisabled={status === "success"}>
          Add CLI to Path
        </Button>
        <AlertDialog isOpen={isOpen} onClose={onClose} leastDestructiveRef={cancelRef}>
          <AlertDialogOverlay>
            <AlertDialogContent>
              <AlertDialogHeader>Failed to add CLI to path</AlertDialogHeader>
              <AlertDialogBody>
                Do you want to retry with Admin Privileges? You will be prompted for authentication
              </AlertDialogBody>
              <AlertDialogFooter>
                <ButtonGroup>
                  <Button variant="ghost" ref={cancelRef} onClick={onClose}>
                    Cancel
                  </Button>
                  <Button
                    variant="solid"
                    onClick={() => {
                      addBinaryToPath({ force: true })
                      onClose()
                    }}>
                    Okay
                  </Button>
                </ButtonGroup>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialogOverlay>
        </AlertDialog>
      </>
    )
  }, [addBinaryToPath, isLoading, isOpen, onClose, status])

  const helpText = useMemo(() => {
    return (
      <>
        Adds the DevPod CLI to your <Code>$PATH</Code>.{" "}
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
    return error !== null && isError(error.val) && <ErrorMessageBox error={error.val} />
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

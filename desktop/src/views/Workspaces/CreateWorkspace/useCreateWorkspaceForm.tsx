import {
  Button,
  Code,
  List,
  ListItem,
  Modal,
  ModalBody,
  ModalCloseButton,
  ModalContent,
  ModalFooter,
  ModalHeader,
  ModalOverlay,
  Text,
  useDisclosure,
} from "@chakra-ui/react"
import { FormEventHandler, useCallback, useEffect, useMemo, useRef, useState } from "react"
import { useForm } from "react-hook-form"
import { client } from "../../../client"
import { useSettings, useWorkspaces } from "../../../contexts"
import { exists } from "../../../lib"
import { randomWords } from "../../../lib/randomWords"
import { TIDEs, TProviders, TWorkspace } from "../../../types"
import { FieldName, TCreateWorkspaceArgs, TCreateWorkspaceSearchParams, TFormValues } from "./types"

const DEFAULT_PREBUILD_REPOSITORY_KEY = "devpod-create-prebuild-repository"
const DEFAULT_CONTAINER_PATH = "__internal-default"

export function useCreateWorkspaceForm(
  params: TCreateWorkspaceSearchParams,
  providers: TProviders | undefined,
  ides: TIDEs | undefined,
  onCreateWorkspace: (args: TCreateWorkspaceArgs) => void
) {
  const formRef = useRef<HTMLFormElement>(null)
  const settings = useSettings()
  const workspaces = useWorkspaces()
  const [isSubmitLoading, setIsSubmitLoading] = useState(false)
  const { register, handleSubmit, formState, watch, setError, setValue, control } =
    useForm<TFormValues>({
      defaultValues: {
        [FieldName.PREBUILD_REPOSITORY]:
          window.localStorage.getItem(DEFAULT_PREBUILD_REPOSITORY_KEY) ?? "",
        [FieldName.PROVIDER]: Object.keys(providers ?? {})[0] ?? undefined,
        [FieldName.DEVCONTAINER_PATH]: undefined,
      },
    })
  const currentSource = watch(FieldName.SOURCE)
  const currentProvider = watch(FieldName.PROVIDER)
  const isSubmitting = useMemo(
    () => formState.isSubmitting || isSubmitLoading,
    [formState.isSubmitting, isSubmitLoading]
  )
  const handleDevcontainerSelected = useCallback((selectedDevcontainer: string | undefined) => {
    setValue(FieldName.DEVCONTAINER_PATH, selectedDevcontainer ?? DEFAULT_CONTAINER_PATH, {
      shouldDirty: true,
      shouldValidate: true,
    })
    formRef.current?.dispatchEvent(new Event("submit", { cancelable: true, bubbles: true }))
  }, [])

  const { modal: selectDevcontainerModal, show: showSelectDevcontainerModal } =
    useSelectDevcontainerModal({ onSelected: handleDevcontainerSelected })

  useEffect(() => {
    const opts = {
      shouldDirty: true,
      shouldValidate: true,
    }
    if (params.workspaceID !== undefined) {
      setValue(FieldName.ID, params.workspaceID, opts)
    }

    if (params.rawSource !== undefined) {
      setValue(FieldName.SOURCE, params.rawSource, opts)
    }

    // default ide
    if (params.ide !== undefined) {
      setValue(FieldName.DEFAULT_IDE, params.ide, opts)
    } else if (ides?.length) {
      const defaultIDE = ides.find((ide) => ide.default)
      if (defaultIDE) {
        setValue(FieldName.DEFAULT_IDE, defaultIDE.name!, opts)
      } else {
        const openvscode = ides.find((ide) => ide.name === "openvscode")
        if (openvscode && openvscode.name) {
          setValue(FieldName.DEFAULT_IDE, openvscode.name, opts)
        }
      }
    }

    // default provider
    if (params.providerID !== undefined) {
      setValue(FieldName.PROVIDER, params.providerID, opts)
    } else if (providers) {
      const defaultProviderID = Object.keys(providers).find(
        (providerID) => providers[providerID]?.default
      )
      if (defaultProviderID) {
        setValue(FieldName.PROVIDER, defaultProviderID, opts)
      }
    }
  }, [ides, params, providers, setValue])

  // Handle workspace name
  useEffect(() => {
    if (exists(currentSource) && currentSource !== "") {
      setValue(FieldName.ID, "", { shouldDirty: true })

      client.workspaces.newID(currentSource).then((res) => {
        if (res.err) {
          setError(FieldName.SOURCE, { message: res.val.message })

          return
        }
        let workspaceID = res.val
        if (!isWorkspaceNameAvailable(workspaceID, workspaces)) {
          workspaceID = `${workspaceID}-${currentProvider}`

          if (isWorkspaceNameAvailable(workspaceID, workspaces)) {
            setValue(FieldName.ID, workspaceID, { shouldDirty: true })

            return
          }

          const words = randomWords({ amount: 2 })
          workspaceID = `${res.val}-${words[0] ?? "x"}-${words[1] ?? "y"}`
          if (isWorkspaceNameAvailable(workspaceID, workspaces)) {
            setValue(FieldName.ID, workspaceID, { shouldDirty: true })

            return
          }

          setError(FieldName.SOURCE, { message: "Workspace with the same name already exists" })

          return
        }
      })
    }
  }, [currentProvider, currentSource, setError, setValue, workspaces])

  const onSubmit = useCallback<FormEventHandler<HTMLFormElement>>(
    (event) =>
      handleSubmit(async (data) => {
        // save prebuild repository
        const maybePrebuildRepo = data[FieldName.PREBUILD_REPOSITORY]
        if (maybePrebuildRepo) {
          window.localStorage.setItem(DEFAULT_PREBUILD_REPOSITORY_KEY, maybePrebuildRepo)
        } else {
          window.localStorage.removeItem(DEFAULT_PREBUILD_REPOSITORY_KEY)
        }

        const workspaceSource = data[FieldName.SOURCE].trim()
        setIsSubmitLoading(true)
        let workspaceID = data[FieldName.ID]
        if (!workspaceID) {
          const newIDResult = await client.workspaces.newID(workspaceSource)
          if (newIDResult.err) {
            setIsSubmitLoading(false)
            setError(FieldName.SOURCE, { message: newIDResult.val.message })

            return
          }

          workspaceID = newIDResult.val
        }

        if (workspaces.find((workspace) => workspace.id === workspaceID)) {
          setIsSubmitLoading(false)
          setError(FieldName.SOURCE, { message: "workspace with the same name already exists" })

          return
        }

        const providerID = data[FieldName.PROVIDER]
        const defaultIDE = data[FieldName.DEFAULT_IDE]

        // set default provider
        const useProviderResult = await client.providers.useProvider(providerID)
        if (useProviderResult.err) {
          setIsSubmitLoading(false)
          setError(FieldName.SOURCE, { message: useProviderResult.val.message })

          return
        }

        let maybeDevcontainerPath = data[FieldName.DEVCONTAINER_PATH]
        if (maybeDevcontainerPath === DEFAULT_CONTAINER_PATH) {
          maybeDevcontainerPath = undefined
        } else if (settings.experimental_multiDevcontainer && maybeDevcontainerPath === "") {
          // check for multiple devcontainers
          const checkDevcontainerSetupResult = await client.workspaces.checkDevcontainerSetup(
            workspaceSource
          )
          setIsSubmitLoading(false)
          if (!checkDevcontainerSetupResult.ok) {
            setError(FieldName.DEVCONTAINER_PATH, {
              message: checkDevcontainerSetupResult.val.message,
            })
          } else if (checkDevcontainerSetupResult.val.configPaths.length > 1) {
            showSelectDevcontainerModal(checkDevcontainerSetupResult.val.configPaths)

            return
          }
        }

        if (!settings.fixedIDE) {
          // set default ide
          const useIDEResult = await client.ides.useIDE(defaultIDE)
          if (useIDEResult.err) {
            setIsSubmitLoading(false)
            setError(FieldName.SOURCE, { message: useIDEResult.val.message })

            return
          }
        }

        setIsSubmitLoading(false)
        const prebuildRepositories = data[FieldName.PREBUILD_REPOSITORY]
          ? [data[FieldName.PREBUILD_REPOSITORY]]
          : []

        onCreateWorkspace({
          workspaceID,
          providerID,
          prebuildRepositories,
          defaultIDE,
          workspaceSource,
          devcontainerPath: maybeDevcontainerPath,
        })
      })(event),
    [handleSubmit, workspaces, settings.fixedIDE, onCreateWorkspace, setError]
  )

  return {
    formRef,
    register,
    setValue,
    isSubmitLoading,
    formState,
    onSubmit,
    isSubmitting,
    currentSource,
    control,
    selectDevcontainerModal,
  }
}

function isWorkspaceNameAvailable(workspaceID: string, workspaces: readonly TWorkspace[]): boolean {
  return workspaces.find((workspace) => workspace.id === workspaceID) === undefined
}

function useSelectDevcontainerModal({
  onSelected,
}: Readonly<{ onSelected: (path: string | undefined) => void }>) {
  const [devcontainerPaths, setDevcontainerPaths] = useState<string[]>([])
  const { isOpen, onClose, onOpen } = useDisclosure()

  const modal = useMemo(
    () => (
      <Modal
        onClose={() => {
          onSelected(undefined)
          onClose()
        }}
        isOpen={isOpen}
        isCentered
        size="3xl"
        scrollBehavior="inside"
        closeOnEsc={true}
        closeOnOverlayClick={true}>
        <ModalOverlay />
        <ModalContent>
          <ModalCloseButton />
          <ModalHeader>Select devcontainer</ModalHeader>
          <ModalBody>
            <Text>
              There are multiple <Code>devcontainer.json</Code> files available. Please select one
              or dismiss to use default.
            </Text>
            <List spacing="2" paddingTop="4">
              {devcontainerPaths.map((devcontainerPath) => (
                <ListItem key={devcontainerPath}>
                  <Button
                    width="full"
                    justifyContent="start"
                    variant="ghost"
                    key={devcontainerPath}
                    onClick={() => {
                      onClose()
                      onSelected(devcontainerPath)
                    }}>
                    {devcontainerPath}
                  </Button>
                </ListItem>
              ))}
            </List>
          </ModalBody>
          <ModalFooter />
        </ModalContent>
      </Modal>
    ),
    [devcontainerPaths, isOpen, onClose, onSelected]
  )

  const show = useCallback(
    (devcontainerPaths: string[]) => {
      setDevcontainerPaths(devcontainerPaths)
      onOpen()
    },
    [onOpen]
  )

  return { show, modal }
}

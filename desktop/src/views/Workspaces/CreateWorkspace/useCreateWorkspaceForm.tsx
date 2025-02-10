import { Routes } from "@/routes"
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
import { createSearchParams } from "react-router-dom"
import { client } from "../../../client"
import { useSettings, useWorkspaces } from "../../../contexts"
import { exists } from "../../../lib"
import { randomWords } from "../../../lib/randomWords"
import { TIDEs, TProviders, TWorkspace } from "../../../types"
import { FieldName, TCreateWorkspaceArgs, TFormValues } from "./types"

const DEFAULT_PREBUILD_REPOSITORY_KEY = "devpod-create-prebuild-repository"
const DEFAULT_CONTAINER_PATH = "__internal-default"

export function useCreateWorkspaceForm(onCreateWorkspace: (args: TCreateWorkspaceArgs) => void) {
  const formRef = useRef<HTMLFormElement>(null)
  const settings = useSettings()
  const workspaces = useWorkspaces<TWorkspace>()
  const [isSubmitLoading, setIsSubmitLoading] = useState(false)
  const { register, handleSubmit, formState, watch, setError, setValue, control, getFieldState } =
    useForm<TFormValues>({
      async defaultValues() {
        const params = Routes.getWorkspaceCreateParamsFromSearchParams(
          createSearchParams(location.search)
        )

        const [providersRes, idesRes] = await Promise.all([
          client.providers.listAll(),
          client.ides.listAll(),
        ])
        let providers: TProviders = {}
        if (providersRes.ok) {
          providers = providersRes.val
        }

        let ides: TIDEs = []
        if (idesRes.ok) {
          ides = idesRes.val
        }

        const defaultProvider =
          params.providerID ??
          Object.keys(providers).find((providerID) => providers[providerID]?.default) ??
          Object.keys(providers)[0] ??
          ""

        const defaultIDE =
          params.ide ??
          ides.find((ide) => ide.default)?.name ??
          ides.find((ide) => ide.name === "openvscode")?.name ??
          ""

        const defaultWorkspaceID = params.workspaceID ?? ""
        const defaultSource = params.rawSource ?? ""
        const defaultPrebuildRepo =
          window.localStorage.getItem(DEFAULT_PREBUILD_REPOSITORY_KEY) ?? ""

        return {
          [FieldName.ID]: defaultWorkspaceID,
          [FieldName.SOURCE]: defaultSource,
          [FieldName.PROVIDER]: defaultProvider,
          [FieldName.DEFAULT_IDE]: defaultIDE,
          [FieldName.DEVCONTAINER_PATH]: undefined,
          [FieldName.PREBUILD_REPOSITORY]: defaultPrebuildRepo,
        }
      },
    })
  const currentSource = watch(FieldName.SOURCE)
  const currentProvider = watch(FieldName.PROVIDER)
  const isSubmitting = useMemo(
    () => formState.isSubmitting || isSubmitLoading,
    [formState.isSubmitting, isSubmitLoading]
  )
  const handleDevcontainerSelected = useCallback(
    (selectedDevcontainer: string | undefined) => {
      setValue(FieldName.DEVCONTAINER_PATH, selectedDevcontainer ?? DEFAULT_CONTAINER_PATH, {
        shouldDirty: true,
        shouldValidate: true,
      })
      formRef.current?.dispatchEvent(new Event("submit", { cancelable: true, bubbles: true }))
    },
    [setValue]
  )

  const { modal: selectDevcontainerModal, show: showSelectDevcontainerModal } =
    useSelectDevcontainerModal({ onSelected: handleDevcontainerSelected })

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
  }, [currentProvider, currentSource, getFieldState, setError, setValue, workspaces])

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
          const checkDevcontainerSetupResult =
            await client.workspaces.checkDevcontainerSetup(workspaceSource)
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
    [
      handleSubmit,
      workspaces,
      settings.experimental_multiDevcontainer,
      settings.fixedIDE,
      onCreateWorkspace,
      setError,
      showSelectDevcontainerModal,
    ]
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

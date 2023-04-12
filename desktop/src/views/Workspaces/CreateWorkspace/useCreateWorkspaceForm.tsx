import { ChangeEvent, FormEventHandler, useCallback, useEffect, useMemo, useState } from "react"
import { useForm } from "react-hook-form"
import { client } from "../../../client"
import { useWorkspaces } from "../../../contexts"
import { TIDEs, TProviders } from "../../../types"
import { FieldName, TCreateWorkspaceArgs, TCreateWorkspaceSearchParams, TFormValues } from "./types"

const DEFAULT_PREBUILD_REPOSITORY_KEY = "devpod-create-prebuild-repository"

export function useCreateWorkspaceForm(
  params: TCreateWorkspaceSearchParams,
  providers: TProviders | undefined,
  ides: TIDEs | undefined,
  onCreateWorkspace: (args: TCreateWorkspaceArgs) => void
) {
  const workspaces = useWorkspaces()
  const [isSubmitLoading, setIsSubmitLoading] = useState(false)
  const { register, handleSubmit, formState, watch, setError, setValue, clearErrors } =
    useForm<TFormValues>({
      defaultValues: {
        [FieldName.PREBUILD_REPOSITORY]:
          window.localStorage.getItem(DEFAULT_PREBUILD_REPOSITORY_KEY) ?? "",
      },
    })
  const currentSource = watch(FieldName.SOURCE)
  const isSubmitting = useMemo(
    () => formState.isSubmitting || isSubmitLoading,
    [formState.isSubmitting, isSubmitLoading]
  )

  useEffect(() => {
    if (params.workspaceID !== undefined) {
      setValue(FieldName.ID, params.workspaceID)
    }

    if (params.rawSource !== undefined) {
      setValue(FieldName.SOURCE, params.rawSource)
    }

    // default ide
    if (params.ide !== undefined) {
      setValue(FieldName.DEFAULT_IDE, params.ide)
    } else if (ides?.length) {
      const defaultIDE = ides.find((ide) => ide.default)
      if (defaultIDE) {
        setValue(FieldName.DEFAULT_IDE, defaultIDE.name!)
      } else {
        const vscode = ides.find((ide) => ide.name === "vscode")
        if (vscode && vscode.name) {
          setValue(FieldName.DEFAULT_IDE, vscode.name)
        }
      }
    }

    // default provider
    if (params.providerID !== undefined) {
      setValue(FieldName.PROVIDER, params.providerID)
    } else if (providers) {
      const defaultProviderID = Object.keys(providers).find(
        (providerID) => providers[providerID]?.default
      )
      if (defaultProviderID) {
        setValue(FieldName.PROVIDER, defaultProviderID)
      }
    }
  }, [ides, params, providers, setValue])

  const onSubmit = useCallback<FormEventHandler<HTMLFormElement>>(
    (event) =>
      handleSubmit(async (data) => {
        // save prebuild repository
        if (data[FieldName.PREBUILD_REPOSITORY]) {
          window.localStorage.setItem(
            DEFAULT_PREBUILD_REPOSITORY_KEY,
            data[FieldName.PREBUILD_REPOSITORY]
          )
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

        // set default ide
        const useIDEResult = await client.ides.useIDE(defaultIDE)
        if (useIDEResult.err) {
          setIsSubmitLoading(false)
          setError(FieldName.SOURCE, { message: useIDEResult.val.message })

          return
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
        })
      })(event),
    [handleSubmit, workspaces, onCreateWorkspace, setError]
  )

  const validateWorkspaceID = useCallback(
    (e: ChangeEvent<HTMLInputElement>) => {
      setValue(FieldName.ID, e.target.value, {
        shouldDirty: true,
      })

      if (/[^a-z0-9-]+/.test(e.target.value)) {
        setError(FieldName.ID, {
          message: "Name can only consist of lower case letters, numbers and dashes",
        })
      } else {
        clearErrors(FieldName.ID)
      }
    },
    [clearErrors, setError, setValue]
  )

  return {
    register,
    setValue,
    isSubmitLoading,
    validateWorkspaceID,
    formState,
    onSubmit,
    isSubmitting,
    currentSource,
  }
}

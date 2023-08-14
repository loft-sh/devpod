import {
  Code,
  FormControl,
  FormErrorMessage,
  FormHelperText,
  FormLabel,
  VStack,
} from "@chakra-ui/react"
import { useCallback } from "react"
import { Controller, useForm } from "react-hook-form"
import { ErrorMessageBox } from "../../../components"
import { useProviders } from "../../../contexts"
import { exists, isError, useFormErrors } from "../../../lib"
import { LoadingProviderIndicator } from "./LoadingProviderIndicator"
import { CustomNameInput } from "./SetupProviderSourceForm"
import { ALLOWED_NAMES_REGEX, mergeProviderOptions } from "./helpers"
import { FieldName, TCloneProviderInfo, TFormValues, TSetupProviderResult } from "./types"
import { useAddProvider } from "./useAddProvider"
import { Form } from "@/components/Form"

type TCloneProviderProps = Readonly<{
  isModal?: boolean
  cloneProviderInfo: TCloneProviderInfo
  onFinish: (result: TSetupProviderResult) => void
  reset: () => void
}>
export function SetupClonedProvider({ cloneProviderInfo, onFinish, reset }: TCloneProviderProps) {
  const [[providers]] = useProviders()
  const { handleSubmit, formState, control, watch } = useForm<TFormValues>({
    defaultValues: {
      [FieldName.PROVIDER_SOURCE]: cloneProviderInfo.sourceProviderSource,
    },
  })
  const newProviderName = watch(FieldName.PROVIDER_NAME)
  const { providerNameError } = useFormErrors([FieldName.PROVIDER_NAME], formState)
  const {
    mutate: addProvider,
    status,
    error,
  } = useAddProvider({
    onSuccess(result) {
      const oldProvider = cloneProviderInfo.sourceProvider

      onFinish({
        providerID: result.providerID,
        optionGroups: result.optionGroups,
        options: mergeProviderOptions(oldProvider.state?.options, result.options),
      })
    },
    onError() {
      reset()
    },
  })
  const onSubmit = useCallback(
    async (values: TFormValues) => {
      addProvider({
        rawProviderSource: values[FieldName.PROVIDER_SOURCE],
        config: { name: values[FieldName.PROVIDER_NAME] },
      })
      // gotta merge the options with the existing state now
    },
    [addProvider]
  )
  const isLoading = status === "loading"

  return (
    <>
      <VStack align="start" spacing={8} width="full" marginBottom={6}>
        <Form onSubmit={handleSubmit(onSubmit)} justifyContent="center">
          <FormControl
            alignSelf="center"
            maxWidth={{ base: "3xl", xl: "4xl" }}
            marginBottom={4}
            isDisabled={isLoading || status === "success"}
            isInvalid={exists(providerNameError)}>
            <FormLabel>Name</FormLabel>
            <Controller
              name={FieldName.PROVIDER_NAME}
              control={control}
              rules={{
                pattern: {
                  value: ALLOWED_NAMES_REGEX,
                  message: "Name can only contain letters, numbers and -",
                },
                validate: {
                  unique: (value) => {
                    if (value === undefined) return true
                    if (value === "") return "Name cannot be empty"

                    return providers?.[value] === undefined ? true : "Name must be unique"
                  },
                },
                maxLength: { value: 48, message: "Name cannot be longer than 48 characters" },
              }}
              render={({ field }) => (
                <CustomNameInput
                  field={field}
                  onAccept={handleSubmit(onSubmit)}
                  isInvalid={exists(providerNameError)}
                  isDisabled={isLoading || status === "success"}
                />
              )}
            />
            {exists(providerNameError) ? (
              <FormErrorMessage>{providerNameError.message ?? "Error"}</FormErrorMessage>
            ) : (
              <FormHelperText>
                Please give your provider a different name from the one specified in its{" "}
                <Code>provider.yaml</Code>
              </FormHelperText>
            )}
          </FormControl>

          {status === "error" && isError(error) && <ErrorMessageBox error={error} />}
          {isLoading && (
            <LoadingProviderIndicator
              label={`Cloning ${cloneProviderInfo.sourceProviderID} -> ${newProviderName}`}
            />
          )}
        </Form>
      </VStack>
    </>
  )
}

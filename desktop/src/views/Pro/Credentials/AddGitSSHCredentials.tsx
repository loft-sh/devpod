import { Form } from "@/components"
import { exists, useFormErrors } from "@/lib"
import { TGitCredentialData } from "@/types"
import {
  Button,
  FormControl,
  FormErrorMessage,
  FormHelperText,
  FormLabel,
  Grid,
  Input,
  Textarea,
  VStack,
} from "@chakra-ui/react"
import { ReactNode } from "react"
import { FieldError, SubmitHandler, useForm } from "react-hook-form"
import { client as globalClient } from "../../../client"
import { File } from "@/icons"

type TFormValues = {
  [FieldName.NAME]: string
  [FieldName.KEY]: string
}
const FieldName = {
  NAME: "name",
  KEY: "key",
} as const
type TAddGitSSHCredentialsProps = Readonly<{
  isDisabled: boolean
  onCreate(name: string | undefined, data: TGitCredentialData): void
}>
export function AddGitSSHCredentials({ isDisabled, onCreate }: TAddGitSSHCredentialsProps) {
  const { handleSubmit, formState, register, setValue } = useForm<TFormValues>({
    mode: "onBlur",
  })
  const errors = useFormErrors(Object.values(FieldName), formState)

  const onSubmit: SubmitHandler<TFormValues> = (data) => {
    try {
      const privateKey = data[FieldName.KEY]
      const privateKeyBase64 = window.btoa(privateKey)
      onCreate(data[FieldName.NAME], {
        key: privateKeyBase64,
      })
    } catch {
      // noop, shouldn't happen
    }
  }

  const handleSelectFileKeyClicked = async () => {
    try {
      const sshDir = await globalClient.getDir("SSH")
      const fileName = await globalClient.selectFile(sshDir)
      if (!fileName || Array.isArray(fileName)) {
        return
      }

      const privateKeyRaw = await globalClient.readFile([fileName])
      const privateKey = new TextDecoder().decode(privateKeyRaw)
      setValue(FieldName.KEY, privateKey)
    } catch {
      // noop
    }
  }

  return (
    <Form paddingX="4" paddingTop="4" onSubmit={handleSubmit(onSubmit)}>
      <VStack gap="6">
        <FormSection
          isDisabled={isDisabled}
          description="The private key to authenticate against a git provider"
          label="Private Key"
          error={errors.keyError}>
          <VStack>
            <Textarea
              variant="outline"
              minH="32"
              spellCheck={false}
              placeholder={`-----BEGIN OPENSSH PRIVATE KEY-----
              ...
-----END OPENSSH PRIVATE KEY-----
              `}
              {...register(FieldName.KEY)}
            />
            <Button
              alignSelf="flex-end"
              isDisabled={isDisabled}
              leftIcon={<File boxSize={4} />}
              onClick={handleSelectFileKeyClicked}>
              Upload from file
            </Button>
          </VStack>
        </FormSection>

        <FormSection
          isDisabled={isDisabled}
          isRequired={false}
          description="Optionally give your credential a name. Leave it empty to generate a random name"
          label="Name"
          error={errors.nameError}>
          <Input
            spellCheck={false}
            placeholder="git-private-key"
            type="text"
            {...register(FieldName.NAME, {
              validate: (value) => {
                if (!value) {
                  return "Name is required"
                }

                if (!/^[a-z][a-z0-9-_]*$/.test(value)) {
                  return "Name can only contain lowercase letters, numbers, - and _"
                }

                return undefined
              },
            })}
          />
        </FormSection>
      </VStack>

      <Button
        mt="4"
        alignSelf={"end"}
        w="fit-content"
        type="submit"
        variant="primary"
        isLoading={formState.isSubmitting || isDisabled}
        isDisabled={!formState.isValid}
        title="Login">
        Add Key
      </Button>
    </Form>
  )
}

type TFormSectionProps = Readonly<{
  label: string
  description: string
  error: FieldError | undefined
  isDisabled: boolean
  isRequired?: boolean
  children: ReactNode
}>
function FormSection({
  label,
  description,
  error,
  isDisabled,
  isRequired = true,
  children,
}: TFormSectionProps) {
  return (
    <FormControl isRequired={isRequired} isInvalid={exists(error)} isDisabled={isDisabled}>
      <Grid gridTemplateColumns="20rem 1fr" columnGap="10" width="full">
        <VStack align="start" justifyContent={"start"} gap="0">
          <FormLabel mb="0">{label}</FormLabel>
          <FormHelperText mt="1">{description}</FormHelperText>
          {exists(error) && <FormErrorMessage>{error.message ?? "Error"}</FormErrorMessage>}
        </VStack>

        {children}
      </Grid>
    </FormControl>
  )
}

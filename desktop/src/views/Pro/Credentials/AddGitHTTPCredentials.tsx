import { client as globalClient } from "@/client"
import { DaemonClient } from "@/client/pro/client"
import { Form } from "@/components"
import { useProContext } from "@/contexts"
import { CheckCircle, Form as FormIcon } from "@/icons"
import { exists, useFormErrors } from "@/lib"
import { TGitCredentialData } from "@/types"
import { QuestionIcon } from "@chakra-ui/icons"
import {
  Button,
  Divider,
  FormControl,
  FormErrorMessage,
  FormHelperText,
  FormLabel,
  Grid,
  HStack,
  Input,
  Link,
  Text,
  VStack,
} from "@chakra-ui/react"
import { useQuery } from "@tanstack/react-query"
import { ReactNode, useState } from "react"
import { FieldError, SubmitHandler, useForm } from "react-hook-form"

type TFormValues = {
  [FieldName.NAME]: string
  [FieldName.TYPE]: string
  [FieldName.HOST]: string | undefined
  [FieldName.PATH]: string | undefined
  [FieldName.USERNAME]: string | undefined
  [FieldName.TOKEN]: string | undefined
}
const FieldName = {
  NAME: "name",
  TYPE: "type",
  HOST: "host",
  PATH: "path",
  USERNAME: "username",
  TOKEN: "token",
} as const
type TCreateGitHTTPCredentialsProps = Readonly<{
  isDisabled: boolean
  onCreate(name: string | undefined, data: TGitCredentialData): void
}>
export function AddGitHTTPCredentials({ isDisabled, onCreate }: TCreateGitHTTPCredentialsProps) {
  const { client } = useProContext()
  const { handleSubmit, formState, register, setValue } = useForm<TFormValues>({
    mode: "onBlur",
  })
  const errors = useFormErrors(Object.values(FieldName), formState)
  const [hostQuery, setHostQuery] = useState("")

  const credentialsQuery = useQuery({
    // eslint-disable-next-line @tanstack/query/exhaustive-deps
    queryKey: ["git-credentials", hostQuery],
    queryFn: async () => {
      const res = await (client as DaemonClient).queryGitCredentialsHelper(hostQuery)
      if (!res.ok) {
        return null
      }
      if (!res.val) {
        return null
      }

      return res.val
    },
    enabled: hostQuery.length > 0,
  })

  const onSubmit: SubmitHandler<TFormValues> = (data) => {
    onCreate(data[FieldName.NAME], {
      host: data[FieldName.HOST]!,
      password: data[FieldName.TOKEN]!,
      user: data[FieldName.USERNAME]!,
      path: data[FieldName.PATH],
    })
  }

  const handleFillClicked = () => {
    const data = credentialsQuery.data
    if (!data) {
      return
    }

    setValue(FieldName.HOST, data.host)
    setValue(FieldName.TOKEN, data.password)
    if (data.username) {
      setValue(FieldName.USERNAME, data.username)
    }
    if (data.path) {
      setValue(FieldName.PATH, data.path)
    }
    setHostQuery("")
  }

  return (
    <Form paddingX="4" paddingTop="4" onSubmit={handleSubmit(onSubmit)}>
      <VStack gap="6" align="start">
        <VStack w="full" align="start">
          <Input
            type="text"
            value={hostQuery}
            placeholder="Query your local git credentials by host, i.e. github.com"
            onChange={(e) => setHostQuery(e.target.value)}
          />
          {credentialsQuery.data ? (
            <HStack alignItems={"center"}>
              <CheckCircle boxSize={5} color="green.500" />
              <Text variant="muted" fontSize="sm">
                Found git credentials locally
              </Text>
              <Button
                variant="outline"
                size="xs"
                leftIcon={<FormIcon boxSize={4} />}
                onClick={handleFillClicked}>
                Fill in
              </Button>
            </HStack>
          ) : (
            <HStack alignItems={"center"}>
              <QuestionIcon boxSize={4} color="gray.500" />
              <Text variant="muted" fontSize="sm">
                No credentials for host <q>{hostQuery}</q> found locally
              </Text>
            </HStack>
          )}
        </VStack>

        <Divider my="4" />

        <FormSection
          isDisabled={isDisabled}
          description="Set the git provider host, i.e. github.com"
          label="Host"
          error={errors.hostError}>
          <Input
            spellCheck={false}
            placeholder="github.com"
            type="text"
            {...register(FieldName.HOST)}
          />
        </FormSection>

        <FormSection
          isDisabled={isDisabled}
          description="Set the git user"
          label="User"
          error={errors.usernameError}>
          <Input
            spellCheck={false}
            placeholder="myuser"
            type="text"
            {...register(FieldName.USERNAME)}
          />
        </FormSection>

        <FormSection
          isDisabled={isDisabled}
          description="Set the personal access token"
          label="Token"
          error={errors.tokenError}>
          <Input
            spellCheck={false}
            placeholder="PAT"
            type="password"
            {...register(FieldName.TOKEN)}
          />
        </FormSection>

        <FormSection
          isDisabled={isDisabled}
          isRequired={false}
          description={
            <Text variant="">
              Optionally set the httpPath for the host. This is most commonly required by Azure
              DevOps. If you&apos;re not sure if you need this, leave it empty. Read more in the{" "}
              <Link
                onClick={() =>
                  globalClient.open(
                    "https://git-scm.com/docs/gitcredentials#Documentation/gitcredentials.txt-useHttpPath"
                  )
                }>
                git documentation
              </Link>
            </Text>
          }
          label="Path"
          error={errors.pathError}>
          <Input spellCheck={false} placeholder="/" type="text" {...register(FieldName.PATH)} />
        </FormSection>

        <FormSection
          isDisabled={isDisabled}
          isRequired={false}
          description="Optionally give your credential a name. Leave it empty to generate a random name"
          label="Name"
          error={errors.nameError}>
          <Input
            spellCheck={false}
            placeholder="pat"
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
        Add Token
      </Button>
    </Form>
  )
}

type TFormSectionProps = Readonly<{
  label: string
  description: ReactNode
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

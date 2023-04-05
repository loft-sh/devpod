import { Box, Button, Grid, HStack, Spinner, Text, useBoolean, VStack } from "@chakra-ui/react"
import { useMutation } from "@tanstack/react-query"
import { Fragment, useEffect, useMemo } from "react"
import { useNavigate, useParams } from "react-router"
import { client } from "../../client"
import { ErrorMessageBox } from "../../components"
import { WarningMessageBox } from "../../components/Warning"
import { useProvider } from "../../contexts"
import { exists, isError } from "../../lib"
import { Routes } from "../../routes"
import { TProviderID, TWithProviderID } from "../../types"
import { ConfigureProviderOptionsForm } from "./AddProvider/ConfigureProviderOptionsForm"
import { getOptionValue, getVisibleOptions } from "./helpers"

export function Provider() {
  const navigate = useNavigate()
  const params = useParams()
  const providerID = useMemo(() => Routes.getProviderId(params), [params])
  const [isEditing, setIsEditing] = useBoolean()
  const [[provider, { error }], { remove }] = useProvider(providerID)

  const options = useMemo(() => getVisibleOptions(provider?.state?.options), [provider])

  useEffect(() => {
    if (remove.status === "success") {
      navigate(Routes.PROVIDERS)
    }
  }, [navigate, remove.status])

  if (!exists(provider)) {
    return <Spinner />
  }

  if (isError(error)) {
    return (
      <>
        <Text>Whoops, something went wrong</Text>
        <ErrorMessageBox error={error} />
      </>
    )
  }

  if (!exists(providerID)) {
    return null
  }

  if (!exists(provider.state)) {
    return <UninitializedProvider providerID={providerID} />
  }

  return (
    <>
      <HStack marginTop="-6">
        <Button onClick={() => setIsEditing.toggle()}>Edit</Button>
        <Button
          colorScheme="red"
          onClick={() => remove.run({ providerID })}
          isLoading={remove.status === "loading"}>
          Delete
        </Button>
      </HStack>

      <Box width="full" marginTop={8}>
        {isEditing ? (
          <ConfigureProviderOptionsForm
            providerID={providerID}
            options={provider.state.options ?? {}}
            optionGroups={provider.config?.optionGroups || []}
            onFinish={() => setIsEditing.off()}
          />
        ) : (
          <Grid>
            {options.map((option) => (
              <Fragment key={option.id}>
                <Text fontWeight="bold">{option.displayName}</Text>
                <Text>{getOptionValue(option)}</Text>
              </Fragment>
            ))}
          </Grid>
        )}
      </Box>
    </>
  )
}
type TUninitializedProviderProps = Readonly<{ providerID: TProviderID }>
function UninitializedProvider({ providerID }: TUninitializedProviderProps) {
  const {
    mutate: initalize,
    status,
    error,
  } = useMutation({
    mutationFn: async ({ providerID }: TWithProviderID) => {
      return (await client.providers.initialize(providerID)).unwrap()
    },
  })

  return (
    <VStack align="start">
      {isError(error) ? (
        <ErrorMessageBox error={error} />
      ) : (
        <WarningMessageBox warning="Looks like this provider isn't initialized yet. If this doesn't change soon, try to initialize the provider again." />
      )}
      <Button
        isLoading={status === "loading"}
        onClick={() =>
          /*TODO: stream response for debugging and wire up --debug option*/
          initalize({ providerID })
        }>
        Initialize
      </Button>
    </VStack>
  )
}

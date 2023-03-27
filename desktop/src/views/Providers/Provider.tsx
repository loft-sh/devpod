import { Box, Button, HStack, SimpleGrid, Spinner, Text, useBoolean } from "@chakra-ui/react"
import { useEffect, useMemo } from "react"
import { useNavigate, useParams } from "react-router"
import { ErrorMessageBox } from "../../components"
import { useProvider } from "../../contexts"
import { exists, isError } from "../../lib"
import { Routes } from "../../routes"
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

      <Box width="full">
        {isEditing ? (
          <ConfigureProviderOptionsForm
            providerID={providerID}
            options={provider.state?.options ?? {}}
            onFinish={() => setIsEditing.off()}
          />
        ) : (
          <SimpleGrid>
            {options.map((option) => (
              <HStack key={option.id}>
                <Text fontWeight="bold">{option.displayName}</Text>
                <Text>{getOptionValue(option)}</Text>
              </HStack>
            ))}
          </SimpleGrid>
        )}
      </Box>
    </>
  )
}

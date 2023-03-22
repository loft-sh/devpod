import {
  Button,
  ButtonGroup,
  Card,
  CardBody,
  CardFooter,
  CardHeader,
  Heading,
  Link,
  SimpleGrid,
} from "@chakra-ui/react"
import { useMemo } from "react"
import { Link as RouterLink } from "react-router-dom"
import { useProviders } from "../../contexts"
import { exists, noop } from "../../lib"
import { Routes } from "../../routes"
import { TProviderID } from "../../types"

type TProviderInfo = Readonly<{ name: TProviderID }>
export function ListProviders() {
  const [[providers], { remove }] = useProviders()
  const providersInfo = useMemo<readonly TProviderInfo[]>(() => {
    if (!exists(providers)) {
      return []
    }

    return Object.entries(providers).map(([name, details]) => {
      return { name, options: JSON.stringify(details.config, null, 2) }
    })
  }, [providers])

  return (
    <SimpleGrid spacing={6} templateColumns="repeat(auto-fill, minmax(20rem, 1fr))">
      {providersInfo.map(({ name }) => (
        <Card variant="outline" key={name}>
          <CardHeader>
            <Heading size="md">
              <Link as={RouterLink} to={Routes.toProvider(name)}>
                {name}
              </Link>
            </Heading>
          </CardHeader>
          <CardBody></CardBody>
          <CardFooter justify="end">
            <ButtonGroup>
              <Button onClick={noop} isLoading={false}>
                Update
              </Button>
              <Button
                colorScheme="red"
                isLoading={remove.status === "loading" && remove.target?.providerID === name}
                onClick={() => remove.run({ providerID: name })}>
                Delete
              </Button>
            </ButtonGroup>
          </CardFooter>
        </Card>
      ))}
    </SimpleGrid>
  )
}

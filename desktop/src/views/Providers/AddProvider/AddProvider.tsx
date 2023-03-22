import { Box, Heading, useBoolean, VStack } from "@chakra-ui/react"
import { useEffect, useRef } from "react"
import { SetupProviderSourceForm } from "./SetupProviderSourceForm"

export function AddProvider() {
  const [ready, setReady] = useBoolean()
  const optionsRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (ready) {
      optionsRef.current?.scrollIntoView({ behavior: "smooth", block: "start" })
    }
  }, [ready])

  return (
    <Box>
      <VStack align="start" spacing={8} width="full">
        <Heading size="sm">1. Setup Provider Source</Heading>
        <SetupProviderSourceForm onFinish={() => setReady.on()} />
      </VStack>

      <VStack align="start" spacing={8} width="full">
        <Heading marginTop={8} size="sm">
          2. Options
        </Heading>
        <VStack ref={optionsRef} align="start" width="full">
          <Box width="full" height="80" backgroundColor="blue" />
        </VStack>
      </VStack>

      <Box height={8} />
    </Box>
  )
}

import { Box, Code, Container, Link, Text, VStack, useColorModeValue } from "@chakra-ui/react"
import { useEffect, useMemo } from "react"
import { Link as RouterLink, useMatch, useRouteError } from "react-router-dom"
import {
  DevPodProvider,
  ProInstancesProvider,
  WorkspaceStore,
  WorkspaceStoreProvider,
  useChangeSettings,
} from "../contexts"
import { Routes } from "../routes"
import { OSSApp } from "./OSSApp"
import { ProApp } from "./ProApp"
import { usePreserveLocation } from "./usePreserveLocation"
import { ErrorBoundary } from "react-error-boundary"
import { ErrorMessageBox } from "@/components"

export function App() {
  const routeMatchPro = useMatch(`${Routes.PRO}/*`)
  usePreserveLocation()
  usePartyParrot()

  const store = useMemo(() => {
    if (routeMatchPro == null) {
      return new WorkspaceStore()
    }
  }, [routeMatchPro])

  return (
    <ErrorBoundary
      fallbackRender={({ error }) => (
        <ErrorMessageBox
          error={error || new Error("Something went wrong. Please restart the application")}
        />
      )}>
      {routeMatchPro == null ? (
        <WorkspaceStoreProvider store={store!}>
          <DevPodProvider>
            <ProInstancesProvider>
              <OSSApp />
            </ProInstancesProvider>
          </DevPodProvider>
        </WorkspaceStoreProvider>
      ) : (
        <ProApp />
      )}
    </ErrorBoundary>
  )
}

export function ErrorPage() {
  const error = useRouteError()
  const contentBackgroundColor = useColorModeValue("white", "background.darkest")

  return (
    <Box height="100vh" width="100vw" backgroundColor={contentBackgroundColor}>
      <Container padding="16">
        <VStack>
          <Text>Whoops, something went wrong or this page doesn&apos;t exist.</Text>
          <Box paddingBottom="6">
            <Link as={RouterLink} to={Routes.ROOT}>
              Go back to home
            </Link>
          </Box>
          <Code>{JSON.stringify(error, null, 2)}</Code>{" "}
        </VStack>
      </Container>
    </Box>
  )
}

function usePartyParrot() {
  const { set: setSettings, settings } = useChangeSettings()

  useEffect(() => {
    const handler = (event: KeyboardEvent) => {
      if (event.shiftKey && event.ctrlKey && event.key.toLowerCase() === "p") {
        const current = settings.partyParrot
        setSettings("partyParrot", !current)
      }
    }
    document.addEventListener("keyup", handler)

    return () => document.addEventListener("keyup", handler)
  }, [setSettings, settings.partyParrot])
}

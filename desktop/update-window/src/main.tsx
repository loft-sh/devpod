import { StrictMode } from "react"
import ReactDOM from "react-dom/client"
import { ThemeProvider } from "@/Theme"
import { SettingsProvider } from "@/contexts"
import { Button, ButtonGroup, Grid, Heading, Text } from "@chakra-ui/react"
import { client } from "@/client"

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(<Root />)

function handleLaterClicked() {
  client.closeCurrentWindow()
}

function handleRestartClicked() {
  client.restart()
}

function Root() {
  return (
    <StrictMode>
      <SettingsProvider>
        <ThemeProvider>
          <Grid width="100vw" height="100vh" placeContent="center" padding="4">
            <Heading size="md">Installed Update</Heading>
            <Text fontSize="md">Restart the application for the changes to take effect</Text>

            <ButtonGroup justifyContent={"end"} marginTop="4">
              <Button variant="ghost" onClick={handleLaterClicked}>
                Not now
              </Button>
              <Button variant="primary" onClick={handleRestartClicked}>
                Restart
              </Button>
            </ButtonGroup>
          </Grid>
        </ThemeProvider>
      </SettingsProvider>
    </StrictMode>
  )
}

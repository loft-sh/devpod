import { client } from "@/client"
import { DevPodProBadge } from "@/icons"
import { BoxProps, Button, ButtonGroup, Grid, GridItem, IconButton } from "@chakra-ui/react"
import { ReactNode, useEffect, useId } from "react"
import { useBorderColor } from "../../Theme"
import { useToolbar } from "../../contexts"
import { Notifications } from "./Notifications"
import { ChevronDownIcon } from "@chakra-ui/icons"

export function Toolbar({ ...boxProps }: BoxProps) {
  const borderColor = useBorderColor()
  const { title, actions } = useToolbar()

  const handleAnnouncementClicked = () => {
    client.openLink("https://devpod.sh/engine")
  }

  return (
    <Grid
      alignContent="center"
      templateRows="1fr"
      templateColumns="minmax(auto, 18rem) 3fr fit-content(15rem)"
      width="full"
      paddingX="4"
      borderBottomColor={borderColor}
      borderBottomWidth="thin"
      {...boxProps}>
      <GridItem display="flex" alignItems="center">
        {title}
      </GridItem>
      <GridItem display="flex" alignItems="center" justifyContent="center">
        {actions}
      </GridItem>
      <GridItem display="flex" alignItems="center" justifyContent="center">
        <Notifications />
        <ButtonGroup isAttached variant="outline">
          <Button
            data-tauri-drag-region // keep!
            leftIcon={<DevPodProBadge width="9" height="8" />}
            onClick={handleAnnouncementClicked}>
            DevPod Pro
          </Button>
          <IconButton
            variant="outline"
            aria-label="Show DevPod Pro instances"
            icon={<ChevronDownIcon boxSize={6} />}
          />
        </ButtonGroup>
      </GridItem>
    </Grid>
  )
}

export function ToolbarTitle({ children }: Readonly<{ children: ReactNode }>) {
  const { setTitle } = useToolbar()

  useEffect(() => {
    setTitle(children)
  }, [children, setTitle])

  return null
}

export function ToolbarActions({ children }: Readonly<{ children: ReactNode }>) {
  const { addAction } = useToolbar()
  const id = useId()

  useEffect(() => {
    const removeActions = addAction(id, children)

    return removeActions
  }, [children, addAction, id])

  return null
}

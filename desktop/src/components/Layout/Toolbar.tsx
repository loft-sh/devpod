import { BoxProps, Grid, GridItem } from "@chakra-ui/react"
import { ReactNode, useEffect, useId } from "react"
import { useBorderColor } from "../../Theme"
import { useToolbar } from "../../contexts"
import { Notifications } from "./Notifications"
import { Pro } from "./Pro"

export function Toolbar({ ...boxProps }: BoxProps) {
  const borderColor = useBorderColor()
  const { title, actions } = useToolbar()

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
        <Pro />
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

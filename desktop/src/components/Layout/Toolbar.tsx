import { Box, BoxProps } from "@chakra-ui/react"
import { ReactNode, useEffect, useId } from "react"
import { useBorderColor } from "../../Theme"
import { useToolbar } from "../../contexts"

export function Toolbar({ ...boxProps }: BoxProps) {
  const borderColor = useBorderColor()

  return <Box borderBottomColor={borderColor} borderBottomWidth="thin" {...boxProps} />
}

function Title() {
  const { title } = useToolbar()

  return <>{title}</>
}

function Actions() {
  const { actions } = useToolbar()

  return <>{actions}</>
}

Toolbar.Title = Title
Toolbar.Actions = Actions

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

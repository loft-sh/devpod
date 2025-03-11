import { ReactNode, useCallback, useMemo, useState } from "react"
import { TToolbarAction, TToolbarContext, ToolbarContext } from "./useToolbar"

export function ToolbarProvider({ children }: Readonly<{ children: ReactNode }>) {
  const [title, setTitle] = useState<ReactNode>(null)
  const [actions, setActions] = useState<readonly TToolbarAction[]>([])
  const addAction = useCallback<TToolbarContext["addAction"]>((id, node) => {
    setActions((actions) => {
      const newActions = actions.slice()
      const index = newActions.findIndex((a) => a.id === id)
      if (index !== -1) {
        newActions.splice(index, 1, { id, node: node })
      } else {
        newActions.push({ id, node: node })
      }

      return newActions
    })

    return () => {
      setActions((actions) => actions.filter((a) => a.id !== id))
    }
  }, [])
  const value = useMemo(
    () => ({ title, setTitle, actions: actions.map((a) => a.node), addAction }),
    [title, actions, addAction]
  )

  return <ToolbarContext.Provider value={value}>{children}</ToolbarContext.Provider>
}

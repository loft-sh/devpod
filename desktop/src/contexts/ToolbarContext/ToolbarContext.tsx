import { ReactNode, createContext, useContext, useMemo, useState, useCallback } from "react"

type TToolbarContext = Readonly<{
  title: ReactNode
  setTitle: (title: ReactNode) => void
  actions: readonly ReactNode[]
  addAction: (id: string, action: ReactNode) => void
}>
type TToolbarAction = Readonly<{
  id: string
  node: ReactNode
}>
const ToolbarContext = createContext<TToolbarContext>(null!)

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

export function useToolbar() {
  return useContext(ToolbarContext)
}

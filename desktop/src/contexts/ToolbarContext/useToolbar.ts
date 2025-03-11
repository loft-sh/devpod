import { ReactNode, createContext, useContext } from "react"

export type TToolbarContext = Readonly<{
  title: ReactNode
  setTitle: (title: ReactNode) => void
  actions: readonly ReactNode[]
  addAction: (id: string, action: ReactNode) => void
}>
export type TToolbarAction = Readonly<{
  id: string
  node: ReactNode
}>
export const ToolbarContext = createContext<TToolbarContext>(null!)
export function useToolbar() {
  return useContext(ToolbarContext)
}

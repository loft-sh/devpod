import { TWorkspace } from "@/types"
import { useMemo } from "react"
import { ProWorkspaceInstance } from "@/contexts"
import { getLastActivity } from "@/lib/pro"

export enum ESortWorkspaceMode {
  RECENTLY_USED = "Recently Used",
  LEAST_RECENTLY_USED = "Least Recently Used",
  RECENTLY_CREATED = "Recently Created",
  LEAST_RECENTLY_CREATED = "Least Recently Created",
}

export const DEFAULT_SORT_WORKSPACE_MODE = ESortWorkspaceMode.RECENTLY_USED

type TSortable<TOriginal> = {
  original: TOriginal
  created: number
  used: number
}

function sortWorkspaces<T>(
  sortables: TSortable<T>[],
  sortMode: ESortWorkspaceMode | undefined
): T[] {
  const copy = [...sortables]

  copy.sort((a, b) => {
    if (sortMode === ESortWorkspaceMode.RECENTLY_USED) {
      return a.used > b.used ? -1 : 1
    }

    if (sortMode === ESortWorkspaceMode.LEAST_RECENTLY_USED) {
      return b.used > a.used ? -1 : 1
    }

    if (sortMode === ESortWorkspaceMode.RECENTLY_CREATED) {
      return a.created > b.created ? -1 : 1
    }

    if (sortMode === ESortWorkspaceMode.LEAST_RECENTLY_CREATED) {
      return b.created > a.created ? -1 : 1
    }

    return 0
  })

  return copy.map((copy) => copy.original)
}

export function useSortWorkspaces(
  workspaces: readonly TWorkspace[] | undefined,
  sortMode: ESortWorkspaceMode | undefined
) {
  return useMemo(() => {
    if (!workspaces) {
      return undefined
    }

    const sortables = workspaces.map((workspace) => ({
      original: workspace,
      created: new Date(workspace.creationTimestamp).getTime(),
      used: new Date(workspace.lastUsed).getTime(),
    }))

    return sortWorkspaces(sortables, sortMode)
  }, [workspaces, sortMode])
}

export function useSortProWorkspaces(
  workspaces: readonly ProWorkspaceInstance[] | undefined,
  sortMode: ESortWorkspaceMode | undefined
) {
  return useMemo(() => {
    if (!workspaces) {
      return undefined
    }

    const sortables = workspaces.map((workspace) => ({
      original: workspace,
      created: new Date(workspace.metadata?.creationTimestamp ?? 0).getTime(),
      used: getLastActivity(workspace)?.getTime() ?? 0,
    }))

    return sortWorkspaces(sortables, sortMode)
  }, [workspaces, sortMode])
}

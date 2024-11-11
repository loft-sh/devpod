import { ProWorkspaceInstance } from "@/contexts"
import { Annotations } from "./constants"

export function getLastActivity(instance: ProWorkspaceInstance): Date | undefined {
  const maybeTimestamp = instance.metadata?.annotations?.[Annotations.SleepModeLastActivity]
  if (!maybeTimestamp) {
    return undefined
  }

  const timestamp = Number.parseInt(maybeTimestamp)
  if (Number.isNaN(timestamp)) {
    return undefined
  }

  return new Date(timestamp * 1_000)
}

import { ProClient } from "@/client"
import { TWorkspaceOwnerFilterState } from "@/components"
import { ManagementV1Project } from "@loft-enterprise/client/gen/models/managementV1Project"
import { ManagementV1Self } from "@loft-enterprise/client/gen/models/managementV1Self"
import { UseQueryResult } from "@tanstack/react-query"
import { Dispatch, SetStateAction, createContext, useContext } from "react"

export type TProContext = Readonly<{
  managementSelfQuery: UseQueryResult<ManagementV1Self | undefined>
  currentProject?: ManagementV1Project
  host: string
  client: ProClient
  isLoadingWorkspaces: boolean
  ownerFilter: TWorkspaceOwnerFilterState
  setOwnerFilter: Dispatch<SetStateAction<TWorkspaceOwnerFilterState>>
}>
export const ProContext = createContext<TProContext>(null!)

export function useProContext() {
  return useContext(ProContext)
}

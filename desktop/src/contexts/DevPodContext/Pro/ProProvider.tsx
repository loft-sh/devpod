import { ProClient, client as globalClient } from "@/client"
import { TWorkspaceOwnerFilterState, ToolbarActions, ToolbarTitle } from "@/components"
import { Annotations, Result } from "@/lib"
import { Routes } from "@/routes"
import { Text } from "@chakra-ui/react"
import { ManagementV1Project } from "@loft-enterprise/client/gen/models/managementV1Project"
import { ManagementV1Self } from "@loft-enterprise/client/gen/models/managementV1Self"
import { UseQueryResult, useQuery } from "@tanstack/react-query"
import {
  Dispatch,
  ReactNode,
  SetStateAction,
  createContext,
  useContext,
  useEffect,
  useMemo,
  useState,
} from "react"
import { Navigate, useNavigate } from "react-router-dom"
import { ProWorkspaceStore, useWorkspaceStore } from "../workspaceStore"
import { ContextSwitcher, HOST_OSS } from "./ContextSwitcher"
import { useProInstances } from "../proInstances"

type TProContext = Readonly<{
  managementSelfQuery: UseQueryResult<ManagementV1Self | undefined>
  projectsQuery: UseQueryResult<readonly ManagementV1Project[] | undefined>
  currentProject?: ManagementV1Project
  host: string
  client: ProClient
  isLoadingWorkspaces: boolean
  ownerFilter: TWorkspaceOwnerFilterState
  setOwnerFilter: Dispatch<SetStateAction<TWorkspaceOwnerFilterState>>
}>
const ProContext = createContext<TProContext>(null!)
export function ProProvider({ host, children }: { host: string; children: ReactNode }) {
  const [[proInstances, { status: proInstancesStatus }]] = useProInstances()
  const [isLoadingWorkspaces, setIsLoadingWorkspaces] = useState(false)
  const [ownerFilter, setOwnerFilter] = useState<TWorkspaceOwnerFilterState>("self")
  const navigate = useNavigate()
  const currentProInstance = useMemo(() => {
    return proInstances?.find((instance) => instance.host == host)
  }, [host, proInstances])
  const { store } = useWorkspaceStore<ProWorkspaceStore>()
  const client = useMemo(() => {
    if (!currentProInstance) {
      return null
    }

    return globalClient.getProClient(currentProInstance)
  }, [currentProInstance])
  const [selectedProject, setSelectedProject] = useState<ManagementV1Project | null>(null)
  const managementSelfQuery = useQuery({
    queryKey: ["managementSelf", client],
    queryFn: async () => {
      return (await client!.getSelf()).unwrap()
    },
    enabled: !!client,
  })
  const projectsQuery = useQuery({
    queryKey: ["pro", host, "projects", client],
    queryFn: async () => {
      return (await client!.listProjects()).unwrap()
    },
    enabled: !!client,
  })

  const currentProject = useMemo<ManagementV1Project | undefined>(() => {
    if (selectedProject) {
      return selectedProject
    }

    return projectsQuery.data?.[0]
  }, [projectsQuery, selectedProject])

  const [cancelWatch, setCancelWatch] = useState<
    { fn: () => Promise<Result<undefined>> } | undefined
  >(undefined)

  const [waitingForCancel, setWaitingForCancel] = useState<boolean>(false)

  useEffect(() => {
    if (!currentProject?.metadata?.name || !client) {
      return
    }
    setIsLoadingWorkspaces(true)

    let canceled = false

    const toCancel = client.watchWorkspaces(
      currentProject.metadata.name,
      ownerFilter,
      (workspaces) => {
        if (canceled) {
          return
        }

        // sort by last activity (newest > oldest)
        const sorted = workspaces.slice().sort((a, b) => {
          const lastActivityA = a.metadata?.annotations?.[Annotations.SleepModeLastActivity]
          const lastActivityB = b.metadata?.annotations?.[Annotations.SleepModeLastActivity]
          if (!(lastActivityA && lastActivityB)) {
            return 0
          }

          return parseInt(lastActivityB, 10) - parseInt(lastActivityA, 10)
        })
        console.log(sorted)
        store.setWorkspaces(sorted)
        // dirty, dirty
        setTimeout(() => {
          setIsLoadingWorkspaces(false)
        }, 1_000)
      }
    )

    function canceler() {
      canceled = true
      setCancelWatch(undefined)
      setWaitingForCancel(true)

      return toCancel().finally(() => setWaitingForCancel(false))
    }

    setCancelWatch({ fn: canceler })

    return () => {
      canceler()
    }
  }, [client, store, currentProject, ownerFilter])

  const handleProjectChanged = (newProject: ManagementV1Project) => {
    setSelectedProject(newProject)
    navigate(Routes.toProInstance(host))
  }

  const handleHostChanged = (newHost: string) => {
    if (newHost === HOST_OSS) {
      navigate(Routes.WORKSPACES)

      return
    }

    setSelectedProject(null) // reset selected project
    navigate(Routes.toProInstance(newHost))
  }

  const value = useMemo<TProContext>(() => {
    return {
      managementSelfQuery,
      currentProject,
      projectsQuery,
      host,
      client: client!,
      isLoadingWorkspaces,
      ownerFilter,
      setOwnerFilter,
    }
  }, [
    managementSelfQuery,
    currentProject,
    projectsQuery,
    host,
    client,
    isLoadingWorkspaces,
    ownerFilter,
  ])

  // this pro instance doesn't exist, let's route back to root
  if (proInstancesStatus == "success" && !currentProInstance) {
    return <Navigate to={Routes.ROOT} />
  }

  if (!client) {
    return null
  }

  return (
    <ProContext.Provider value={value}>
      <ToolbarTitle>
        <Text maxW="60" fontSize="sm" overflow="hidden" textOverflow="ellipsis" whiteSpace="nowrap">
          {host}
        </Text>
      </ToolbarTitle>
      <ToolbarActions>
        <ContextSwitcher
          currentHost={host}
          onHostChange={handleHostChanged}
          projects={projectsQuery.data ?? []}
          currentProject={currentProject!}
          onProjectChange={handleProjectChanged}
          onCancelWatch={cancelWatch?.fn}
          waitingForCancel={waitingForCancel}
        />
      </ToolbarActions>
      {children}
    </ProContext.Provider>
  )
}

export function useProContext() {
  return useContext(ProContext)
}

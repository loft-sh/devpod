import { client as globalClient } from "@/client"
import { DaemonClient } from "@/client/pro/client"
import { TWorkspaceOwnerFilterState, ToolbarActions, ToolbarTitle } from "@/components"
import { Annotations, Result } from "@/lib"
import { Routes } from "@/routes"
import { Text } from "@chakra-ui/react"
import { ManagementV1Project } from "@loft-enterprise/client/gen/models/managementV1Project"
import { useQuery } from "@tanstack/react-query"
import { ReactNode, useEffect, useMemo, useState } from "react"
import { Navigate, useNavigate } from "react-router-dom"
import { useProInstances } from "../proInstances"
import { ProWorkspaceStore, useWorkspaceStore } from "../workspaceStore"
import { ContextSwitcher } from "./ContextSwitcher"
import { HOST_OSS } from "./constants"
import { ProContext, TProContext } from "./useProContext"

export function ProProvider({ host, children }: { host: string; children: ReactNode }) {
  const [[proInstances, { status: proInstancesStatus }]] = useProInstances()
  const [isLoadingWorkspaces, setIsLoadingWorkspaces] = useState(false)
  const [ownerFilter, setOwnerFilter] = useState<TWorkspaceOwnerFilterState>("self")
  const [currentProjectName, setCurrentProjectName] = useState<string | undefined>(
    () => localStorage.getItem(getProjectStorageKey(host)) ?? undefined
  )
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
    if (projectsQuery.data == null) {
      return undefined
    }

    if (!currentProjectName) {
      return projectsQuery.data[0] ?? undefined
    }

    const maybeProject =
      projectsQuery.data.find((project) => project.metadata?.name === currentProjectName) ??
      undefined
    if (!maybeProject) {
      return projectsQuery.data[0] ?? undefined
    }

    return maybeProject
  }, [currentProjectName, projectsQuery.data])

  const [cancelWatch, setCancelWatch] = useState<
    { fn: () => Promise<Result<undefined>> } | undefined
  >(undefined)

  const [waitingForCancel, setWaitingForCancel] = useState<boolean>(false)

  useEffect(() => {
    if (!currentProject?.metadata?.name || !client) {
      return
    }
    setIsLoadingWorkspaces(true)

    if (client instanceof DaemonClient) {
      // daemon client impl
      return client.watchWorkspaces(currentProject.metadata.name, ownerFilter, (workspaces) => {
        // sort by last activity (newest > oldest)
        const sorted = workspaces.slice().sort((a, b) => {
          const lastActivityA = a.metadata?.annotations?.[Annotations.SleepModeLastActivity]
          const lastActivityB = b.metadata?.annotations?.[Annotations.SleepModeLastActivity]
          if (!(lastActivityA && lastActivityB)) {
            return 0
          }

          return parseInt(lastActivityB, 10) - parseInt(lastActivityA, 10)
        })
        store.setWorkspaces(sorted)
        setIsLoadingWorkspaces(false)
      })
    } else {
      let canceled = false
      // proxy client impl
      const toCancel = client.watchWorkspacesProxy(
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
    }
  }, [client, store, currentProject, ownerFilter])

  const handleProjectChanged = (newProject: ManagementV1Project) => {
    const projectName = newProject.metadata?.name ?? ""
    localStorage.setItem(getProjectStorageKey(host), projectName)
    setCurrentProjectName(projectName)
    navigate(Routes.toProInstance(host))
  }

  const handleHostChanged = (newHost: string) => {
    if (newHost === HOST_OSS) {
      navigate(Routes.WORKSPACES)

      return
    }

    const projectName = localStorage.getItem(getProjectStorageKey(newHost)) ?? undefined
    setCurrentProjectName(projectName)
    navigate(Routes.toProInstance(newHost))
  }

  const value = useMemo<TProContext>(() => {
    return {
      managementSelfQuery,
      currentProject,
      host,
      client: client!,
      isLoadingWorkspaces,
      ownerFilter,
      setOwnerFilter,
    }
  }, [managementSelfQuery, currentProject, host, client, isLoadingWorkspaces, ownerFilter])

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

const PROJECT_STORAGE_KEY = "devpod_current_project"

function getProjectStorageKey(host: string) {
  return `${PROJECT_STORAGE_KEY}_${host}`
}

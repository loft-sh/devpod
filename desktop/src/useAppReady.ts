import { useEffect, useId, useRef } from "react"
import { useNavigate } from "react-router"
import { client } from "./client"
import { startWorkspaceAction } from "./contexts"
import { Routes } from "./routes"

export function useAppReady() {
  const isReadyLockRef = useRef<boolean>(false)
  const viewID = useId()
  const navigate = useNavigate()

  // notifies underlying layer that ui is ready for communication
  useEffect(() => {
    if (!isReadyLockRef.current) {
      isReadyLockRef.current = true
      ;(async () => {
        const unsubscribe = await client.subscribe("event", async (event) => {
          if (event === "ShowDashboard") {
            navigate(Routes.WORKSPACES)
          } else {
            const data = event.OpenWorkspace
            const workspacesResult = await client.workspaces.listAll()
            if (workspacesResult.err) {
              return
            }
            const maybeWorkspace = workspacesResult.val.find((w) => w.id === data.workspace_id)

            if (maybeWorkspace !== undefined) {
              const actionID = startWorkspaceAction({
                workspaceID: maybeWorkspace.id,
                streamID: viewID,
                config: {
                  id: maybeWorkspace.id,
                  providerConfig: {
                    providerID: maybeWorkspace.provider?.name ?? undefined,
                  },
                  ideConfig: {
                    name: maybeWorkspace.ide?.name ?? null,
                  },
                },
              })
              // TODO: Continue here: navigate to actionID
              // For now, navigate to workspace view

              navigate(Routes.toWorkspace(maybeWorkspace.id))

              return
            }

            navigate(
              Routes.toWorkspaceCreate({
                providerID: data.provider_id,
                rawSource: data.source,
                ide: data.ide,
              })
            )
          }
        })

        await client.ready()

        return unsubscribe
      })()
    }
  }, [navigate, viewID])
}

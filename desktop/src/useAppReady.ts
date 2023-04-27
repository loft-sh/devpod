import { useEffect, useId, useRef } from "react"
import { useNavigate } from "react-router"
import { client } from "./client"
import { startWorkspaceAction } from "./contexts"
import { Routes } from "./routes"
import { appWindow } from "@tauri-apps/api/window"

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
          await appWindow.setFocus()

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

              navigate(Routes.toAction(actionID))

              return
            }

            navigate(
              Routes.toWorkspaceCreate({
                workspaceID: data.workspace_id,
                providerID: data.provider_id,
                rawSource: data.source,
                ide: data.ide,
              })
            )
          }
        })

        try {
          await client.ready()
        } catch (err) {
          console.error(err)
        }

        return unsubscribe
      })()
    }
  }, [navigate, viewID])
}

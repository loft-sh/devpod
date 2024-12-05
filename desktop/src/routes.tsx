import { Outlet, Params, Path, createBrowserRouter } from "react-router-dom"
import { App, ErrorPage } from "./App"
import { TActionID } from "./contexts"
import { TProInstanceDetail, exists } from "./lib"
import { TProviderID, TSupportedIDE, TWorkspaceID } from "./types"
import { Actions, Pro, Providers, Settings, Workspaces } from "./views"

export const Routes = {
  ROOT: "/",
  SETTINGS: "/settings",
  WORKSPACES: "/workspaces",
  ACTIONS: "/actions",
  get ACTION(): string {
    return `${Routes.ACTIONS}/:action`
  },
  get WORKSPACE_CREATE(): string {
    return `${Routes.WORKSPACES}/new`
  },
  toWorkspaceCreate(
    options: Readonly<{
      workspaceID: TWorkspaceID | null
      providerID: TProviderID | null
      ide: string | null
      rawSource: string | null
    }>
  ): Partial<Path> {
    const searchParams = new URLSearchParams()
    for (const [key, value] of Object.entries(options)) {
      if (exists(value)) {
        searchParams.set(key, value)
      }
    }

    return {
      pathname: Routes.WORKSPACE_CREATE,
      search: searchParams.toString(),
    }
  },
  toAction(actionID: TActionID, onSuccess?: string): string {
    if (onSuccess) {
      return `${Routes.ACTIONS}/${actionID}?onSuccess=${encodeURIComponent(onSuccess)}`
    }

    return `${Routes.ACTIONS}/${actionID}`
  },
  getActionID(params: Params<string>): string | undefined {
    // Needs to match `:action` from detail route exactly!
    return params["action"]
  },
  getWorkspaceCreateParamsFromSearchParams(searchParams: URLSearchParams): Partial<
    Readonly<{
      workspaceID: TWorkspaceID
      providerID: TProviderID
      ide: TSupportedIDE
      rawSource: string
    }>
  > {
    return {
      workspaceID: searchParams.get("workspaceID") ?? undefined,
      providerID: searchParams.get("providerID") ?? undefined,
      ide: (searchParams.get("ide") as TSupportedIDE | null) ?? undefined,
      rawSource: searchParams.get("rawSource") ?? undefined,
    }
  },
  PROVIDERS: "/providers",
  get PROVIDER(): string {
    return `${Routes.PROVIDERS}/:provider`
  },
  toProvider(providerID: string): string {
    return `${Routes.PROVIDERS}/${providerID}`
  },
  getProviderId(params: Params<string>): string | undefined {
    // Needs to match `:provider` from detail route exactly!
    return params["provider"]
  },
  PRO: "/pro",
  PRO_INSTANCE: "/pro/:host",
  PRO_WORKSPACE: "/pro/:host/:workspace",
  PRO_WORKSPACE_SELECT_PRESET: "/pro/:host/select-preset",
  PRO_WORKSPACE_CREATE: "/pro/:host/new",
  PRO_SETTINGS: "/pro/:host/settings",
  toProInstance(host: string): string {
    // This is a workaround for react-routers interaction with hostnames as path components
    const h = host.replaceAll(".", "-")

    return `/pro/${h}`
  },
  toProWorkspace(host: string, instanceID: string): string {
    const base = this.toProInstance(host)

    return `${base}/${instanceID}`
  },
  toProWorkspaceCreate(host: string, fromPreset?: string): string {
    const base = this.toProInstance(host)

    return `${base}/new${fromPreset ? `?fromPreset=${fromPreset}` : ""}`
  },
  toProSelectPreset(host: string): string {
    const base = this.toProInstance(host)

    return `${base}/select-preset`
  },
  toProWorkspaceDetail(host: string, instanceID: string, detail: TProInstanceDetail): string {
    const base = this.toProInstance(host)

    return `${base}/${instanceID}?tab=${detail}`
  },
  toProSettings(host: string): string {
    const base = this.toProInstance(host)

    return `${base}/settings`
  },
  getProWorkspaceDetailsParams(
    searchParams: URLSearchParams
  ): Partial<Readonly<{ tab: TProInstanceDetail | null }>> {
    return { tab: searchParams.get("tab") as TProInstanceDetail | null }
  },
} as const

export const router = createBrowserRouter([
  {
    path: Routes.ROOT,
    element: <App />,
    errorElement: <ErrorPage />,
    children: [
      {
        path: Routes.PRO,
        element: <ProRoot />,
        children: [
          {
            path: Routes.PRO_INSTANCE,
            element: <Pro.ProInstance />,
            children: [
              {
                index: true,
                element: <Pro.ListWorkspaces />,
              },
              {
                path: Routes.PRO_WORKSPACE,
                element: <Pro.Workspace />,
              },
              {
                path: Routes.PRO_WORKSPACE_CREATE,
                element: <Pro.CreateWorkspace />,
              },
              {
                path: Routes.PRO_WORKSPACE_SELECT_PRESET,
                element: <Pro.SelectPreset />,
              },
              { path: Routes.PRO_SETTINGS, element: <Pro.Settings /> },
            ],
          },
        ],
      },
      {
        path: Routes.WORKSPACES,
        element: <Workspaces.Workspaces />,
        children: [
          {
            index: true,
            element: <Workspaces.ListWorkspaces />,
          },
          {
            path: Routes.WORKSPACE_CREATE,
            element: <Workspaces.CreateWorkspace />,
          },
        ],
      },
      {
        path: Routes.PROVIDERS,
        element: <Providers.Providers />,
        children: [
          { index: true, element: <Providers.ListProviders /> },
          {
            path: Routes.PROVIDER,
            element: <Providers.Provider />,
          },
        ],
      },
      {
        path: Routes.ACTIONS,
        element: <Actions.Actions />,
        children: [{ path: Routes.ACTION, element: <Actions.Action /> }],
      },
      { path: Routes.SETTINGS, element: <Settings.Settings /> },
    ],
  },
])

function ProRoot() {
  return <Outlet />
}

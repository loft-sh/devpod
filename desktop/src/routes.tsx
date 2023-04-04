import { createBrowserRouter, Params, Path } from "react-router-dom"
import { App, ErrorPage } from "./App"
import { TActionID } from "./contexts"
import { exists } from "./lib"
import { TProviderID, TSupportedIDE } from "./types"
import {
  AddProvider,
  CreateWorkspace,
  ListProviders,
  ListWorkspaces,
  Provider,
  Providers,
  Settings,
  Workspace,
  Workspaces,
} from "./views"

export const Routes = {
  ROOT: "/",
  SETTINGS: "/settings",
  WORKSPACES: "/workspaces",
  get WORKSPACE() {
    return `${Routes.WORKSPACES}/:workspace`
  },
  get WORKSPACE_CREATE() {
    return `${Routes.WORKSPACES}/new`
  },
  toWorkspaceCreate(
    options: Readonly<{
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
  toWorkspace(workspaceID: string, actionID?: TActionID) {
    return `${Routes.WORKSPACES}/${workspaceID}?actionID=${actionID}`
  },
  getWorkspaceId(params: Params<string>): string | undefined {
    // Needs to match `:workspace` from detail route exactly!
    return params["workspace"]
  },
  getActionIDFromSearchParams(searchParams: URLSearchParams): TActionID | undefined {
    return (searchParams.get("actionID") ?? undefined) as TActionID | undefined
  },
  getWorkspaceCreateParamsFromSearchParams(searchParams: URLSearchParams): Partial<
    Readonly<{
      providerID: TProviderID
      ide: TSupportedIDE
      rawSource: string
    }>
  > {
    return {
      providerID: searchParams.get("providerID") ?? undefined,
      ide: (searchParams.get("ide") as TSupportedIDE | null) ?? undefined,
      rawSource: searchParams.get("rawSource") ?? undefined,
    }
  },
  PROVIDERS: "/providers",
  get PROVIDER() {
    return `${Routes.PROVIDERS}/:provider`
  },
  get PROVIDER_ADD() {
    return `${Routes.PROVIDERS}/add`
  },
  toProvider(providerID: string) {
    return `${Routes.PROVIDERS}/${providerID}`
  },
  getProviderId(params: Params<string>): string | undefined {
    // Needs to match `:provider` from detail route exactly!
    return params["provider"]
  },
} as const

export const router = createBrowserRouter([
  {
    path: Routes.ROOT,
    element: <App />,
    errorElement: <ErrorPage />,
    children: [
      {
        path: Routes.WORKSPACES,
        element: <Workspaces />,
        children: [
          {
            index: true,
            element: <ListWorkspaces />,
          },
          {
            path: Routes.WORKSPACE,
            element: <Workspace />,
          },
          {
            path: Routes.WORKSPACE_CREATE,
            element: <CreateWorkspace />,
          },
        ],
      },
      {
        path: Routes.PROVIDERS,
        element: <Providers />,
        children: [
          { index: true, element: <ListProviders /> },
          {
            path: Routes.PROVIDER,
            element: <Provider />,
          },
          {
            path: Routes.PROVIDER_ADD,
            element: <AddProvider />,
          },
        ],
      },
      { path: Routes.SETTINGS, element: <Settings /> },
    ],
  },
])

import { createBrowserRouter, Params } from "react-router-dom"
import { App, ErrorPage } from "./App"
import { TActionID } from "./contexts"
import { TAddProviderConfig } from "./types"
import {
  CreateWorkspace,
  ListWorkspaces,
  Providers,
  Settings,
  Workspace,
  Workspaces,
  ListProviders,
  Provider,
  AddProvider,
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

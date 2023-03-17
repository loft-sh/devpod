import { createBrowserRouter, Params } from "react-router-dom"
import { App, ErrorPage } from "./App"
import {
  CreateWorkspace,
  ListWorkspaces,
  Providers,
  Settings,
  Workspace,
  Workspaces,
} from "./views"

export const Routes = {
  ROOT: "/",
  SETTINGS: "/settings",
  WORKSPACES: "/workspaces",
  PROVIDERS: "/providers",
  get WORKSPACE() {
    return `${Routes.WORKSPACES}/:workspace`
  },
  get WORKSPACE_CREATE() {
    return `${Routes.WORKSPACES}/new`
  },
  toWorkspace(workspaceId: string) {
    return `${Routes.WORKSPACES}/${workspaceId}`
  },
  getWorkspaceId(params: Params<string>): string | undefined {
    // needs to match `:workspace` from detail route exactly
    return params["workspace"]
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
      },
      { path: Routes.SETTINGS, element: <Settings /> },
    ],
  },
])

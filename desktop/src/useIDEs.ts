import { useQuery } from "@tanstack/react-query"
import { useMemo } from "react"
import { client } from "./client"
import { useSettings } from "./contexts"
import { QueryKeys } from "./queryKeys"
import { TIDE, TIDEs } from "@/types"

// See pkg/config/ide.go for names
const FLEET_IDE_NAME = "fleet"
const JUPYTER_IDE_NAME = "jupyternotebook"
const VSCODE_INSIDERS = "vscode-insiders"
const CURSOR = "cursor"
const POSITRON = "positron"
const CODIUM = "codium"
const ZED = "zed"
const RSTUDIO = "rstudio"

export function useIDEs() {
  const idesQuery = useQuery({
    queryKey: QueryKeys.IDES,
    queryFn: async () => (await client.ides.listAll()).unwrap(),
  })
  const settings = useSettings()

  const ides = useMemo(
    () =>
      idesQuery.data?.filter((ide) => {
        if (!ide.experimental) return true

        if (ide.name === FLEET_IDE_NAME && settings.experimental_fleet) return true
        if (ide.name === JUPYTER_IDE_NAME && settings.experimental_jupyterNotebooks) return true
        if (ide.name === VSCODE_INSIDERS && settings.experimental_vscodeInsiders) return true
        if (ide.name === CURSOR && settings.experimental_cursor) return true
        if (ide.name === POSITRON && settings.experimental_positron) return true
        if (ide.name === CODIUM && settings.experimental_codium) return true
        if (ide.name === ZED && settings.experimental_zed) return true
        if (ide.name === RSTUDIO && settings.experimental_rstudio) return true

        return false
      }),
    [settings, idesQuery.data]
  )

  return useMemo(
    () => ({ ides, defaultIDE: idesQuery.data?.find((ide) => ide.default) }),
    [ides, idesQuery.data]
  )
}

export function useGroupIDEs(ides?: TIDEs) {
  return useMemo(() => {
    return ides?.reduce(
      (accum, ide) => {
        const group = ide.group ?? "Other"

        if (group === "Primary") {
          accum.primary.push(ide)
        } else {
          if (!accum.subMenus[group]) {
            accum.subMenus[group] = []
            accum.subMenuGroups.push(group)
          }

          accum.subMenus[group]!.push(ide)
        }

        return accum
      },
      {
        primary: [] as TIDE[],
        subMenuGroups: [] as string[],
        subMenus: {} as { [key: string]: TIDE[] },
      }
    )
  }, [ides])
}

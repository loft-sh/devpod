import { useEffect } from "react"
import { Location, matchRoutes, useLocation } from "react-router"
import { client } from "../client"
import { LocalStorageBackend, Store } from "../lib"
import { Routes } from "@/routes"

const LOCATION_KEY = "location"
const CURRENT_LOCATION_KEY = "current"
type TLocationStore = { [CURRENT_LOCATION_KEY]: Location }
const store = new Store<TLocationStore>(new LocalStorageBackend<TLocationStore>(LOCATION_KEY))

export function usePreserveLocation() {
  const location = useLocation()

  useEffect(() => {
    // only save location for these routes
    const match = matchRoutes(
      [
        { path: Routes.ROOT },
        { path: Routes.PROVIDER },
        { path: Routes.PROVIDERS },
        { path: Routes.WORKSPACES },
        { path: Routes.PRO_INSTANCE },
      ],
      location
    )
    if (match == null) {
      return
    }

    try {
      store.set(CURRENT_LOCATION_KEY, location)
    } catch (err) {
      client.log("error", `Failed to serialize location: ${err}`)
    }
  }, [location])
}

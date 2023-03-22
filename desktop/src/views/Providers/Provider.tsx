import { useMemo } from "react"
import { useParams } from "react-router"
import { Routes } from "../../routes"

export function Provider() {
  const params = useParams()
  const providerID = useMemo(() => Routes.getProviderId(params), [params])

  return <>Provider :) {providerID}</>
}

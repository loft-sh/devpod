import { useMemo } from "react"
import { useParams } from "react-router-dom"

export function useProHost() {
  const { host: urlHost } = useParams<{ host: string | undefined }>()

  const host = useMemo(() => {
    return urlHost?.replaceAll("-", ".")
  }, [urlHost])

  return host
}

import { useParams } from "react-router-dom"

export function useProHost() {
  const { host } = useParams<{ host: string | undefined }>()

  return host
}

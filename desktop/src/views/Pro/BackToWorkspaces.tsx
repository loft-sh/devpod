import { useProContext } from "@/contexts"
import { Routes } from "@/routes"
import { ChevronLeftIcon } from "@chakra-ui/icons"
import { Link } from "@chakra-ui/react"
import { Link as RouterLink } from "react-router-dom"

export function BackToWorkspaces() {
  const { host } = useProContext()

  return (
    <Link as={RouterLink} color="gray.600" to={Routes.toProInstance(host)}>
      <ChevronLeftIcon boxSize={5} /> Back to Workspaces
    </Link>
  )
}

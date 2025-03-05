import { useProContext } from "@/contexts"
import { Routes } from "@/routes"
import { ChevronLeftIcon } from "@chakra-ui/icons"
import { Link, useColorModeValue } from "@chakra-ui/react"
import { Link as RouterLink } from "react-router-dom"

export function BackToWorkspaces() {
  const { host } = useProContext()
  const color = useColorModeValue("gray.600", "gray.400")

  return (
    <Link as={RouterLink} color={color} to={Routes.toProInstance(host)}>
      <ChevronLeftIcon boxSize={5} /> Back to Workspaces
    </Link>
  )
}

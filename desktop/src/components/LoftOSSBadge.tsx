import { Link, Text } from "@chakra-ui/react"
import { client } from "../client"
import { Loft } from "../icons"

export function LoftOSSBadge() {
  return (
    <Link
      display="flex"
      alignItems="center"
      justifyContent="start"
      onClick={() => client.open("https://loft.sh/")}>
      <Text fontSize="sm" variant="muted" marginRight="2">
        Open sourced by
      </Text>
      <Loft width="10" height="6" />
    </Link>
  )
}

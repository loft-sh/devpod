import { Link, Text, useColorModeValue } from "@chakra-ui/react"
import { client } from "../client"
import { Loft } from "../icons"

export function LoftOSSBadge() {
  const textColor = useColorModeValue("gray.500", "gray.400")

  return (
    <Link
      display="flex"
      alignItems="center"
      justifyContent="start"
      onClick={() => client.open("https://loft.sh/")}>
      <Text fontSize="sm" color={textColor} marginRight="2">
        Open sourced by
      </Text>
      <Loft width="10" height="6" />
    </Link>
  )
}

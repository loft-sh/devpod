import { radioAnatomy } from "@chakra-ui/anatomy"
import { createMultiStyleConfigHelpers } from "@chakra-ui/react"

const { defineMultiStyleConfig } = createMultiStyleConfigHelpers(radioAnatomy.keys)

export const Radio = defineMultiStyleConfig({
  defaultProps: {
    colorScheme: "primary",
  },
})

import { menuAnatomy } from "@chakra-ui/anatomy"
import { createMultiStyleConfigHelpers } from "@chakra-ui/react"

const { definePartsStyle, defineMultiStyleConfig } = createMultiStyleConfigHelpers(menuAnatomy.keys)
export const Menu = defineMultiStyleConfig({
  baseStyle: definePartsStyle({
    item: {
      fontSize: "sm",
    },
  }),
})

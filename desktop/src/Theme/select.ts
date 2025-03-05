import { selectAnatomy } from "@chakra-ui/anatomy"
import { createMultiStyleConfigHelpers } from "@chakra-ui/react"

const { definePartsStyle, defineMultiStyleConfig } = createMultiStyleConfigHelpers(
  selectAnatomy.keys
)
export const Select = defineMultiStyleConfig({
  baseStyle: definePartsStyle({
    addon: {
      borderColor: "gray.200",
      _dark: {
        borderColor: "gray.700",
      },
    },
    field: {
      borderColor: "gray.200",
      _dark: {
        borderColor: "gray.700",
      },
    },
  }),
})

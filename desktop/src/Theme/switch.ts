import { switchAnatomy } from "@chakra-ui/anatomy"
import { createMultiStyleConfigHelpers } from "@chakra-ui/react"

const { definePartsStyle, defineMultiStyleConfig } = createMultiStyleConfigHelpers(
  switchAnatomy.keys
)
export const Switch = defineMultiStyleConfig({
  baseStyle: definePartsStyle(({ theme }) => {
    const from = theme.colors.primary["400"]
    const to = theme.colors.primary["800"]

    return {
      track: {
        _checked: {
          background: `linear-gradient(90deg, ${from} 30%, ${to})`,
        },
      },
    }
  }),
})

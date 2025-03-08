import { modalAnatomy } from "@chakra-ui/anatomy"
import { createMultiStyleConfigHelpers } from "@chakra-ui/react"
import { mode } from "@chakra-ui/theme-tools"

const { definePartsStyle, defineMultiStyleConfig } = createMultiStyleConfigHelpers(
  modalAnatomy.keys
)
export const Modal = defineMultiStyleConfig({
  baseStyle: definePartsStyle((props) => {
    return {
      body: {
        bg: mode("white", "gray.900")(props),
      },
      dialog: {
        bg: mode("white", "gray.900")(props),
      },
    }
  }),
})

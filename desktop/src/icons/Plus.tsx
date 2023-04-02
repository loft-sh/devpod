import { createIcon } from "@chakra-ui/react"
import { defaultProps } from "./defaultProps"

export const Plus = createIcon({
  displayName: "Plus",
  viewBox: "0 0 20 20",
  defaultProps,
  path: (
    <path d="M10.75 6.75a.75.75 0 00-1.5 0v2.5h-2.5a.75.75 0 000 1.5h2.5v2.5a.75.75 0 001.5 0v-2.5h2.5a.75.75 0 000-1.5h-2.5v-2.5z" />
  ),
})

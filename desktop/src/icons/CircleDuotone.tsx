import { createIcon } from "@chakra-ui/react"
import { defaultProps } from "./defaultProps"

export const CircleDuotone = createIcon({
  displayName: "CircleDuotone",
  viewBox: "0 0 16 16",
  defaultProps,
  path: (
    <circle
      fill="currentColor"
      fillOpacity={0.4}
      stroke="currentColor"
      cx="8"
      cy="8"
      r="6"
      strokeWidth="1.5"
    />
  ),
})

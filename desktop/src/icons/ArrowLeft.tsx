import { createIcon } from "@chakra-ui/react"
import { defaultProps } from "./defaultProps"

export const ArrowLeft = createIcon({
  displayName: "ArrowLeft",
  viewBox: "0 0 20 20",
  defaultProps,
  path: (
    <path
      fillRule="evenodd"
      d="M15 10a.75.75 0 01-.75.75H7.612l2.158 1.96a.75.75 0 11-1.04 1.08l-3.5-3.25a.75.75 0 010-1.08l3.5-3.25a.75.75 0 111.04 1.08L7.612 9.25h6.638A.75.75 0 0115 10z"
      clipRule="evenodd"
    />
  ),
})

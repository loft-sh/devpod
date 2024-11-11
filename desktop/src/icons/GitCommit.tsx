import { createIcon } from "@chakra-ui/react"
import { defaultProps } from "./defaultProps"

export const GitCommit = createIcon({
  displayName: "GitCommit",
  viewBox: "0 0 16 16",
  defaultProps,
  path: [
    <path key="dot" fill="currentColor" d="M1.333 7.333h13.333v1.333H1.333z" />,
    <path
      key="stroke"
      fill="#fff"
      stroke="currentColor"
      strokeWidth="1.4"
      d="M9.967 8a1.967 1.967 0 1 1-3.934 0 1.967 1.967 0 0 1 3.934 0Z"
    />,
  ],
})

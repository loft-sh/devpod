import { createIcon } from "@chakra-ui/react"
import { defaultProps } from "./defaultProps"

export const WorkspaceStatus = createIcon({
  displayName: "WorkspaceStatus",
  viewBox: "0 0 12 12",
  defaultProps,
  path: (
    <path
      fill="currentColor"
      d="M6 0C2.69 0 0 2.69 0 6s2.69 6 6 6 6-2.69 6-6-2.69-6-6-6Zm3.36 6.568h-.91l-.884 1.769a.612.612 0 0 1-.518.316.563.563 0 0 1-.505-.316L4.952 5.192l-.53 1.06a.577.577 0 0 1-.506.316H2.653A.566.566 0 0 1 2.084 6c0-.316.253-.568.569-.568h.91l.883-1.769c.19-.379.821-.379 1.01 0l1.58 3.145.543-1.073a.577.577 0 0 1 .505-.316h1.263c.316 0 .569.253.569.568 0 .316-.24.581-.556.581Z"
    />
  ),
})

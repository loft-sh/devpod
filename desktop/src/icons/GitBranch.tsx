import { createIcon } from "@chakra-ui/react"
import { defaultProps } from "./defaultProps"

export const GitBranch = createIcon({
  displayName: "GitBranch",
  viewBox: "0 0 12 14",
  defaultProps,
  path: (
    <path d="M9.563 1.516a1.751 1.751 0 0 0-.524 3.42v1.498L3 8.412V3.97A1.751 1.751 0 1 0 .687 2.312c0 .77.497 1.422 1.188 1.658v6.061a1.751 1.751 0 1 0 2.313 1.658c0-.769-.497-1.422-1.188-1.658v-.434L9.617 7.43a.79.79 0 0 0 .545-.753V4.909c.67-.247 1.15-.89 1.15-1.643 0-.966-.784-1.75-1.75-1.75Zm-7.876.796a.75.75 0 0 1 1.5 0 .75.75 0 0 1-1.5 0Zm1.5 9.376a.75.75 0 0 1-1.5 0 .75.75 0 0 1 1.5 0Zm6.376-7.672a.75.75 0 0 1 0-1.5.75.75 0 0 1 0 1.5Z" />
  ),
})

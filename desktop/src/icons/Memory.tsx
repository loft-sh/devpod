import { createIcon } from "@chakra-ui/react"
import { defaultProps } from "./defaultProps"

export const Memory = createIcon({
  displayName: "Memory",
  viewBox: "0 0 24 24",
  defaultProps,
  path: [
    <path
      key="1"
      fillRule="evenodd"
      clipRule="evenodd"
      d="M17.5 2.50012H6.5V21.5001H17.5V2.50012ZM6.5 1.00012C5.67157 1.00012 5 1.67169 5 2.50012V21.5001C5 22.3285 5.67157 23.0001 6.5 23.0001H17.5C18.3284 23.0001 19 22.3285 19 21.5001V2.50012C19 1.67169 18.3284 1.00012 17.5 1.00012H6.5Z"
    />,
    <path key="2" d="M1 5.00012H5V7.00012H1V5.00012Z" />,
    <path key="3" d="M19 5.00012H23V7.00012H19V5.00012Z" />,
    <path key="4" d="M1 9.00012H5V11.0001H1V9.00012Z" />,
    <path key="5" d="M19 9.00012H23V11.0001H19V9.00012Z" />,
    <path key="6" d="M1 13.0001H5V15.0001H1V13.0001Z" />,
    <path key="7" d="M19 13.0001H23V15.0001H19V13.0001Z" />,
    <path key="8" d="M1 17.0001H5V19.0001H1V17.0001Z" />,
    <path key="9" d="M19 17.0001H23V19.0001H19V17.0001Z" />,
  ],
})

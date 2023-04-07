import { Heading, HStack } from "@chakra-ui/react"
import { ReactNode } from "react"
import { exists } from "../../lib"
import { ToolbarTitle } from "./Toolbar"
import { TViewTitle } from "./types"

type TNavigationViewLayoutProps = Readonly<{ title: TViewTitle | null; children?: ReactNode }>
export function NavigationViewLayout({ title, children }: TNavigationViewLayoutProps) {
  return (
    <>
      {exists(title) && (
        <ToolbarTitle>
          <HStack align="center" width="full" overflow="hidden">
            {exists(title.leadingAction) && title.leadingAction}
            <Heading
              overflow="hidden"
              whiteSpace={"nowrap"}
              textOverflow="ellipsis"
              as={title.priority === "high" ? "h1" : "h2"}
              size="sm">
              {title.label}
            </Heading>
            {exists(title.trailingAction) && title.trailingAction}
          </HStack>
        </ToolbarTitle>
      )}
      {children}
    </>
  )
}

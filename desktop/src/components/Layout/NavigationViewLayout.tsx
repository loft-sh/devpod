import { Heading, HStack } from "@chakra-ui/react"
import { ReactNode } from "react"
import { exists } from "../../lib"
import { TViewTitle } from "./types"

type TNavigationViewLayoutProps = Readonly<{ title: TViewTitle | null; children?: ReactNode }>
export function NavigationViewLayout({ title, children }: TNavigationViewLayoutProps) {
  return (
    <>
      {exists(title) && (
        <HStack align="center">
          {exists(title.leadingAction) && title.leadingAction}
          <Heading as={title.priority === "high" ? "h1" : "h2"} size="xl">
            {title.label}
          </Heading>
          {exists(title.trailingAction) && title.trailingAction}
        </HStack>
      )}
      {children}
    </>
  )
}

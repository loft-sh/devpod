import { Box, Heading, Link, ListItem, UnorderedList } from "@chakra-ui/react"
import Markdown from "markdown-to-jsx"
import { client } from "../client"

export type TLinkClickEvent = React.MouseEvent<HTMLLinkElement> & { target: HTMLLinkElement }
export type TChangeLogProps = Readonly<{ rawMarkdown: string }>

export function Changelog({ rawMarkdown }: TChangeLogProps) {
  return (
    <Box paddingX="6" paddingY="2" marginBottom="4">
      <Markdown
        options={{
          overrides: {
            h2: {
              component: Heading,
              props: {
                size: "md",
                marginBottom: "2",
                marginTop: "4",
              },
            },
            h3: {
              component: Heading,
              props: {
                size: "sm",
                marginBottom: "2",
                marginTop: "4",
              },
            },
            a: {
              component: Link,
              props: {
                onClick: (e: TLinkClickEvent) => {
                  e.preventDefault()
                  client.open(e.target.href)
                },
              },
            },
            ul: {
              component: UnorderedList,
            },
            li: {
              component: ListItem,
            },
          },
        }}>
        {rawMarkdown}
      </Markdown>
    </Box>
  )
}

import {
  Box,
  Button,
  Heading,
  Link,
  ListItem,
  Modal,
  ModalBody,
  ModalCloseButton,
  ModalContent,
  ModalFooter,
  ModalHeader,
  ModalOverlay,
  Text,
  UnorderedList,
  useDisclosure,
} from "@chakra-ui/react"
import Markdown from "markdown-to-jsx"
import { useEffect, useMemo, useState } from "react"
import { client } from "../client"
import { Release } from "../gen"
import { useReleases, useVersion } from "../lib"

const LAST_INSTALLED_VERSION_KEY = "devpod-last-installed-version"
type TLinkClickEvent = React.MouseEvent<HTMLLinkElement> & { target: HTMLLinkElement }

export function useChangelogModal(isReady: boolean) {
  const currentVersion = useVersion()
  const releases = useReleases()
  const { isOpen, onClose, onOpen } = useDisclosure()
  const [latestRelease, setLatestRelease] = useState<Release | null>(null)
  const modal = useMemo(
    () =>
      latestRelease !== null ? (
        <Modal onClose={onClose} isOpen={isOpen} scrollBehavior="inside" size="3xl" isCentered>
          <ModalOverlay />
          <ModalContent>
            <ModalCloseButton />
            <ModalHeader>Changelog</ModalHeader>
            <ModalBody>
              {latestRelease.body ? (
                <Changelog rawMarkdown={latestRelease.body} />
              ) : (
                <Text>This release doesn&apos;t have a changelog</Text>
              )}
            </ModalBody>
            <ModalFooter>
              <Button onClick={onClose}>Done</Button>
            </ModalFooter>
          </ModalContent>
        </Modal>
      ) : null,
    [isOpen, latestRelease, onClose]
  )

  useEffect(() => {
    if (!isReady || !currentVersion || !releases) {
      return
    }

    const latestVersion = localStorage.getItem(LAST_INSTALLED_VERSION_KEY)
    const maybeRelease = releases.find((r) => r.tag_name === `v${currentVersion}`)

    if (latestVersion !== currentVersion) {
      localStorage.setItem(LAST_INSTALLED_VERSION_KEY, currentVersion)

      if (maybeRelease !== undefined && !maybeRelease.name?.endsWith("[skip changelog]")) {
        setLatestRelease(maybeRelease)
        onOpen()
      }
    }
  }, [currentVersion, isReady, onOpen, releases])

  return { modal }
}

type TChangeLogProps = Readonly<{ rawMarkdown: string }>
function Changelog({ rawMarkdown }: TChangeLogProps) {
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

import {
  Button,
  Modal,
  ModalBody,
  ModalCloseButton,
  ModalContent,
  ModalFooter,
  ModalHeader,
  ModalOverlay,
  Text,
  useDisclosure,
} from "@chakra-ui/react"
import { useEffect, useMemo, useState } from "react"
import { Release } from "../gen"
import { useReleases, useVersion } from "../lib"
import { Changelog } from "./Changelog"

const LAST_INSTALLED_VERSION_KEY = "devpod-last-installed-version"

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

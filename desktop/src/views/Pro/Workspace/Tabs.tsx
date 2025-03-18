import { TProInstanceDetail } from "@/lib"
import { ComponentType, useMemo } from "react"
import { TTabProps } from "./types"
import { Logs } from "./Logs"
import { Configuration } from "./Configuration"
import { Tab, TabList, TabPanel, TabPanels, Tabs, useColorModeValue } from "@chakra-ui/react"
import { useSearchParams } from "react-router-dom"
import { Routes } from "@/routes"

const DETAILS_TABS: Readonly<{
  key: TProInstanceDetail
  label: string
  component: ComponentType<TTabProps>
}>[] = [
  { key: "logs", label: "Logs", component: Logs },
  { key: "configuration", label: "Configuration", component: Configuration },
]
type TWorkspaceTabProps = Readonly<{}> & TTabProps
export function WorkspaceTabs({ ...tabProps }: TWorkspaceTabProps) {
  const headerBackgroundColor = useColorModeValue("white", "background.darkest")
  const contentBackgroundColor = useColorModeValue("gray.50", "background.darkest")
  const [searchParams, setSearchParams] = useSearchParams()

  const tabIndex = useMemo(() => {
    const currentTab = Routes.getProWorkspaceDetailsParams(searchParams).tab

    const idx = DETAILS_TABS.findIndex((v) => v.key === currentTab)
    if (idx === -1) {
      return 0
    }

    return idx
  }, [searchParams])

  const handleTabIndexChanged = (newIndex: number) => {
    const key = DETAILS_TABS[newIndex]?.key
    if (!key) return
    setSearchParams((prev) => {
      prev.set("tab", key)

      return prev
    })
  }

  return (
    <Tabs
      colorScheme="gray"
      isLazy
      w="full"
      h="full"
      index={tabIndex}
      onChange={handleTabIndexChanged}>
      <TabList ml="-8" px="8" mb="0" bgColor={headerBackgroundColor}>
        {DETAILS_TABS.map(({ key, label }) => (
          <Tab fontWeight="semibold" key={key}>
            {label}
          </Tab>
        ))}
      </TabList>
      <TabPanels h="99%">
        {DETAILS_TABS.map(({ label, component: Component }) => (
          <TabPanel
            bgColor={contentBackgroundColor}
            minH={label === "Configuration" ? "" : "full"}
            width="100vw"
            ml="-8"
            px="12"
            pt="8"
            pb="0"
            key={label}>
            <Component {...tabProps} />
          </TabPanel>
        ))}
      </TabPanels>
    </Tabs>
  )
}

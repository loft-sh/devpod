import { Tab, TabList, Tabs } from "@chakra-ui/react"
import { TWorkspaceSourceType } from "@/types"

const SOURCE_TYPE_MAP = {
  0: "local",
  1: "git",
  2: "image",
  local: 0,
  git: 1,
  image: 2,
}

type TSourceTypeTabsProps = Readonly<{
  sourceType: TWorkspaceSourceType
  onSourceTypeChanged: (type: TWorkspaceSourceType) => void
}>
export function SourceTypeTabs({ sourceType, onSourceTypeChanged }: TSourceTypeTabsProps) {
  const typeTabIndex = SOURCE_TYPE_MAP[sourceType]
  const handleSourceTypeChanged = (index: number) => {
    onSourceTypeChanged(SOURCE_TYPE_MAP[index as 0 | 1 | 2] as TWorkspaceSourceType)
  }

  return (
    <Tabs width="90%" variant="muted" index={typeTabIndex} onChange={handleSourceTypeChanged}>
      <TabList>
        <Tab>Folder</Tab>
        <Tab>Git Repo</Tab>
        <Tab>Image</Tab>
      </TabList>
    </Tabs>
  )
}

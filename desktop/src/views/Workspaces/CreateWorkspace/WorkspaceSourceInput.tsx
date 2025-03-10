import { ChevronDownIcon } from "@chakra-ui/icons"
import {
  Button,
  Icon,
  Input,
  InputProps,
  Popover,
  PopoverArrow,
  PopoverContent,
  PopoverTrigger,
  Tab,
  TabList,
  TabPanel,
  TabPanelProps,
  TabPanels,
  Tabs,
  Tooltip,
  VStack,
  useColorModeValue,
} from "@chakra-ui/react"
import { useCallback, useMemo, useState } from "react"
import { ControllerRenderProps } from "react-hook-form"
import { FiFolder } from "react-icons/fi"
import { useBorderColor } from "../../../Theme"
import { client } from "../../../client"
import { FieldName, TFormValues } from "./types"
import { TWorkspaceSourceType } from "@/types"

// WARN: Make sure these match the regexes in /pkg/git/git.go
const GIT_REPOSITORY_PATTERN =
  "((?:(?:https?|git|ssh)://)?(?:[^@/\\n]+@)?(?:[^:/\\n]+)(?:[:/][^/\\n]+)+(?:\\.git)?)"
const GIT_REPOSITORY_REGEX = new RegExp(GIT_REPOSITORY_PATTERN)
const BRANCH_REGEX = new RegExp(`^${GIT_REPOSITORY_PATTERN}@([a-zA-Z0-9\\./\\-\\_]+)$`)
const COMMIT_REGEX = new RegExp(`^${GIT_REPOSITORY_PATTERN}@sha256:([a-zA-Z0-9]+)$`)
const PR_REGEX = new RegExp(`^${GIT_REPOSITORY_PATTERN}@pull\\/([0-9]+)\\/head$`)
const SUBPATH_REGEX = new RegExp(`^${GIT_REPOSITORY_PATTERN}@subpath:([a-zA-Z0-9\\./\\-\\_]+)$`)

const AdvancedGitSetting = {
  BRANCH: "branch",
  COMMIT: "commit",
  PR: "pr",
  SUBPATH: "subpath",
} as const
const ADVANCED_GIT_SETTING_TABS = [
  AdvancedGitSetting.BRANCH,
  AdvancedGitSetting.COMMIT,
  AdvancedGitSetting.PR,
  AdvancedGitSetting.SUBPATH,
]
type TAdvancedGitSetting = (typeof AdvancedGitSetting)[keyof typeof AdvancedGitSetting]
const INITIAL_ADVANCED_SETTINGS = { option: AdvancedGitSetting.BRANCH, value: "" }

const SOURCE_TYPE_MAP = {
  0: "local",
  1: "git",
  2: "image",
  local: 0,
  git: 1,
  image: 2,
}
type TWorkspaceSourceInputProps = Readonly<{
  field: ControllerRenderProps<TFormValues, (typeof FieldName)["SOURCE"]>
  sourceType: TWorkspaceSourceType
  onSourceTypeChanged: (type: TWorkspaceSourceType) => void
}>
export function WorkspaceSourceInput({
  field,
  sourceType,
  onSourceTypeChanged,
}: TWorkspaceSourceInputProps) {
  const inputBackgroundColor = useColorModeValue("white", "background.darkest")
  const borderColor = useBorderColor()
  const [tabIndex, setTabIndex] = useState(0)
  const typeTabIndex = SOURCE_TYPE_MAP[sourceType]
  const handleSourceTypeChanged = (index: number) => {
    onSourceTypeChanged(SOURCE_TYPE_MAP[index as 0 | 1 | 2] as TWorkspaceSourceType)
  }
  const [advancedGitSettings, setAdvancedGitSettings] =
    useState<Readonly<{ option: TAdvancedGitSetting | null; value: string }>>(
      INITIAL_ADVANCED_SETTINGS
    )
  const updateAdvancedGitSettings = <TKey extends keyof typeof advancedGitSettings>(
    key: TKey,
    val: (typeof advancedGitSettings)[TKey]
  ) => {
    setAdvancedGitSettings((prev) => {
      const newSettings = { ...prev, [key]: val }
      if (newSettings.option === prev.option && newSettings.value === prev.value) {
        return prev
      }

      // NOTE: field.value can be undefined
      // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
      if (newSettings.value !== "" && newSettings.option !== null && field.value !== undefined) {
        // update source
        field.onChange(appendGitSetting(field.value, newSettings.option, newSettings.value))
        // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
      } else if (newSettings.value === "" && field.value !== undefined) {
        field.onChange(pruneGitSettings(field.value))
      }

      return newSettings
    })
  }

  const handleSelectFolderClicked = useCallback(async () => {
    const selected = await client.selectFromDir()
    if (typeof selected === "string") {
      field.onChange(selected)
    }
  }, [field])

  const handleAdvancedOptionTabChanged = (index: number) => {
    const option = ADVANCED_GIT_SETTING_TABS[index]
    updateAdvancedGitSettings("option", option as TAdvancedGitSetting)
    setTabIndex(index)
  }

  const handlePopoverOpened = () => {
    // synchronize field value with internal state
    const settings = parseAdvancedSettings(field.value)
    if (settings.option === null) {
      // set default
      settings.option = INITIAL_ADVANCED_SETTINGS.option
    }

    setAdvancedGitSettings((prev) => {
      if (settings.option === prev.option && settings.value === prev.value) {
        return prev
      }

      return settings
    })
    setTabIndex(ADVANCED_GIT_SETTING_TABS.indexOf(settings.option))
  }

  const inputCommonProps = useMemo<InputProps>(() => {
    return {
      spellCheck: false,
      backgroundColor: inputBackgroundColor,
      fontSize: "md",
      height: "12",
      type: "text",
      value: field.value,
      onChange: field.onChange,
      width: "full",
    }
  }, [field.onChange, field.value, inputBackgroundColor])

  const tabPanelProps = useMemo<TabPanelProps>(() => {
    return {
      px: "0",
      display: "flex",
      flexDir: "row",
      flexWrap: "nowrap",
      alignItems: "center",
      width: "full",
    }
  }, [])

  const isGitRepoValid = GIT_REPOSITORY_REGEX.test(field.value)
  const allowAdvancedInput = field.value && isGitRepoValid
  const allowPullRequest =
    advancedGitSettings.value.length === 0 || !isNaN(parseInt(advancedGitSettings.value))

  return (
    <Tabs width="90%" variant="muted" index={typeTabIndex} onChange={handleSourceTypeChanged}>
      <TabList>
        <Tab>Folder</Tab>
        <Tab>Git Repo</Tab>
        <Tab>Image</Tab>
      </TabList>
      <TabPanels>
        <TabPanel {...tabPanelProps}>
          <Input
            {...inputCommonProps}
            borderTopRightRadius={0}
            borderBottomRightRadius={0}
            placeholder="/path/to/workspace"
          />
          <Button
            leftIcon={<Icon as={FiFolder} />}
            transform="auto"
            borderTopLeftRadius={0}
            borderBottomLeftRadius={0}
            borderTopWidth={"thin"}
            borderRightWidth={"thin"}
            borderBottomWidth={"thin"}
            minW="28"
            borderColor={borderColor}
            height={inputCommonProps.height}
            onClick={handleSelectFolderClicked}>
            Browse...
          </Button>
        </TabPanel>
        <TabPanel {...tabPanelProps}>
          <Input
            {...inputCommonProps}
            borderTopRightRadius={0}
            borderBottomRightRadius={0}
            placeholder="github.com/loft-sh/devpod-example-go"
          />
          <Popover isLazy onOpen={handlePopoverOpened}>
            <PopoverTrigger>
              <Button
                leftIcon={<ChevronDownIcon boxSize={5} />}
                transform="auto"
                borderTopLeftRadius={0}
                borderBottomLeftRadius={0}
                borderTopWidth={"thin"}
                borderRightWidth={"thin"}
                borderBottomWidth={"thin"}
                borderColor={borderColor}
                minW="32"
                height={inputCommonProps.height}>
                Advanced...
              </Button>
            </PopoverTrigger>
            <PopoverContent width="auto" padding="4">
              <PopoverArrow />
              <VStack>
                <Tabs
                  variant="muted-popover"
                  size="sm"
                  index={tabIndex}
                  onChange={handleAdvancedOptionTabChanged}>
                  <TabList>
                    <Tab>Branch</Tab>
                    <Tab>Commit</Tab>
                    <Tab isDisabled={!allowPullRequest}>
                      <Tooltip
                        label={!allowPullRequest ? "Pull request reference must be a number" : ""}>
                        Pull Request
                      </Tooltip>
                    </Tab>
                    <Tab>Sub Folder</Tab>
                  </TabList>
                  <TabPanels paddingTop="2">
                    <Tooltip
                      placement="top-start"
                      label={
                        !allowAdvancedInput
                          ? // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
                            `Git repository "${field.value ?? ""}" is empty or invalid`
                          : ""
                      }>
                      <Input
                        value={advancedGitSettings.value}
                        isDisabled={!allowAdvancedInput}
                        onChange={(e) => updateAdvancedGitSettings("value", e.target.value)}
                        placeholder={getAdvancedSettingsPlaceholder(
                          ADVANCED_GIT_SETTING_TABS[tabIndex]
                        )}
                      />
                    </Tooltip>
                  </TabPanels>
                </Tabs>
              </VStack>
            </PopoverContent>
          </Popover>
        </TabPanel>
        <TabPanel {...tabPanelProps}>
          <Input {...inputCommonProps} width="full" placeholder="alpine" />
        </TabPanel>
      </TabPanels>
    </Tabs>
  )
}

function appendGitSetting(
  currentValue: string,
  setting: TAdvancedGitSetting,
  settingValue: string
): string {
  currentValue = pruneGitSettings(currentValue)

  switch (setting) {
    case AdvancedGitSetting.BRANCH:
      return `${currentValue}@${settingValue}`
    case AdvancedGitSetting.COMMIT:
      return `${currentValue}@sha256:${settingValue}`
    case AdvancedGitSetting.PR:
      if (isNaN(parseInt(settingValue, 10))) {
        return currentValue
      }

      return `${currentValue}@pull/${settingValue}/head`
    case AdvancedGitSetting.SUBPATH:
      return `${currentValue}@subpath:${settingValue}`
  }
}

function pruneGitSettings(value: string): string {
  return value
    .replace(/\/$/, "")
    .replace(BRANCH_REGEX, "$1")
    .replace(COMMIT_REGEX, "$1")
    .replace(PR_REGEX, "$1")
    .replace(SUBPATH_REGEX, "$1")
}

function getAdvancedSettingsPlaceholder(setting: TAdvancedGitSetting | undefined): string {
  if (setting === undefined) {
    return ""
  }

  switch (setting) {
    case AdvancedGitSetting.BRANCH:
      return "Enter git branch"
    case AdvancedGitSetting.COMMIT:
      return "Enter SHA256 hash"
    case AdvancedGitSetting.PR:
      return "Enter PR reference number"
    case AdvancedGitSetting.SUBPATH:
      return "Enter sub folder path"
  }
}

function parseAdvancedSettings(value: string | undefined): {
  option: TAdvancedGitSetting | null
  value: string
} {
  if (!value) {
    return { option: null, value: "" }
  }

  let matches = value.match(COMMIT_REGEX)
  if (matches && matches[2]) {
    return { option: AdvancedGitSetting.COMMIT, value: matches[2] }
  }
  matches = value.match(SUBPATH_REGEX)
  if (matches && matches[2]) {
    return { option: AdvancedGitSetting.SUBPATH, value: matches[2] }
  }
  matches = value.match(PR_REGEX)
  if (matches && matches[2] && !isNaN(parseInt(matches[2]))) {
    return { option: AdvancedGitSetting.PR, value: matches[2] }
  }
  matches = value.match(BRANCH_REGEX)
  if (matches && matches[2]) {
    return { option: AdvancedGitSetting.BRANCH, value: matches[2] }
  }

  // shouldn't happen
  return { value: "", option: null }
}

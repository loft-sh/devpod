import { ChevronDownIcon } from "@chakra-ui/icons"
import {
  Button,
  Icon,
  Input,
  InputGroup,
  InputLeftAddon,
  Popover,
  PopoverArrow,
  PopoverContent,
  PopoverTrigger,
  Select,
  Tab,
  TabList,
  TabPanels,
  Tabs,
  Tooltip,
  VStack,
  useColorModeValue,
  useToken,
} from "@chakra-ui/react"
import debounce from "lodash.debounce"
import { useCallback, useMemo, useState } from "react"
import { useFormContext } from "react-hook-form"
import { FiFolder } from "react-icons/fi"
import { useBorderColor } from "../../../Theme"
import { client } from "../../../client"
import { FieldName, TFormValues } from "./types"

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

type TSourceInputProps = Readonly<{ isDisabled: boolean }>
export function SourceInput({ isDisabled }: TSourceInputProps) {
  const { register, formState, watch, setValue, trigger: validate } = useFormContext<TFormValues>()
  const currentValue = watch(FieldName.SOURCE)
  const sourceType = watch(FieldName.SOURCE_TYPE, "git")
  const inputBackgroundColor = useColorModeValue("white", "black")
  const borderColor = useBorderColor()
  const errorBorderColor = useToken("colors", "red.500")
  const [tabIndex, setTabIndex] = useState(0)
  const [advancedGitSettings, setAdvancedGitSettings] =
    useState<Readonly<{ option: TAdvancedGitSetting | null; value: string }>>(
      INITIAL_ADVANCED_SETTINGS
    )

  const updateAdvancedGitSettings = useCallback(
    <TKey extends keyof typeof advancedGitSettings>(
      key: TKey,
      val: (typeof advancedGitSettings)[TKey]
    ) => {
      setAdvancedGitSettings((prev) => {
        const newSettings = { ...prev, [key]: val }
        if (newSettings.option === prev.option && newSettings.value === prev.value) {
          return prev
        }

        // NOTE: currentValue can be undefined
        // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
        if (newSettings.value !== "" && newSettings.option !== null && currentValue !== undefined) {
          // update source
          setValue(
            FieldName.SOURCE,
            appendGitSetting(currentValue, newSettings.option, newSettings.value),
            { shouldDirty: true, shouldTouch: true, shouldValidate: true }
          )
          // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
        } else if (newSettings.value === "" && currentValue !== undefined) {
          setValue(FieldName.SOURCE, pruneGitSettings(currentValue), {
            shouldDirty: true,
            shouldTouch: true,
            shouldValidate: true,
          })
        }

        return newSettings
      })
    },
    [currentValue, setValue]
  )

  const handleSelectFolderClicked = useCallback(async () => {
    const selected = await client.selectFromDir()
    if (typeof selected === "string") {
      setValue(FieldName.SOURCE, selected, {
        shouldDirty: true,
        shouldValidate: true,
        shouldTouch: true,
      })
    }
  }, [setValue])

  const handleAdvancedOptionTabChanged = useCallback(
    (index: number) => {
      const option = ADVANCED_GIT_SETTING_TABS[index]
      updateAdvancedGitSettings("option", option as TAdvancedGitSetting)
      setTabIndex(index)
    },
    [updateAdvancedGitSettings]
  )

  const handlePopoverOpened = useCallback(() => {
    // synchronize currentValue with internal state
    const settings = parseAdvancedSettings(currentValue)
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
  }, [currentValue])

  const hasErrors = formState.errors[FieldName.SOURCE]
  const allowPullRequest =
    advancedGitSettings.value.length === 0 || !isNaN(parseInt(advancedGitSettings.value))

  const { placeholder, secondaryAction } = useMemo(() => {
    if (sourceType === "local") {
      return {
        placeholder: "/path/to/workspace",
        secondaryAction: (
          <Button
            isDisabled={isDisabled}
            aria-invalid={hasErrors ? "true" : undefined}
            _invalid={{
              borderStyle: "solid",
              borderWidth: "1px",
              borderLeftWidth: 0,
              borderColor: errorBorderColor,
            }}
            leftIcon={<Icon as={FiFolder} />}
            transform="auto"
            borderTopLeftRadius={0}
            borderBottomLeftRadius={0}
            borderTopWidth={"thin"}
            borderRightWidth={"thin"}
            borderBottomWidth={"thin"}
            minW="28"
            borderColor={borderColor}
            height="10"
            onClick={handleSelectFolderClicked}>
            Browse...
          </Button>
        ),
      }
    }
    if (sourceType === "image") {
      return { placeholder: "alpine" }
    }

    return {
      placeholder: "github.com/loft-sh/devpod-example-go",
      secondaryAction: (
        <Popover isLazy onOpen={handlePopoverOpened}>
          <PopoverTrigger>
            <Button
              isDisabled={isDisabled}
              aria-invalid={hasErrors ? "true" : undefined}
              _invalid={{
                borderStyle: "solid",
                borderWidth: "1px",
                borderLeftWidth: 0,
                borderColor: errorBorderColor,
              }}
              leftIcon={<ChevronDownIcon boxSize={5} />}
              transform="auto"
              borderTopLeftRadius={0}
              borderBottomLeftRadius={0}
              borderTopWidth={"thin"}
              borderRightWidth={"thin"}
              borderBottomWidth={"thin"}
              borderColor={borderColor}
              minW="32"
              height="10">
              Advanced...
            </Button>
          </PopoverTrigger>
          <PopoverContent width="auto" padding="4">
            <PopoverArrow />
            <VStack>
              <Tabs
                variant="muted"
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
                      hasErrors
                        ? // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
                          `Git repository "${currentValue ?? ""}" is empty or invalid`
                        : ""
                    }>
                    <Input
                      value={advancedGitSettings.value}
                      isDisabled={!!hasErrors}
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
      ),
    }
  }, [
    advancedGitSettings.value,
    allowPullRequest,
    borderColor,
    currentValue,
    errorBorderColor,
    handleAdvancedOptionTabChanged,
    handlePopoverOpened,
    handleSelectFolderClicked,
    hasErrors,
    isDisabled,
    sourceType,
    tabIndex,
    updateAdvancedGitSettings,
  ])

  return (
    <InputGroup zIndex="docked">
      <InputLeftAddon padding="0" h="10">
        <Select
          {...register(FieldName.SOURCE_TYPE, { onChange: () => validate(FieldName.SOURCE) })}
          _invalid={{
            borderStyle: "solid",
            borderWidth: "1px",
            borderRightWidth: 0,
            borderColor: errorBorderColor,
          }}
          borderTopRightRadius="0"
          borderBottomRightRadius="0"
          focusBorderColor="transparent"
          cursor="pointer"
          w="full"
          border="none">
          <option value="git">Repo</option>
          <option value="local">Local Folder</option>
          <option value="image">Image</option>
        </Select>
      </InputLeftAddon>
      <Input
        {...register(FieldName.SOURCE, {
          validate: (value, { sourceType }) => {
            return new Promise((res) => {
              debounce(() => {
                if (sourceType === "git") {
                  return res(GIT_REPOSITORY_REGEX.test(value))
                }

                return res(true)
              }, 700)()
            })
          },
        })}
        _invalid={{
          borderWidth: "1px",
          borderLeftWidth: 0,
          borderRightWidth: 0,
          borderColor: errorBorderColor,
        }}
        spellCheck={false}
        backgroundColor={inputBackgroundColor}
        fontSize="md"
        height="10"
        type="text"
        w="full"
        placeholder={placeholder}
        borderTopRightRadius={0}
        borderBottomRightRadius={0}
      />
      {secondaryAction}
    </InputGroup>
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

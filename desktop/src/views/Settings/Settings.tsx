import {
  Button,
  Checkbox,
  Code,
  Divider,
  FormLabel,
  Grid,
  HStack,
  Heading,
  Icon,
  Link,
  Radio,
  RadioGroup,
  Select,
  Switch,
  Tab,
  TabList,
  TabPanel,
  TabPanels,
  Tabs,
  Text,
  VStack,
  useColorMode,
} from "@chakra-ui/react"
import { compareVersions } from "compare-versions"
import { ReactNode, useEffect, useMemo, useState } from "react"
import { HiMagnifyingGlassPlus } from "react-icons/hi2"
import { client } from "../../client"
import { ToolbarTitle, useInstallCLI } from "../../components"
import { TSettings, useChangeSettings } from "../../contexts"
import {
  getIDEDisplayName,
  isMacOS,
  useArch,
  usePlatform,
  useReleases,
  useUpdate,
  useVersion,
} from "../../lib"
import { useWelcomeModal } from "../../useWelcomeModal"
import {
  useAgentURLOption,
  useDockerCredentialsForwardingOption,
  useGitCredentialsForwardingOption,
  useTelemetryOption,
} from "./useContextOptions"
import { useIDESettings } from "./useIDESettings"
import {
  useCLIFlagsOption,
  useDotfilesOption,
  useExtraEnvVarsOption,
  useProxyOptions,
  useSSHKeySignatureOption,
} from "./useSettingsOptions"

const SETTINGS_TABS = [
  { label: "General", component: <GeneralSettings /> },
  { label: "Customization", component: <CustomizationSettings /> },
  { label: "Appearance", component: <AppearanceSettings /> },
  { label: "Updates", component: <UpdateSettings /> },
  { label: "Experimental", component: <ExperimentalSettings /> },
]

export function Settings() {
  return (
    <>
      <ToolbarTitle>
        <Heading as="h3" size="sm">
          Settings
        </Heading>
      </ToolbarTitle>

      <Tabs isLazy isFitted variant="muted">
        <TabList marginBottom="6">
          {SETTINGS_TABS.map(({ label }) => (
            <Tab key={label}>{label}</Tab>
          ))}
        </TabList>
        <TabPanels>
          {SETTINGS_TABS.map(({ label, component }) => (
            <TabPanel key={label}>{component}</TabPanel>
          ))}
        </TabPanels>
      </Tabs>
    </>
  )
}

function GeneralSettings() {
  const { settings, set } = useChangeSettings()
  const { modal: welcomeModal, show: showWelcomeModal } = useWelcomeModal()
  const { input: agentURLInput, helpText: agentURLHelpText } = useAgentURLOption()
  const { input: proxyInput, helpText: proxyHelpText } = useProxyOptions()
  const { input: telemetryInput, helpText: telemetryHelpText } = useTelemetryOption()
  const {
    badge: installCLIBadge,
    button: installCLIButton,
    helpText: installCLIHelpText,
    errorMessage: installCLIErrorMessage,
  } = useInstallCLI()

  return (
    <>
      <SettingSection title="CLI" description={installCLIHelpText}>
        <HStack>
          {installCLIButton}
          {installCLIErrorMessage}
          {installCLIBadge}
        </HStack>
      </SettingSection>

      <SettingSection
        title="Debug mode"
        description={
          <>
            Run all DevPod commands with the <Code>--debug</Code> flag, making it easier to
            troubleshoot
          </>
        }>
        <Switch
          isChecked={settings.debugFlag}
          onChange={(e) => set("debugFlag", e.target.checked)}
        />
      </SettingSection>

      <SettingSection title="Logs" description={"Open the logs for DevPod Desktop"}>
        <Button variant="outline" onClick={() => client.openDir("AppLog")}>
          Open Logs
        </Button>
      </SettingSection>

      <SettingSection title="Agent URL" description={agentURLHelpText}>
        {agentURLInput}
      </SettingSection>

      <SettingSection title="Proxy Configuration" description={proxyHelpText}>
        {proxyInput}
      </SettingSection>

      <SettingSection title="Telemetry" description={telemetryHelpText}>
        {telemetryInput}
      </SettingSection>

      <SettingSection
        showDivider={false}
        title="Show Intro"
        description="Show the introduction to DevPod again">
        <Button variant="outline" onClick={() => showWelcomeModal({ cancellable: true })}>
          Open
        </Button>
      </SettingSection>

      <VStack align="start" paddingTop="16" paddingBottom="8">
        <Heading size="sm" as="h4" color="red.600">
          Danger Zone
        </Heading>
        <Button variant="outline" colorScheme="red" onClick={() => client.quit()}>
          Quit DevPod
        </Button>
      </VStack>

      {welcomeModal}
    </>
  )
}

function CustomizationSettings() {
  const { input: dotfilesInput } = useDotfilesOption()
  const { input: gitSSHSignatureInput } = useSSHKeySignatureOption()
  const { settings, set } = useChangeSettings()
  const { ides, defaultIDE, updateDefaultIDE } = useIDESettings()
  const { input: dockerCredentialForwardingInput, helpText: dockerCredentialForwardingHelpText } =
    useDockerCredentialsForwardingOption()
  const { input: gitCredentialForwardingInput, helpText: gitCredentialForwardingHelpText } =
    useGitCredentialsForwardingOption()

  return (
    <>
      <SettingSection
        title="IDE"
        description="Select the default IDE you're using for workspaces. This will be overridden whenever you create a workspace with a different IDE. You can prevent this by checking the 'Always use this IDE' checkbox">
        <>
          <Select
            maxWidth="52"
            textTransform="capitalize"
            onChange={(e) => updateDefaultIDE({ ide: e.target.value })}
            value={defaultIDE ? defaultIDE.name! : undefined}>
            {ides?.map((ide) => (
              <option key={ide.name} value={ide.name!}>
                {getIDEDisplayName(ide)}
              </option>
            ))}
          </Select>
          <Checkbox
            isChecked={settings.fixedIDE}
            onChange={(e) => set("fixedIDE", e.target.checked)}>
            Always use this IDE
          </Checkbox>
        </>
      </SettingSection>

      <SettingSection
        title="Dotfiles"
        description="Set the dotfiles git repository to use inside workspaces">
        {dotfilesInput}
      </SettingSection>

      <SettingSection
        title="SSH Key for Git commit signing"
        description="Set path of your SSH key you want to use for signing Git commits">
        {gitSSHSignatureInput}
      </SettingSection>

      <SettingSection
        title="Docker credentials forwarding"
        description={dockerCredentialForwardingHelpText}>
        {dockerCredentialForwardingInput}
      </SettingSection>

      <SettingSection
        showDivider={false}
        title="Git credentials forwarding"
        description={gitCredentialForwardingHelpText}>
        {gitCredentialForwardingInput}
      </SettingSection>
    </>
  )
}

function AppearanceSettings() {
  const { settings, set } = useChangeSettings()

  return (
    <>
      {isMacOS && (
        <SettingSection
          title="Translucent UI"
          description="Use transparency in the sidebar and other UI elements">
          <Switch
            isChecked={settings.transparency}
            onChange={(e) => set("transparency", e.target.checked)}
          />
        </SettingSection>
      )}

      <SettingSection title="Zoom level" description="Adjust the zoom level">
        <HStack>
          <Select
            onChange={(e) => set("zoom", e.target.value as TSettings["zoom"])}
            value={settings.zoom}>
            <option value={"sm"}>Small</option>
            <option value={"md"}>Regular</option>
            <option value={"lg"}>Large</option>
            <option value={"xl"}>Extra Large</option>
          </Select>
          <Icon as={HiMagnifyingGlassPlus} boxSize="6" color="gray.600" />
        </HStack>
      </SettingSection>

      <SettingSection showDivider={false} title="Sidebar position" description="">
        <RadioGroup
          value={settings.sidebarPosition}
          onChange={(newValue: TSettings["sidebarPosition"]) => set("sidebarPosition", newValue)}>
          <HStack>
            <Radio value="left">Left</Radio>
            <Radio value="right">Right</Radio>
          </HStack>
        </RadioGroup>
      </SettingSection>
    </>
  )
}
function UpdateSettings() {
  const { settings, set } = useChangeSettings()
  const platform = usePlatform()
  const arch = useArch()
  const version = useVersion()
  const [selectedVersion, setSelectedVersion] = useState<string | undefined>(undefined)
  const { isChecking, check, isUpdateAvailable, pendingUpdate } = useUpdate()
  const releases = useReleases()
    ?.slice()
    .sort((a, b) => compareVersions(b.tag_name, a.tag_name))
  const downloadLink = useMemo(() => {
    const release = releases?.find((release) => release.tag_name === selectedVersion)
    if (!release) {
      return undefined
    }
    if (!platform || !arch) {
      return release.html_url
    }
    let p: string = platform
    if (p === "darwin") {
      p = "macos"
    }

    const r = new RegExp(`${p}_${arch}`, "i")
    const asset = release.assets.find((asset) => r.test(asset.name))

    return asset?.browser_download_url
  }, [arch, platform, releases, selectedVersion])

  useEffect(() => {
    if (version) {
      setSelectedVersion(`v${version}`)
    }
  }, [version])

  const updateAvailableMessage = useMemo<undefined | string>(() => {
    if (isUpdateAvailable === undefined && pendingUpdate === undefined) {
      return undefined
    }

    return isUpdateAvailable || pendingUpdate !== undefined
      ? "New version available"
      : "No new version"
  }, [isUpdateAvailable, pendingUpdate])

  return (
    <>
      <SettingSection
        title="Automatically keep up to date"
        description="Download and install new versions in the background">
        <Switch
          isChecked={settings.autoUpdate}
          onChange={(e) => set("autoUpdate", e.target.checked)}
        />
      </SettingSection>
      <SettingSection
        title="Versions"
        description="Manage and explore DevPod versions"
        showDivider={false}>
        <>
          <VStack align="start" width="full" marginBottom="4">
            <Text fontSize="sm">
              Current version: {version}
              <Text as="span" fontWeight="semibold">
                {updateAvailableMessage !== undefined ? ` - ${updateAvailableMessage}` : ""}
              </Text>
            </Text>
            <HStack>
              <Button variant="outline" isLoading={isChecking} onClick={() => check()}>
                Check for updates
              </Button>
            </HStack>
          </VStack>

          <Text fontSize="sm">Or download a specific version</Text>
          <VStack align="start" width="full" overflow="hidden">
            <Select
              maxWidth="52"
              onChange={(e) => setSelectedVersion(e.target.value)}
              value={selectedVersion}>
              <option value={undefined}>Select version</option>
              {releases?.map((release) => (
                <option key={release.tag_name} value={release.tag_name}>
                  {release.tag_name}
                </option>
              ))}
            </Select>
            {downloadLink && (
              <Text fontSize="sm" width="full">
                Visit{" "}
                <Link onClick={() => client.open(downloadLink)} fontSize="sm">
                  Github
                </Link>{" "}
                to download {selectedVersion}
              </Text>
            )}
          </VStack>
        </>
      </SettingSection>
    </>
  )
}

function ExperimentalSettings() {
  const { input: cliFlagsInput, helpText: cliFlagsHelpText } = useCLIFlagsOption()
  const { input: extraEnvVarsInput, helpText: extraEnvVarsHelpText } = useExtraEnvVarsOption()
  const { settings, set } = useChangeSettings()
  const { setColorMode } = useColorMode()

  return (
    <VStack align="start">
      <SettingSection
        title="Multiple devcontainer detection"
        description="Whenever new workspaces are created, check if there are multiple devcontainers in the source. This might take a while for larger repositories">
        <Switch
          id="multicontainer"
          isChecked={settings.experimental_multiDevcontainer}
          onChange={(e) => set("experimental_multiDevcontainer", e.target.checked)}
        />
      </SettingSection>

      <SettingSection
        title="Experimental IDEs"
        description=" Enable experimental IDEs. These IDEs are not officially supported by DevPod and might be unstable. We are working on making them generally available">
        <HStack>
          <Switch
            isChecked={settings.experimental_fleet}
            onChange={(e) => set("experimental_fleet", e.target.checked)}
          />
          <FormLabel marginBottom="0" whiteSpace="nowrap" fontSize="sm">
            JetBrains Fleet
          </FormLabel>
        </HStack>

        <HStack width="full" align="center">
          <Switch
            isChecked={settings.experimental_jupyterNotebooks}
            onChange={(e) => set("experimental_jupyterNotebooks", e.target.checked)}
          />
          <FormLabel marginBottom="0" whiteSpace="nowrap" fontSize="sm">
            Jupyter Notebooks
          </FormLabel>
        </HStack>

        <HStack width="full" align="center">
          <Switch
            isChecked={settings.experimental_vscodeInsiders}
            onChange={(e) => set("experimental_vscodeInsiders", e.target.checked)}
          />
          <FormLabel marginBottom="0" whiteSpace="nowrap" fontSize="sm">
            VSCode Insiders
          </FormLabel>
        </HStack>

        <HStack width="full" align="center">
          <Switch
            isChecked={settings.experimental_cursor}
            onChange={(e) => set("experimental_cursor", e.target.checked)}
          />
          <FormLabel marginBottom="0" whiteSpace="nowrap" fontSize="sm">
            Cursor
          </FormLabel>
        </HStack>

        <HStack width="full" align="center">
          <Switch
            isChecked={settings.experimental_positron}
            onChange={(e) => set("experimental_positron", e.target.checked)}
          />
          <FormLabel marginBottom="0" whiteSpace="nowrap" fontSize="sm">
            Positron
          </FormLabel>
        </HStack>

        <HStack width="full" align="center">
          <Switch
            isChecked={settings.experimental_codium}
            onChange={(e) => set("experimental_codium", e.target.checked)}
          />
          <FormLabel marginBottom="0" whiteSpace="nowrap" fontSize="sm">
            Codium
          </FormLabel>
        </HStack>

        <HStack width="full" align="center">
          <Switch
            isChecked={settings.experimental_zed}
            onChange={(e) => set("experimental_zed", e.target.checked)}
          />
          <FormLabel marginBottom="0" whiteSpace="nowrap" fontSize="sm">
            Zed
          </FormLabel>
        </HStack>

        <HStack width="full" align="center">
          <Switch
            isChecked={settings.experimental_rstudio}
            onChange={(e) => set("experimental_rstudio", e.target.checked)}
          />
          <FormLabel marginBottom="0" whiteSpace="nowrap" fontSize="sm">
            RStudio Server
          </FormLabel>
        </HStack>

        <HStack width="full" align="center">
          <Switch
            isChecked={settings.experimental_windsurf}
            onChange={(e) => set("experimental_windsurf", e.target.checked)}
          />
          <FormLabel marginBottom="0" whiteSpace="nowrap" fontSize="sm">
            Windsurf
          </FormLabel>
        </HStack>
      </SettingSection>

      <SettingSection title="Additional CLI Flags" description={cliFlagsHelpText}>
        {cliFlagsInput}
      </SettingSection>

      <SettingSection title="Additional Environment Variables" description={extraEnvVarsHelpText}>
        {extraEnvVarsInput}
      </SettingSection>

      <SettingSection title="DevPod Pro (beta)" description="Enable DevPod Pro login and creation">
        <Switch
          isChecked={settings.experimental_devPodPro}
          onChange={(e) => set("experimental_devPodPro", e.target.checked)}
        />
      </SettingSection>

      <SettingSection title="Color Mode" description="" showDivider={false}>
        <RadioGroup
          value={settings.experimental_colorMode}
          onChange={(newValue: TSettings["experimental_colorMode"]) => {
            set("experimental_colorMode", newValue)
            setColorMode(newValue)
          }}>
          <HStack>
            <Radio value="light">Light</Radio>
            <Radio value="dark">Dark</Radio>
          </HStack>
        </RadioGroup>
      </SettingSection>
    </VStack>
  )
}

type TSettingDescriptionProps = Readonly<{ children: ReactNode }>
function SettingDescription({ children }: TSettingDescriptionProps) {
  return (
    <Text color={"gray.600"} _dark={{ color: "gray.300" }} fontSize="sm">
      {children}
    </Text>
  )
}

type TSettingSectionProps = Readonly<{
  title: string
  description: ReactNode
  showDivider?: boolean
  children: ReactNode
}>
function SettingSection({
  title,
  description,
  showDivider = true,
  children,
}: TSettingSectionProps) {
  return (
    <>
      <Grid gridTemplateColumns="32rem 1fr" columnGap="20" width="full">
        <VStack align="start" gap="0">
          <Heading as="h4" size="sm" fontWeight="medium">
            {title}
          </Heading>
          <SettingDescription>{description}</SettingDescription>
        </VStack>

        <VStack align="start" paddingX="2" width="full" overflow="hidden">
          {children}
        </VStack>
      </Grid>
      {showDivider && <SettingDivider />}
    </>
  )
}

function SettingDivider() {
  return <Divider marginY="4" />
}

import { Code, Link, Switch } from "@chakra-ui/react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { useCallback, useMemo } from "react"
import { client } from "../../client"
import { QueryKeys } from "../../queryKeys"
import { TContextOptionName } from "../../types"
import { ClearableInput } from "./ClearableInput"

const DEFAULT_DEVPOD_AGENT_URL = "https://github.com/loft-sh/devpod/releases/latest/download/"

export function useContextOptions() {
  const queryClient = useQueryClient()
  const { data: options } = useQuery({
    queryKey: QueryKeys.CONTEXT_OPTIONS,
    queryFn: async () => (await client.context.listOptions()).unwrap(),
  })
  const { mutate: updateOption } = useMutation({
    mutationFn: async ({ option, value }: { option: TContextOptionName; value: string }) => {
      ;(await client.context.setOption(option, value)).unwrap()
    },
    onSettled: () => {
      queryClient.invalidateQueries(QueryKeys.CONTEXT_OPTIONS)
    },
  })

  return useMemo(
    () => ({
      options,
      updateOption,
    }),
    [options, updateOption]
  )
}

export function useAgentURLOption() {
  const { options, updateOption } = useContextOptions()

  const handleChanged = useCallback(
    (newValue: string) => {
      const value = newValue.trim()
      updateOption({ option: "AGENT_URL", value })
    },
    [updateOption]
  )

  const input = useMemo(
    () => (
      <ClearableInput
        placeholder="Override Agent URL"
        defaultValue={options?.AGENT_URL.value ?? ""}
        onChange={handleChanged}
      />
    ),
    [handleChanged, options?.AGENT_URL.value]
  )

  const helpText = useMemo(
    () => (
      <>
        Set the Agent URL. If you leave this empty, it will be pulled from{" "}
        <Code>{DEFAULT_DEVPOD_AGENT_URL}</Code>
      </>
    ),
    []
  )

  return { input, helpText }
}

export function useTelemetryOption() {
  const { options, updateOption } = useContextOptions()

  const input = useMemo(
    () => (
      <Switch
        isChecked={options?.TELEMETRY.value === "true"}
        onChange={(e) => updateOption({ option: "TELEMETRY", value: e.target.checked.toString() })}
      />
    ),
    [options?.TELEMETRY.value, updateOption]
  )

  const helpText = useMemo(
    () => (
      <>
        Telemetry plays an important role in improving DevPod for everyone.{" "}
        <strong>We never collect any actual values, only anonymized metadata!</strong> For an
        in-depth explanation, please refer to the{" "}
        <Link onClick={() => client.open("https://devpod.sh/docs/other-topics/telemetry")}>
          documentation
        </Link>
      </>
    ),
    []
  )

  return { input, helpText }
}

export function useDockerCredentialsForwardingOption() {
  const { options, updateOption } = useContextOptions()

  const input = useMemo(
    () => (
      <Switch
        isChecked={options?.SSH_INJECT_DOCKER_CREDENTIALS.value === "true"}
        onChange={(e) =>
          updateOption({
            option: "SSH_INJECT_DOCKER_CREDENTIALS",
            value: e.target.checked.toString(),
          })
        }
      />
    ),
    [options?.SSH_INJECT_DOCKER_CREDENTIALS.value, updateOption]
  )

  const helpText = useMemo(
    () => <>Enable to forward your local docker credentials to workspaces</>,
    []
  )

  return { input, helpText }
}

export function useGitCredentialsForwardingOption() {
  const { options, updateOption } = useContextOptions()

  const input = useMemo(
    () => (
      <Switch
        isChecked={options?.SSH_INJECT_GIT_CREDENTIALS.value === "true"}
        onChange={(e) =>
          updateOption({
            option: "SSH_INJECT_GIT_CREDENTIALS",
            value: e.target.checked.toString(),
          })
        }
      />
    ),
    [options?.SSH_INJECT_GIT_CREDENTIALS.value, updateOption]
  )

  const helpText = useMemo(
    () => <>Enable to forward your local HTTPS based git credentials to workspaces</>,
    []
  )

  return { input, helpText }
}

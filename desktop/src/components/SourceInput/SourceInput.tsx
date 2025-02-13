import React, { ChangeEvent, ForwardedRef, forwardRef, useCallback, useMemo } from "react"
import { extractRevisionType, extractSourceValue } from "@/components/SourceInput/url-parser"
import { ERevisionType, REVISION_TYPE_CONFIG } from "@/components/SourceInput/type"
import {
  Button,
  Icon,
  Input,
  InputGroup,
  InputLeftAddon,
  useColorModeValue,
  useToken,
} from "@chakra-ui/react"
import { RevisionPopover } from "@/components/SourceInput/RevisionPopover"
import { TWorkspaceSourceType } from "@/types"
import { ControllerFieldState, ControllerRenderProps, UseFormTrigger } from "react-hook-form"
import {
  FieldName as OSSFieldName,
  TFormValues as OSSFormValues,
} from "@/views/Workspaces/CreateWorkspace/types"
import {
  FieldName as ProFieldName,
  TFormValues as ProFormValues,
} from "@/views/Pro/CreateWorkspace/types"
import { FiFolder } from "react-icons/fi"
import { useBorderColor } from "@/Theme"
import { client } from "@/client"
import debounce from "lodash.debounce"

type TCommonSourceInputProps = {
  leftAddon?: React.ReactNode
  fieldState: ControllerFieldState
  field:
    | ControllerRenderProps<OSSFormValues, (typeof OSSFieldName)["SOURCE"]>
    | ControllerRenderProps<ProFormValues, (typeof ProFieldName)["SOURCE"]>
  trigger?: UseFormTrigger<OSSFormValues> | UseFormTrigger<ProFormValues>
  resetPreset?: VoidFunction
  height?: React.ComponentProps<typeof Input>["height"]
}

type TInternalSourceInputProps = TCommonSourceInputProps & {
  source: string
  borderColor: string
  errorBorderColor: string
  error: boolean
  onChange: (...event: any[]) => void
}

export type TSourceInputProps = TCommonSourceInputProps & {
  mode?: TWorkspaceSourceType
}

export function SourceInput({
  leftAddon,
  resetPreset,
  fieldState,
  field,
  height,
  trigger,
  mode = "git",
}: TSourceInputProps) {
  const borderColor = useBorderColor()
  const errorBorderColor = useToken("colors", "red.500")

  const debouncedValidation = useMemo(() => {
    return debounce(() => {
      trigger?.(field.name)
    }, 500)
  }, [trigger, field.name])

  const onChange = useCallback(
    (...event: any[]) => {
      field.onChange(...event)
      debouncedValidation()
    },
    [debouncedValidation, field]
  )

  const props: TInternalSourceInputProps = {
    resetPreset,
    field,
    fieldState,
    leftAddon,
    borderColor,
    errorBorderColor,
    height,
    onChange,
    source: (field.value as string | undefined) ?? "",
    error: fieldState.isDirty && fieldState.invalid,
  }

  if (mode === "image") {
    return <ImageInput ref={field.ref} {...props} />
  } else if (mode === "local") {
    return <LocalFolderInput ref={field.ref} {...props} />
  }

  return <RepositoryInput ref={field.ref} {...props} />
}

const BaseInput = forwardRef(function InnerBaseInput(
  props: React.ComponentProps<typeof Input>,
  ref: ForwardedRef<HTMLInputElement>
) {
  const inputBackgroundColor = useColorModeValue("white", "black")
  const errorBorderColor = useToken("colors", "red.500")

  const invalid: React.ComponentProps<typeof Input>["_invalid"] = useMemo(
    () => ({
      borderStyle: "solid",
      borderWidth: "1px",
      borderColor: errorBorderColor,
    }),
    [errorBorderColor]
  )

  return (
    <Input
      ref={ref}
      _invalid={invalid}
      maxLength={2048}
      type={"text"}
      width={"full"}
      spellCheck={false}
      backgroundColor={inputBackgroundColor}
      fontSize={"md"}
      {...props}
    />
  )
})

const ImageInput = forwardRef(function InnerImageInput(
  { leftAddon, field, source, error, onChange, height }: TInternalSourceInputProps,
  ref: ForwardedRef<HTMLInputElement>
) {
  return (
    <InputGroup zIndex="docked">
      {leftAddon && <InputLeftAddon p={0}>{leftAddon}</InputLeftAddon>}
      <BaseInput
        ref={ref}
        value={source}
        height={height}
        onBlur={field.onBlur}
        name={field.name}
        onChange={onChange}
        disabled={field.disabled}
        isInvalid={error}
        aria-invalid={error ? "true" : undefined}
        borderLeftRadius={leftAddon ? 0 : undefined}
        placeholder="alpine"
      />
    </InputGroup>
  )
})

const LocalFolderInput = forwardRef(function InnerLocalFolderInput(
  {
    leftAddon,
    field,
    resetPreset,
    source,
    borderColor,
    errorBorderColor,
    error,
    onChange,
    height,
  }: TInternalSourceInputProps,
  ref: ForwardedRef<HTMLInputElement>
) {
  const handleSelectFolderClicked = useCallback(async () => {
    const selected = await client.selectFromDir()
    if (typeof selected === "string") {
      onChange(selected)
      resetPreset?.()
    }
  }, [onChange, resetPreset])

  return (
    <InputGroup zIndex="docked">
      {leftAddon && <InputLeftAddon p={0}>{leftAddon}</InputLeftAddon>}
      <BaseInput
        ref={ref}
        value={source}
        onBlur={field.onBlur}
        name={field.name}
        height={height}
        onChange={onChange}
        disabled={field.disabled}
        isInvalid={error}
        aria-invalid={error ? "true" : undefined}
        borderRightRadius={0}
        borderLeftRadius={leftAddon ? 0 : undefined}
        placeholder="/path/to/workspace"
      />
      <Button
        isDisabled={field.disabled}
        aria-invalid={error ? "true" : undefined}
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
        height={height ?? "10"}
        onClick={handleSelectFolderClicked}>
        Browse...
      </Button>
    </InputGroup>
  )
})

const RepositoryInput = forwardRef(function InnerRepositoryInput(
  {
    field,
    leftAddon,
    source,
    errorBorderColor,
    onChange,
    height,
    error,
  }: TInternalSourceInputProps,
  ref: ForwardedRef<HTMLInputElement>
) {
  const applyGitURL = useCallback(
    (value: string) => {
      onChange(value ? value : undefined)
    },
    [onChange]
  )

  const changeGitURL = useCallback(
    (e: ChangeEvent<HTMLInputElement>) => {
      applyGitURL(e.target.value)
    },
    [applyGitURL]
  )

  const applyRevision = useCallback(
    (revision: string, revisionType: ERevisionType) => {
      const originalRevisionType = extractRevisionType(source) ?? ERevisionType.BRANCH
      const sourceValue = extractSourceValue(source, originalRevisionType)

      if (!revision) {
        applyGitURL(sourceValue.repository ?? "")

        return
      }

      const formattedPartial = REVISION_TYPE_CONFIG[revisionType].formatter(revision)
      applyGitURL(`${sourceValue.repository ?? ""}${formattedPartial}`)
    },
    [source, applyGitURL]
  )

  const invalid: React.ComponentProps<typeof Input>["_invalid"] = useMemo(
    () => ({
      borderStyle: "solid",
      borderWidth: "1px",
      borderRightWidth: 0,
      borderLeftWidth: leftAddon ? 0 : undefined,
      borderColor: errorBorderColor,
    }),
    [errorBorderColor, leftAddon]
  )

  return (
    <InputGroup zIndex="docked">
      {leftAddon && <InputLeftAddon p={0}>{leftAddon}</InputLeftAddon>}
      <BaseInput
        _invalid={invalid}
        ref={ref}
        value={source}
        onBlur={field.onBlur}
        name={field.name}
        height={height}
        onChange={changeGitURL}
        disabled={field.disabled}
        isInvalid={error}
        aria-invalid={error ? "true" : undefined}
        borderRightRadius={0}
        borderLeftRadius={leftAddon ? 0 : undefined}
        placeholder="github.com/loft-sh/devpod-example-go"
      />
      <RevisionPopover
        disabled={field.disabled || error}
        source={source}
        triggerHeight={height}
        onApplyRequested={applyRevision}
      />
    </InputGroup>
  )
})

export function validateSourceInput(value: string, mode: TWorkspaceSourceType) {
  if (mode === "git") {
    const type = extractRevisionType(value) ?? ERevisionType.BRANCH
    const sourceValue = extractSourceValue(value, type)

    if (sourceValue.repositoryValid) {
      return true
    }

    if (value) {
      return "URL is malformed."
    }

    return "URL is required."
  }

  return true
}

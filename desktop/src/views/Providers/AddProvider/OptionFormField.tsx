import { TOptionWithID } from "../helpers"
import { Controller, useFormContext } from "react-hook-form"
import { ReactNode, useMemo } from "react"
import { exists } from "../../../lib"
import { AutoComplete } from "../../../components"
import {
  Checkbox,
  FormControl,
  FormErrorMessage,
  FormHelperText,
  FormLabel,
  Input,
  Select,
} from "@chakra-ui/react"

type TOptionFormField = TOptionWithID &
  Readonly<{ isRequired?: boolean; refreshSubOptions?: (id: string) => void }>
export function OptionFormField({
  id,
  defaultValue,
  value,
  password,
  description,
  type,
  displayName,
  suggestions,
  enum: enumProp,
  refreshSubOptions,
  subOptionsCommand,
  isRequired = false,
}: TOptionFormField) {
  const { register, formState } = useFormContext()
  const optionError = formState.errors[id]

  const input = useMemo<ReactNode>(() => {
    const registerProps = register(id, { required: isRequired })
    const valueProp = exists(value) ? { defaultValue: value } : {}
    const defaultValueProp = exists(defaultValue) ? { defaultValue } : {}
    const props = { ...defaultValueProp, ...valueProp, ...registerProps }
    const refresh = () => {
      refreshSubOptions?.(id)
    }

    if (exists(suggestions)) {
      return (
        <Controller
          name={id}
          defaultValue={value ?? defaultValue ?? undefined}
          rules={{ required: isRequired }}
          render={({ field: { onChange, onBlur, value: v, ref } }) => {
            return (
              <AutoComplete
                ref={ref}
                value={v || ""}
                onBlur={wrapFunction(onBlur, refresh, !!subOptionsCommand)}
                onChange={(value) => {
                  if (value) {
                    onChange(value)
                  }
                }}
                placeholder={`Enter ${displayName}`}
                options={suggestions.map((s) => ({ key: s, label: s }))}
              />
            )
          }}
        />
      )
    } else if (enumProp?.length) {
      let placeholder: string | undefined = "Select option"
      if (value) {
        placeholder = undefined
      }

      return (
        <Select
          {...props}
          onChange={wrapFunction(props.onChange, refresh, !!subOptionsCommand)}
          placeholder={placeholder}>
          {enumProp.map((val, i) => (
            <option key={i} value={val}>
              {val}
            </option>
          ))}
        </Select>
      )
    }

    switch (type) {
      case "boolean":
        return (
          <Checkbox defaultChecked={props.defaultValue === "true"} {...props}>
            {displayName}
          </Checkbox>
        )
      case "number":
        return (
          <Input
            spellCheck={false}
            placeholder={`Enter ${displayName}`}
            type="number"
            {...props}
            onBlur={wrapFunction(props.onBlur, refresh, !!subOptionsCommand)}
          />
        )
      case "duration":
        return (
          <Input
            spellCheck={false}
            placeholder={`Enter ${displayName}`}
            type="text"
            {...props}
            onBlur={wrapFunction(props.onBlur, refresh, !!subOptionsCommand)}
          />
        )
      case "string":
        return (
          <Input
            spellCheck={false}
            placeholder={`Enter ${displayName}`}
            type={password ? "password" : "text"}
            {...props}
            onBlur={wrapFunction(props.onBlur, refresh, !!subOptionsCommand)}
          />
        )
      default:
        return (
          <Input
            spellCheck={false}
            placeholder={`Enter ${displayName}`}
            type={password ? "password" : "text"}
            {...props}
            onBlur={wrapFunction(props.onBlur, refresh, !!subOptionsCommand)}
          />
        )
    }
  }, [
    register,
    id,
    isRequired,
    value,
    defaultValue,
    suggestions,
    enumProp,
    type,
    refreshSubOptions,
    subOptionsCommand,
    displayName,
    password,
  ])

  return (
    <FormControl isRequired={isRequired}>
      <FormLabel>{displayName}</FormLabel>
      {input}
      {exists(optionError) ? (
        <FormErrorMessage>{optionError.message?.toString() ?? "Error"}</FormErrorMessage>
      ) : (
        exists(description) && <FormHelperText>{description}</FormHelperText>
      )}
    </FormControl>
  )
}

function wrapFunction(
  fn: any | undefined,
  wrap: (() => void) | undefined,
  shouldWrap: boolean
): (event?: any) => void {
  return (event) => {
    if (fn) {
      fn(event)
    }

    if (shouldWrap) {
      wrap?.()
    }
  }
}

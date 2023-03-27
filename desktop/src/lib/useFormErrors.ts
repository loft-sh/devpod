import { useMemo } from "react"
import { FieldError, FieldValues, FormState } from "react-hook-form"

type TErr<TPrefix extends string> = `${TPrefix}Error`
type TFormErrors<T extends Record<string, unknown>> = {
  [K in keyof T as TErr<K extends string ? K : never>]?: FieldError
}

export function useFormErrors<TFormValues extends FieldValues>(
  fieldNames: readonly (keyof TFormValues)[],
  formState: FormState<TFormValues>
): TFormErrors<TFormValues> {
  return useMemo(() => {
    return fieldNames.reduce<TFormErrors<TFormValues>>(
      (acc, curr) => ({ ...acc, [`${String(curr)}Error`]: formState.errors[curr] }),
      {}
    )
  }, [fieldNames, formState.errors])
}

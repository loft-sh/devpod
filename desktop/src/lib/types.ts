export type TAction<
  TType extends string,
  TPayload extends unknown | undefined = undefined,
> = TPayload extends undefined
  ? Readonly<{
      type: TType
    }>
  : {
      type: TType
      payload: TPayload
    }

const PRO_INSTANCE_DETAILS = ["logs", "configuration"] as const
export type TProInstanceDetail = (typeof PRO_INSTANCE_DETAILS)[number]

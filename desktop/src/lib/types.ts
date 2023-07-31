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

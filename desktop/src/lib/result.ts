export class Err<TError extends Error> {
  readonly ok = false
  readonly err = true

  constructor(public readonly val: TError) {}

  public unwrap(): undefined {
    throw new Error(this.val.message, { cause: this.val })
  }
}

export class Ok<T> {
  readonly ok = true
  readonly err = false

  constructor(public readonly val: T) {}

  public unwrap(): T {
    return this.val
  }
}

// eslint-disable-next-line @typescript-eslint/naming-convention
export type ResultError = Ok<undefined> | Err<Failed>
// eslint-disable-next-line @typescript-eslint/naming-convention
export type Result<T> = Ok<T> | Err<Failed>

// eslint-disable-next-line @typescript-eslint/naming-convention
export type ErrorType = string

export const ErrorTypeUnknown: ErrorType = ""
export const ErrorTypeCancelled: ErrorType = "cancelled"

// eslint-disable-next-line @typescript-eslint/no-unused-vars
export const MapErrorCode = (code: number): ErrorType => {
  return ErrorTypeUnknown
}

export class Return {
  static Ok() {
    return new Ok<undefined>(undefined)
  }

  static Value<TVal>(val: TVal) {
    return new Ok<TVal>(val)
  }

  static Failed(message: string, reason: string = "", type: ErrorType = ErrorTypeUnknown) {
    return new Err<Failed>(new Failed(message, type, reason))
  }

  static Error<TError extends Error>(val: TError): Err<TError> {
    return new Err<TError>(val)
  }
}

export class Failed extends Error {
  constructor(
    public readonly message: string,
    public readonly type: ErrorType = ErrorTypeUnknown,
    public readonly reason: string = ""
  ) {
    super(message)
  }
}

export class Err<E> {
    readonly ok = false
    readonly err = true

    constructor(public readonly val: E) {}

    public unwrap(): undefined {
        throw(this.val)
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

export type ResultError = Ok<undefined> | Err<Failed>
export type Result<T> = Ok<T> | Err<Failed>

export type ErrorType = string

export const ErrorTypeUnknown: ErrorType = ""

export const MapErrorCode = (code: number): ErrorType => {
    return ErrorTypeUnknown
}

export class Return {
    static Ok() {
        return new Ok<undefined>(undefined)
    }

    static Value<E>(val: E) {
        return new Ok<E>(val)
    }

    static Failed(
        message: string | JSX.Element,
        reason: string = "",
        type: ErrorType = ErrorTypeUnknown
    ) {
        return new Err<Failed>(new Failed(message, type, reason))
    }

    static Error<E>(val: E): Err<E> {
        return new Err<E>(val)
    }
}

export class Failed {
    constructor(
        public readonly message: string | JSX.Element,
        public readonly type: ErrorType = ErrorTypeUnknown,
        public readonly reason: string = ""
    ) {}
}

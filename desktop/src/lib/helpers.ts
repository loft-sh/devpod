export function exists<T extends any | null | undefined>(
  arg: T
): arg is Exclude<T, null | undefined> {
  return arg !== undefined && arg !== null
}

export function isError(error: unknown): error is Error {
  return error instanceof Error
}

export function noop(): void {}
export async function noopAsync(): Promise<void> {}

// WARN: All keys and values of `map` need to be serializable by `JSON.stringify` for this to work!
// See the [MDN docs](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/JSON/stringify#description)
// if you're unsure
export function serializeMap<T extends Map<unknown, unknown>>(map: T): string {
  return JSON.stringify(Array.from(map.entries()))
}

export function deserializeMap<T extends Map<unknown, unknown>>(serializedMap: string): T {
  return new Map(JSON.parse(serializedMap)) as T
}

export function isEmpty<T extends { length: number }>(arg: T): boolean {
  return arg.length <= 0
}

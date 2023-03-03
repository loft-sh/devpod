export function exists<T extends any | null | undefined>(
  arg: T
): arg is Exclude<T, null | undefined> {
  return arg !== undefined && arg !== null
}

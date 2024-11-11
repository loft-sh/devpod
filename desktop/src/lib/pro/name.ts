export function getDisplayName(
  entity:
    | Readonly<{
        metadata?: { name?: string }
        spec?: {
          displayName?: string
        }
      }>
    | undefined,
  fallback: string = ""
): string {
  if (entity?.spec?.displayName) {
    return entity.spec.displayName
  }

  if (entity?.metadata?.name) {
    return entity.metadata.name
  }

  return fallback
}

export async function safeMaxName(str: string, maxLength: number): Promise<string> {
  if (str.length <= maxLength) {
    return str
  }

  try {
    const text = new TextEncoder().encode(str)
    const digest = await crypto.subtle.digest("SHA-256", text)
    const hash = new Uint8Array(digest).reduce((s, b) => s + b.toString(16).padStart(2, "0"), "")

    return `${str.substring(0, maxLength - 8)}-${hash.substring(0, 7)}`
  } catch (err) {
    console.error("Failed to hash string", str, err)

    return str.substring(0, maxLength)
  }
}

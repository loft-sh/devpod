import {
  GIT_REPOSITORY_REGEX,
  REVISION_TYPE_CONFIG,
  REVISION_TYPE_TEST_ORDER,
  ERevisionType,
} from "./type"

export function stripProtocol(url: string): string {
  const prefixes = ["ssh://", "git@", "http://", "https://", "file://"]

  for (const prefix of prefixes) {
    if (url.startsWith(prefix)) {
      return url.substring(prefix.length)
    }
  }

  return url
}

export function extractPath(url: string): string {
  const stripped = stripProtocol(url)

  return stripped.substring(stripped.indexOf("/"))
}

export function extractRevisionType(url: string): ERevisionType | undefined {
  const path = extractPath(url)

  for (const type of REVISION_TYPE_TEST_ORDER) {
    const matches = REVISION_TYPE_CONFIG[type].partialRegex.test(path)
    if (matches) {
      return type
    }
  }

  return undefined
}

export function extractSourceValue(url: string | undefined, revisionType: ERevisionType) {
  const path = extractPath(url ?? "")

  const match = REVISION_TYPE_CONFIG[revisionType].partialRegex.exec(path)

  const matchStr = match?.[1] ?? undefined
  const revision = match?.[2] ?? undefined

  let repository = url
  if (repository && matchStr) {
    const idx = repository.lastIndexOf(matchStr)
    if (idx !== -1) {
      repository = repository.slice(0, idx) + repository.slice(idx + matchStr.length)
    }
  }

  return {
    repository,
    revision,
    repositoryValid: GIT_REPOSITORY_REGEX.test(repository ?? ""),
  }
}

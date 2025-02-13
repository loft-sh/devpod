export enum ERevisionType {
  BRANCH = "Branch",
  COMMIT = "Commit",
  PULL_REQUEST = "Pull Request",
  SUBPATH = "Subpath",
}

// Order used for testing the types by regex.
// Pull request needs to be tested before branch because "pull/[num]/head" also tests positive for BRANCH...
export const REVISION_TYPE_TEST_ORDER = Object.values(ERevisionType).reverse()

// Patterns used for detecting a valid git URL.
// WARN: Make sure these match the regexes in /pkg/git/git.go
export const GIT_REPOSITORY_REGEX = new RegExp(
  "^((?:(?:https?|git|ssh)://)?(?:[^@/\\n]+@)?(?:[^:/\\n]+)(?:[:/][^/\\n]+)+(?:\\.git)?)$"
)

export const REVISION_TYPE_CONFIG: {
  [key in ERevisionType]: {
    placeholder: string
    partialRegex: RegExp
    formatter: (val: string) => string
  }
} = {
  [ERevisionType.BRANCH]: {
    placeholder: "Enter git branch",
    partialRegex: /(@([a-zA-Z0-9./\-_]+))$/,
    formatter: (val: string) => `@${val}`,
  },
  [ERevisionType.COMMIT]: {
    placeholder: "Enter SHA256 hash",
    partialRegex: /(@sha256:([a-zA-Z0-9]+))$/,
    formatter: (val: string) => `@sha256:${val}`,
  },
  [ERevisionType.PULL_REQUEST]: {
    placeholder: "Enter PR reference number",
    partialRegex: /(@pull\/([0-9]+)\/head)$/,
    formatter: (val: string) => `@pull/${val}/head`,
  },
  [ERevisionType.SUBPATH]: {
    placeholder: "Enter sub folder path",
    partialRegex: /(@subpath:([a-zA-Z0-9\\./\-_]+))$/,
    formatter: (val: string) => `@subpath:${val}`,
  },
}

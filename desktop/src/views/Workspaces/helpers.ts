import {
  WORKSPACE_SOURCE_BRANCH_DELIMITER,
  WORKSPACE_SOURCE_COMMIT_DELIMITER,
} from "../../constants"
import { exists } from "../../lib"
import { TWorkspace, TIDEs } from "../../types"

export function getIDEName(ide: TWorkspace["ide"], ides: TIDEs | undefined) {
  const maybeIDE = ides?.find((i) => i.name === ide?.name)

  return maybeIDE?.displayName ?? ide?.name ?? maybeIDE?.name ?? "Unknown"
}

export function getSourceName({
  gitRepository,
  gitBranch,
  gitCommit,
  localFolder,
  image,
}: NonNullable<TWorkspace["source"]>): string {
  if (exists(gitRepository) && exists(gitCommit)) {
    return `${gitRepository}${WORKSPACE_SOURCE_COMMIT_DELIMITER}${gitCommit}`
  }

  if (exists(gitRepository) && exists(gitBranch)) {
    return `${gitRepository}${WORKSPACE_SOURCE_BRANCH_DELIMITER}${gitBranch}`
  }

  if (exists(gitRepository)) {
    return gitRepository
  }

  if (exists(image)) {
    return image
  }

  if (exists(localFolder)) {
    return localFolder
  }

  return ""
}

import { exists } from "../../lib"
import { TWorkspace, TIDEs } from "../../types"

export function getIDEName(ide: TWorkspace["ide"], ides: TIDEs | undefined) {
  const maybeIDE = ides?.find((i) => i.name === ide?.name)

  return maybeIDE?.displayName ?? ide?.name ?? maybeIDE?.name ?? "Unknown"
}

export function getSourceName({
  gitRepository,
  gitBranch,
  localFolder,
  image,
}: NonNullable<TWorkspace["source"]>): string {
  if (exists(gitRepository) && exists(gitBranch)) {
    return `${gitRepository}@${gitBranch}`
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

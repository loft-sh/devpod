import { WorkspaceInstanceSource } from "./constants"
import { TWorkspaceSourceType } from "@/types"

export class Source {
  readonly type: TWorkspaceSourceType
  readonly value: string

  constructor(type?: TWorkspaceSourceType, value?: string) {
    this.type = type ?? "git"
    this.value = value ?? ""
  }

  static fromRaw(rawSource?: string): Source {
    if (rawSource?.startsWith(WorkspaceInstanceSource.prefixGit)) {
      return new Source("git", rawSource.replace(WorkspaceInstanceSource.prefixGit, ""))
    }

    if (rawSource?.startsWith(WorkspaceInstanceSource.prefixImage)) {
      return new Source("image", rawSource.replace(WorkspaceInstanceSource.prefixImage, ""))
    }

    if (rawSource?.startsWith(WorkspaceInstanceSource.prefixLocal)) {
      return new Source("local", rawSource.replace(WorkspaceInstanceSource.prefixLocal, ""))
    }

    return new Source()
  }

  public stringify(): string {
    const value = this.value.trim()
    switch (this.type) {
      case "git":
        return `${WorkspaceInstanceSource.prefixGit}${value}`
      case "image":
        return `${WorkspaceInstanceSource.prefixImage}${value}`
      case "local":
        return `${WorkspaceInstanceSource.prefixLocal}${value}`
    }
  }
}

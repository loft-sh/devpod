import { UpdateWorkspace } from "../CreateWorkspace"
import { TTabProps } from "./types"

export function Configuration({ instance, template }: TTabProps) {
  return <UpdateWorkspace instance={instance} template={template} />
}

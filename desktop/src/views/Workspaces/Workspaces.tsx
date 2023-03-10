import {useWorkspaces} from "../../contexts/DevPodContext/DevPodContext";
import {useMemo} from "react";
import {exists} from "../../helpers";
import {ListItem, Text, UnorderedList} from "@chakra-ui/react";
import {Link} from "react-router-dom";

type TWorkspaceRow = Readonly<{ name: string; providerName: string | null }>
export function WorkspacesTab() {
    const workspaces = useWorkspaces()
    const providerRows = useMemo<readonly TWorkspaceRow[]>(() => {
        if (!exists(workspaces)) {
            return []
        }

        return workspaces.reduce<readonly TWorkspaceRow[]>((acc, { id, provider }) => {
            if (!exists(id)) {
                return acc
            }

            return [...acc, { providerName: provider?.name ?? null, name: id }]
        }, [])
    }, [workspaces])

    return (
        <>
            <div>Workspaces</div>
            <Link to={"/open?test=test"}>Test</Link>
            <UnorderedList>
                {providerRows.map((row) => (
                    <ListItem key={row.name}>
                        <Text fontWeight="bold">{row.name}</Text>

                        {exists(row.providerName) && <Text>Provider: {row.providerName}</Text>}
                    </ListItem>
                ))}
            </UnorderedList>
        </>
    )
}

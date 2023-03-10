import {Box, useToken} from "@chakra-ui/react";
import {DevpodIcon} from "../../icons";

export function Header() {
    // FIXME: refactor into type-safe hook
    const iconColor = useToken("colors", "primary")
    return (
        <Box width="full" paddingX={4} paddingY={4}>
            <DevpodIcon boxSize={8} color={iconColor} />
        </Box>
    )
}

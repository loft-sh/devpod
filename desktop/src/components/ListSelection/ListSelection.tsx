import { Pause, Trash } from "@/icons"
import { Button, Checkbox, FormLabel, HStack } from "@chakra-ui/react"
import { ThemeTypings } from "@chakra-ui/styled-system"
import { ReactElement, useMemo } from "react"

type TSelectionAction = {
  label: string
  icon?: ReactElement
  perform?: () => unknown
  style?: {
    colorScheme?: ThemeTypings["colorSchemes"]
  }
}

type TListSelectionProps = {
  totalAmount: number
  selectionAmount: number
  handleSelectAllClicked?: () => void
  selectionActions?: TSelectionAction[]
}

export function ListSelection({
  totalAmount,
  selectionAmount,
  handleSelectAllClicked,
  selectionActions,
}: TListSelectionProps) {
  return (
    <HStack>
      <Checkbox
        id="select-all"
        isIndeterminate={selectionAmount > 0 && selectionAmount < totalAmount}
        isChecked={totalAmount > 0 && selectionAmount === totalAmount}
        onChange={handleSelectAllClicked}
      />
      <FormLabel whiteSpace="nowrap" paddingTop="2" htmlFor="select-all">
        {selectionAmount === 0 ? "Select all" : ` ${selectionAmount} of ${totalAmount} selected`}
      </FormLabel>
      {selectionAmount > 0 && (
        <>
          {selectionActions?.map((action, index) => (
            <Button
              key={index}
              variant={"ghost"}
              colorScheme={action.style?.colorScheme}
              leftIcon={action.icon}
              onClick={action.perform}>
              {action.label}
            </Button>
          ))}
        </>
      )}
    </HStack>
  )
}

type TWorkspaceListSelectionProps = Omit<TListSelectionProps, "selectionActions"> & {
  handleDeleteClicked?: () => void
  handleStopAllClicked?: () => void
}

export function WorkspaceListSelection({
  handleDeleteClicked,
  handleStopAllClicked,
  ...props
}: TWorkspaceListSelectionProps) {
  const actions: TSelectionAction[] = useMemo(
    () => [
      {
        label: "Stop",
        icon: <Pause boxSize={4} />,
        perform: handleStopAllClicked,
      },
      {
        label: "Delete",
        icon: <Trash boxSize={4} />,
        perform: handleDeleteClicked,
        style: {
          colorScheme: "red",
        },
      },
    ],
    [handleStopAllClicked, handleDeleteClicked]
  )

  return <ListSelection {...props} selectionActions={actions} />
}

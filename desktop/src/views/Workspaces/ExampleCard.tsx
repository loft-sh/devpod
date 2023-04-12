import { Card, Image } from "@chakra-ui/react"

type TExampleCardProps = {
  image: string
  source: string
  isSelected: boolean
  onClick: (source: string) => void
}

export function ExampleCard({ image, source, isSelected, onClick }: TExampleCardProps) {
  const isSelectedProps = isSelected
    ? {
        boxShadow:
          "0px 0.6px 0.8px hsl(0deg 0% 0% / 0.09), -0.2px 2.5px 3.3px -1.3px hsl(0deg 0% 0% / 0.18)",
        borderColor: "primary.500",
        borderWidth: "thin",
      }
    : {}

  return (
    <Card
      _hover={{
        boxShadow: "rgba(186, 80, 255, 0.8) 0px 1px 4px 0px",
      }}
      transition={"box-shadow .5s"}
      width={"32"}
      height={"32"}
      alignItems={"center"}
      display={"flex"}
      justifyContent={"center"}
      cursor={"pointer"}
      onClick={() => onClick(source)}
      padding={"2.5"}
      {...isSelectedProps}>
      <Image objectFit="contain" overflow="hidden" width="fill" height="fill" src={image} />
    </Card>
  )
}

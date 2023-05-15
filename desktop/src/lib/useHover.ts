import { LegacyRef, useEffect, useRef, useState } from "react"

export function useHover<T extends HTMLButtonElement>(): [boolean, LegacyRef<T>] {
  const [isHovering, setIsHovering] = useState<boolean>(false)

  const ref = useRef<T>(null)

  useEffect(
    () => {
      const handleMouseOver = () => setIsHovering(true)
      const handleMouseOut = () => setIsHovering(false)

      setTimeout(() => {
        const node = ref.current
        if (node) {
          node.addEventListener("mouseover", handleMouseOver)
          node.addEventListener("mouseout", handleMouseOut)

          return () => {
            node.removeEventListener("mouseover", handleMouseOver)
            node.removeEventListener("mouseout", handleMouseOut)
          }
        }
      })
    },
    // rerun if ref changes!
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [ref.current]
  )

  return [isHovering, ref]
}

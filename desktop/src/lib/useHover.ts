import { MutableRefObject, useEffect, useRef, useState } from "react"

export function useHover<T extends HTMLElement>(): [boolean, MutableRefObject<T | null>] {
  const [isHovering, setIsHovering] = useState<boolean>(false)

  const ref = useRef<T | null>(null)

  useEffect(
    () => {
      const node = ref.current
      const handleMouseOver = (): void => setIsHovering(true)
      const handleMouseOut = (): void => setIsHovering(false)
      if (node) {
        node.addEventListener("mouseover", handleMouseOver)
        node.addEventListener("mouseout", handleMouseOut)

        return () => {
          node.removeEventListener("mouseover", handleMouseOver)
          node.removeEventListener("mouseout", handleMouseOut)
        }
      }
    },
    // rerun if ref changes!
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [ref.current]
  )

  return [isHovering, ref]
}

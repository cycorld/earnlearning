import { cn } from "@/lib/utils"

interface SpinnerProps {
  size?: "sm" | "md" | "lg"
  className?: string
}

/**
 * 브랜드 스피너 — border-primary + border-t-transparent 조합을 재사용.
 * 페이지 중앙 배치 시엔 부모에 flex justify-center 를 줄 것.
 */
export function Spinner({ size = "md", className }: SpinnerProps) {
  const sizeClasses = {
    sm: "h-5 w-5 border-[3px]",
    md: "h-8 w-8 border-4",
    lg: "h-12 w-12 border-4",
  }
  return (
    <div
      role="status"
      aria-label="loading"
      className={cn(
        "animate-spin rounded-full border-primary border-t-transparent",
        sizeClasses[size],
        className,
      )}
    />
  )
}

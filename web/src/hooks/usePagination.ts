import { useState, useMemo } from 'react'

interface UsePaginationOptions {
  pageSize?: number
}

interface UsePaginationResult<T> {
  page: number
  pageSize: number
  totalPages: number
  totalItems: number
  pageItems: T[]
  setPage: (page: number) => void
  nextPage: () => void
  prevPage: () => void
  canNext: boolean
  canPrev: boolean
  startIndex: number
  endIndex: number
}

export function usePagination<T>(
  items: T[] | undefined,
  options: UsePaginationOptions = {},
): UsePaginationResult<T> {
  const { pageSize = 12 } = options
  const [page, setPage] = useState(1)

  const totalItems = items?.length ?? 0
  const totalPages = Math.max(1, Math.ceil(totalItems / pageSize))

  // Reset to page 1 if current page exceeds total
  const safePage = Math.min(page, totalPages)

  const pageItems = useMemo(() => {
    if (!items) return []
    const start = (safePage - 1) * pageSize
    return items.slice(start, start + pageSize)
  }, [items, safePage, pageSize])

  const startIndex = totalItems === 0 ? 0 : (safePage - 1) * pageSize + 1
  const endIndex = Math.min(safePage * pageSize, totalItems)

  return {
    page: safePage,
    pageSize,
    totalPages,
    totalItems,
    pageItems,
    setPage: (p: number) => setPage(Math.max(1, Math.min(p, totalPages))),
    nextPage: () => setPage((p) => Math.min(p + 1, totalPages)),
    prevPage: () => setPage((p) => Math.max(p - 1, 1)),
    canNext: safePage < totalPages,
    canPrev: safePage > 1,
    startIndex,
    endIndex,
  }
}

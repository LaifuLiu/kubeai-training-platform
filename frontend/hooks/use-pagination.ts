import { useState } from "react";

export interface UsePaginationOptions {
  initialPage?: number;
  initialSize?: number;
}

export function usePagination({
  initialPage = 1,
  initialSize = 10,
}: UsePaginationOptions = {}) {
  const [page, setPage] = useState(initialPage);
  const [size, setSize] = useState(initialSize);

  const changePage = (newPage: number) => {
    if (newPage < 1) return;
    setPage(newPage);
  };

  const changeSize = (newSize: number) => {
    if (newSize < 1 || newSize > 100) return;

    setSize(newSize);
    setPage(1); // Reset to the first page when page size changes.
  };

  return {
    page,
    size,
    changePage,
    changeSize,
  };
}

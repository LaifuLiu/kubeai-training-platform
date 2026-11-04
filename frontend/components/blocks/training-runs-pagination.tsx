import {
  Pagination,
  PaginationContent,
  PaginationEllipsis,
  PaginationItem,
  PaginationLink,
  PaginationNext,
  PaginationPrevious,
} from "@/components/ui/pagination";

export function TrainingRunsPagination({
  page,
  totalPages,
  changePage,
}: {
  page: number;
  totalPages: number;
  changePage: (page: number) => void;
}) {
  return (
    <Pagination>
      <PaginationContent>
        <PaginationItem>
          <PaginationPrevious
            onClick={() => changePage(page - 1)}
            aria-disabled={page === 1}
            className={page === 1 ? "pointer-events-none opacity-50" : ""}
          />
        </PaginationItem>

        {Array.from({ length: totalPages }, (_, index) => index + 1).map(
          (pageNumber) => (
            <PaginationItem key={pageNumber}>
              <PaginationLink
                isActive={pageNumber === page}
                onClick={() => changePage(pageNumber)}
              >
                {pageNumber}
              </PaginationLink>
            </PaginationItem>
          ),
        )}

        <PaginationItem>
          <PaginationNext
            onClick={() => changePage(page + 1)}
            aria-disabled={page === totalPages}
            className={
              page === totalPages ? "pointer-events-none opacity-50" : ""
            }
          />
        </PaginationItem>
      </PaginationContent>
    </Pagination>
  );
}

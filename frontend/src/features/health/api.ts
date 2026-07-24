import { useQuery } from "@tanstack/react-query";

import { apiGet } from "@/shared/api/client";
import type { ReadyStatus } from "@/shared/types";

/** Polls readiness. Server state lives in TanStack Query, never copied to a store. */
export function useReadiness() {
  return useQuery({
    queryKey: ["readiness"],
    queryFn: ({ signal }) => apiGet<ReadyStatus>("/readyz", signal),
    refetchInterval: 10_000,
    retry: false,
  });
}

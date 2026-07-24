import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { apiPost, refreshSession, setAccessToken } from "@/shared/api/client";
import type { AuthUser } from "@/shared/types";

const SESSION_KEY = ["session"] as const;

/**
 * Bootstraps the session from the refresh cookie. Runs once on load: if the
 * cookie is valid it yields the signed-in user and stashes the access token;
 * otherwise it errors and the app treats the visitor as anonymous.
 *
 * retry is off because a 401 here is a definitive "not signed in", not a
 * transient failure worth retrying.
 */
export function useSession() {
  const query = useQuery<AuthUser | null>({
    queryKey: SESSION_KEY,
    queryFn: async () => (await refreshSession()).user,
    retry: false,
    staleTime: Infinity,
    refetchOnWindowFocus: false,
  });

  return { user: query.data ?? null, isLoading: query.isLoading };
}

/** Signs out: revokes the current session server-side, then clears local state. */
export function useLogout() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => apiPost("/auth/logout"),
    onSuccess: () => {
      setAccessToken(null);
      queryClient.setQueryData<AuthUser | null>(SESSION_KEY, null);
    },
  });
}

/** Starts the GitHub login by navigating the whole page to the backend. */
export function startGitHubLogin(): void {
  window.location.href = "/api/v1/auth/github";
}

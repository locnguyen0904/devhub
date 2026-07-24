import type { ReactNode } from "react";
import { Navigate } from "react-router-dom";

import { useSession } from "@/features/auth/api";

/** Redirects to the feed when no one is signed in. Guards the editor and the
 * author's own post list. */
export function RequireAuth({ children }: { children: ReactNode }) {
  const { user, isLoading } = useSession();
  if (isLoading) return null;
  if (!user) return <Navigate to="/" replace />;
  return children;
}

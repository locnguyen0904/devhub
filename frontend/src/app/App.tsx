import { useState } from "react";
import { Link, Outlet, useNavigate } from "react-router-dom";

import { AuthControls } from "@/features/auth/AuthControls";
import { useSession } from "@/features/auth/api";
import { useTheme } from "@/shared/theme/use-theme";

const THEME_LABEL: Record<string, string> = {
  system: "System",
  light: "Light",
  dark: "Dark",
};

// The callback bounces here with ?auth_error=1 when GitHub login fails.
const loginFailed = new URLSearchParams(window.location.search).has("auth_error");

/** App shell: header with nav and theme toggle, and the routed page below. */
export function App() {
  const { theme, cycle } = useTheme();
  const { user } = useSession();
  const navigate = useNavigate();
  const [search, setSearch] = useState("");

  return (
    <div className="min-h-screen">
      <header className="border-b border-border-subtle">
        <div className="mx-auto flex max-w-3xl items-center justify-between gap-4 px-6 py-4">
          <nav className="flex items-center gap-4">
            <Link to="/" className="text-lg font-semibold text-accent">
              DevHub
            </Link>
            {user && (
              <>
                <Link to="/me/posts" className="text-sm text-text-muted hover:text-text-primary">
                  My posts
                </Link>
                <Link to="/me/saved" className="text-sm text-text-muted hover:text-text-primary">
                  Saved
                </Link>
              </>
            )}
          </nav>
          <form
            className="hidden flex-1 sm:block"
            onSubmit={(e) => {
              e.preventDefault();
              if (search.trim().length >= 2) {
                void navigate(`/search?q=${encodeURIComponent(search.trim())}`);
              }
            }}
          >
            <input
              value={search}
              onChange={(e) => {
                setSearch(e.target.value);
              }}
              placeholder="Search posts…"
              className="w-full rounded-[--radius-control] border border-border-strong bg-surface px-3 py-1.5 text-sm outline-none focus:border-accent"
            />
          </form>
          <div className="flex items-center gap-3">
            {user && (
              <Link
                to="/new"
                className="rounded-[--radius-control] bg-accent px-3 py-1.5 text-sm font-medium text-accent-fg hover:bg-accent-hover"
              >
                Write
              </Link>
            )}
            <AuthControls />
            <button
              type="button"
              onClick={cycle}
              className="rounded-[--radius-control] border border-border-strong px-3 py-1.5 text-sm text-text-muted hover:bg-surface-sunken"
            >
              {THEME_LABEL[theme]}
            </button>
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-3xl px-6 py-10">
        {loginFailed && (
          <div
            role="alert"
            className="mb-6 rounded-[--radius-card] border border-border-subtle bg-surface-raised p-4 text-danger"
          >
            GitHub sign-in failed. Please try again.
          </div>
        )}
        <Outlet />
      </main>
    </div>
  );
}

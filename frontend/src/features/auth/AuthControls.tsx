import { startGitHubLogin, useLogout, useSession } from "./api";

/** Header control that shows the signed-in user or a GitHub login button. */
export function AuthControls() {
  const { user, isLoading } = useSession();
  const logout = useLogout();

  if (isLoading) {
    return <span className="h-8 w-24 rounded bg-surface-sunken" aria-hidden />;
  }

  if (!user) {
    return (
      <button
        type="button"
        onClick={startGitHubLogin}
        className="rounded-[--radius-control] bg-accent px-3 py-1.5 text-sm font-medium text-accent-fg hover:bg-accent-hover"
      >
        Sign in with GitHub
      </button>
    );
  }

  return (
    <div className="flex items-center gap-3">
      {user.avatar_url && (
        <img
          src={user.avatar_url}
          alt={user.display_name}
          className="h-8 w-8 rounded-full"
        />
      )}
      <span className="text-sm text-text-muted">{user.display_name}</span>
      <button
        type="button"
        onClick={() => {
          logout.mutate();
        }}
        disabled={logout.isPending}
        className="rounded-[--radius-control] border border-border-strong px-3 py-1.5 text-sm text-text-muted hover:bg-surface-sunken disabled:opacity-50"
      >
        Sign out
      </button>
    </div>
  );
}

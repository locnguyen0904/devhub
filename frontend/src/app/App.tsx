import { AuthControls } from "@/features/auth/AuthControls";
import { StatusPanel } from "@/features/health/StatusPanel";
import { useTheme } from "@/shared/theme/use-theme";

const THEME_LABEL: Record<string, string> = {
  system: "System",
  light: "Light",
  dark: "Dark",
};

// The callback bounces here with ?auth_error=1 when GitHub login fails.
const loginFailed = new URLSearchParams(window.location.search).has("auth_error");

export function App() {
  const { theme, cycle } = useTheme();

  return (
    <div className="min-h-screen">
      <header className="border-b border-border-subtle">
        <div className="mx-auto flex max-w-3xl items-center justify-between px-6 py-4">
          <span className="text-lg font-semibold text-accent">DevHub</span>
          <div className="flex items-center gap-3">
            <AuthControls />
            <button
              type="button"
              onClick={cycle}
              className="rounded-[--radius-control] border border-border-strong px-3 py-1.5 text-sm text-text-muted hover:bg-surface-sunken"
            >
              Theme: {THEME_LABEL[theme]}
            </button>
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-3xl space-y-6 px-6 py-10">
        {loginFailed && (
          <div
            role="alert"
            className="rounded-[--radius-card] border border-border-subtle bg-surface-raised p-4 text-danger"
          >
            GitHub sign-in failed. Please try again.
          </div>
        )}
        <StatusPanel />
      </main>
    </div>
  );
}

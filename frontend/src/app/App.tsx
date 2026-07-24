import { StatusPanel } from "@/features/health/StatusPanel";
import { useTheme } from "@/shared/theme/use-theme";

const THEME_LABEL: Record<string, string> = {
  system: "System",
  light: "Light",
  dark: "Dark",
};

export function App() {
  const { theme, cycle } = useTheme();

  return (
    <div className="min-h-screen">
      <header className="border-b border-border-subtle">
        <div className="mx-auto flex max-w-3xl items-center justify-between px-6 py-4">
          <span className="text-lg font-semibold text-accent">DevHub</span>
          <button
            type="button"
            onClick={cycle}
            className="rounded-[--radius-control] border border-border-strong px-3 py-1.5 text-sm text-text-muted hover:bg-surface-sunken"
          >
            Theme: {THEME_LABEL[theme]}
          </button>
        </div>
      </header>

      <main className="mx-auto max-w-3xl px-6 py-10">
        <StatusPanel />
      </main>
    </div>
  );
}

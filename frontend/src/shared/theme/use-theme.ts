/**
 * Theme state. Lives in shared/ because it is the only place allowed to touch
 * localStorage and matchMedia (CLAUDE.md §4).
 *
 * The initial class on <html> is set by an inline script in index.html before
 * first paint; this hook takes over once React mounts.
 */
import { useCallback, useEffect, useState } from "react";

export type Theme = "light" | "dark" | "system";

const STORAGE_KEY = "theme";
const DARK_QUERY = "(prefers-color-scheme: dark)";

function readStored(): Theme {
  const saved = localStorage.getItem(STORAGE_KEY);
  return saved === "light" || saved === "dark" ? saved : "system";
}

function resolve(theme: Theme): boolean {
  if (theme === "system") return window.matchMedia(DARK_QUERY).matches;
  return theme === "dark";
}

export function useTheme() {
  const [theme, setTheme] = useState<Theme>(readStored);

  // Synchronising the DOM class and localStorage with state is exactly what
  // Effects are for: an external system, not a derived value.
  useEffect(() => {
    document.documentElement.classList.toggle("dark", resolve(theme));

    if (theme === "system") {
      localStorage.removeItem(STORAGE_KEY);
      const media = window.matchMedia(DARK_QUERY);
      const onChange = () => {
        document.documentElement.classList.toggle("dark", media.matches);
      };
      media.addEventListener("change", onChange);
      return () => {
        media.removeEventListener("change", onChange);
      };
    }

    localStorage.setItem(STORAGE_KEY, theme);
    return undefined;
  }, [theme]);

  const cycle = useCallback(() => {
    setTheme((current) =>
      current === "system" ? "light" : current === "light" ? "dark" : "system",
    );
  }, []);

  return { theme, setTheme, cycle };
}

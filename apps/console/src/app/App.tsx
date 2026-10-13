import { APIStatus } from "../features/jobs/components/api-status";
import { JobsWorkspace } from "../features/jobs/pages/jobs-workspace";
import { ApertureIcon, MoonIcon, SunIcon } from "../shared/icons";
import { useTheme } from "../shared/theme/use-theme";

export default function App() {
  const { theme, toggleTheme } = useTheme();

  return (
    <div className="min-h-screen bg-canvas text-ink">
      <header className="border-b border-line/80 bg-canvas/90 backdrop-blur-xl">
        <div className="mx-auto flex h-18 max-w-[1480px] items-center justify-between px-5 sm:px-8">
          <div className="flex items-center gap-3">
            <div className="grid size-9 place-items-center rounded-xl bg-pine text-mint shadow-sm">
              <ApertureIcon className="size-5" />
            </div>
            <div>
              <span className="font-display text-lg font-semibold tracking-tight">
                ImageFlow
              </span>
              <span className="ml-2 text-sm text-muted">Console</span>
            </div>
          </div>
          <div className="flex items-center gap-2 sm:gap-3">
            <button
              type="button"
              className="theme-toggle"
              onClick={toggleTheme}
              aria-label={`Cambiar a tema ${theme === "light" ? "oscuro" : "claro"}`}
              title={`Usar tema ${theme === "light" ? "oscuro" : "claro"}`}
            >
              {theme === "light" ? (
                <MoonIcon className="size-4" />
              ) : (
                <SunIcon className="size-4" />
              )}
            </button>
            <span className="hidden text-xs font-medium text-muted sm:block">
              API local
            </span>
            <APIStatus />
          </div>
        </div>
      </header>

      <main className="mx-auto max-w-[1480px] px-5 py-6 sm:px-8 sm:py-8">
        <JobsWorkspace />
      </main>
      <footer className="mx-auto flex max-w-[1480px] items-center justify-between px-5 pb-8 pt-3 text-xs text-muted sm:px-8">
        <span>ImageFlow Console · v0.1</span>
        <span>API-first image operations</span>
      </footer>
    </div>
  );
}

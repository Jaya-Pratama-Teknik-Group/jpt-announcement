import { readonly, ref } from "vue";

export type ThemePreference = "light" | "dark" | "system";
export type ResolvedTheme = "light" | "dark";

const STORAGE_KEY = "jpt-announce-theme";

const themePreference = ref<ThemePreference>("light");
const resolvedTheme = ref<ResolvedTheme>("light");

let mediaQuery: MediaQueryList | null = null;
let listenerBound = false;

function isValidPreference(value: string | null): value is ThemePreference {
  return value === "light" || value === "dark" || value === "system";
}

function resolveTheme(pref: ThemePreference): ResolvedTheme {
  if (pref === "light" || pref === "dark") return pref;
  if (typeof window !== "undefined" && window.matchMedia) {
    return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
  }
  return "light";
}

function applyTheme(pref: ThemePreference): void {
  const nextResolved = resolveTheme(pref);
  resolvedTheme.value = nextResolved;
  if (typeof document !== "undefined") {
    document.documentElement.dataset.theme = nextResolved;
    document.documentElement.style.colorScheme = nextResolved;
  }
}

function onSystemThemeChange(): void {
  if (themePreference.value === "system") {
    applyTheme("system");
  }
}

export function initTheme(): void {
  let saved: string | null = null;
  try {
    saved = window.localStorage.getItem(STORAGE_KEY);
  } catch {
    saved = null;
  }

  themePreference.value = isValidPreference(saved) ? saved : "light";
  applyTheme(themePreference.value);

  if (!listenerBound && typeof window !== "undefined" && window.matchMedia) {
    mediaQuery = window.matchMedia("(prefers-color-scheme: dark)");
    mediaQuery.addEventListener("change", onSystemThemeChange);
    listenerBound = true;
  }
}

export function setThemePreference(pref: ThemePreference): void {
  themePreference.value = pref;
  applyTheme(pref);
  try {
    window.localStorage.setItem(STORAGE_KEY, pref);
  } catch {
    // Ignore storage quota / privacy-mode errors; runtime theme still applies.
  }
}

export function useTheme() {
  return {
    themePreference: readonly(themePreference),
    resolvedTheme: readonly(resolvedTheme),
    setThemePreference
  };
}

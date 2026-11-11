(() => {
  const key = "kbmd-theme";
  const light = "light";
  const dark = "dark";

  const stored = () => {
    try {
      const value = localStorage.getItem(key);
      return value === light || value === dark ? value : "";
    } catch {
      return "";
    }
  };

  const browserTheme = () =>
    window.matchMedia(`(prefers-color-scheme: ${dark})`).matches ? dark : light;

  const effective = () => document.documentElement.dataset.theme || browserTheme();

  const applyStored = () => {
    const theme = stored();
    if (theme) document.documentElement.dataset.theme = theme;
    else delete document.documentElement.dataset.theme;
  };

  const paint = () => {
    const button = document.getElementById("theme-toggle");
    if (!button) return;
    const isDark = effective() === dark;
    const next = isDark ? light : dark;
    button.textContent = `${next[0].toUpperCase()}${next.slice(1)}`;
    button.setAttribute("aria-pressed", String(isDark));
    button.setAttribute("aria-label", `Switch to ${next} mode`);
  };

  applyStored();

  document.addEventListener("DOMContentLoaded", () => {
    paint();
    document.getElementById("theme-toggle")?.addEventListener("click", () => {
      const next = effective() === dark ? light : dark;
      try {
        localStorage.setItem(key, next);
      } catch {
        // Storage can be blocked; the choice still applies to this page.
      }
      document.documentElement.dataset.theme = next;
      paint();
    });
    window.matchMedia(`(prefers-color-scheme: ${dark})`).addEventListener("change", () => {
      if (!stored()) paint();
    });
  });
})();

/**
 * Приложение собирается как SPA: adapter-static с `fallback: "index.html"`,
 * серверного рендера нет, данные приходят только из браузера (00-CONTEXT §2, §7.2).
 */
export const ssr = false;
export const prerender = false;

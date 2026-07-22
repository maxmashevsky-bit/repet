export type IconName =
  | "arrow"
  | "attach"
  | "bell"
  | "book"
  | "calendar"
  | "chart"
  | "chat"
  | "check"
  | "chevron"
  | "clock"
  | "close"
  | "eye"
  | "globe"
  | "lock"
  | "menu"
  | "monitor"
  | "palette"
  | "plus"
  | "search"
  | "send"
  | "shield"
  | "task"
  | "user";

export function Icon({ name, size = 22, className = "" }: { name: IconName; size?: number; className?: string }) {
  const common = { fill: "none", stroke: "currentColor", strokeWidth: 1.8, strokeLinecap: "round" as const, strokeLinejoin: "round" as const };
  let content: React.ReactNode;

  switch (name) {
    case "arrow": content = <><path d="M5 12h14" /><path d="m14 7 5 5-5 5" /></>; break;
    case "attach": content = <path d="m8.5 12.8 6.8-6.8a3 3 0 0 1 4.2 4.2l-8.6 8.6a5 5 0 0 1-7.1-7.1l8.5-8.5" />; break;
    case "bell": content = <><path d="M18 8a6 6 0 0 0-12 0c0 7-3 7-3 9h18c0-2-3-2-3-9" /><path d="M10 21h4" /></>; break;
    case "book": content = <><path d="M4 5.5A3.5 3.5 0 0 1 7.5 2H11v17H7.5A3.5 3.5 0 0 0 4 22.5v-17Z" /><path d="M20 5.5A3.5 3.5 0 0 0 16.5 2H13v17h3.5a3.5 3.5 0 0 1 3.5 3.5v-17Z" /></>; break;
    case "calendar": content = <><rect x="3" y="5" width="18" height="16" rx="2" /><path d="M16 3v4M8 3v4M3 10h18" /></>; break;
    case "chart": content = <><path d="M4 19V9M10 19V5M16 19v-7M22 19V2" /><path d="m3 8 6-5 6 4 7-6" /></>; break;
    case "chat": content = <path d="M21 12a8 8 0 0 1-8 8H7l-4 2 1.3-4.3A9 9 0 1 1 21 12Z" />; break;
    case "check": content = <path d="m5 12 4 4L19 6" />; break;
    case "chevron": content = <path d="m9 18 6-6-6-6" />; break;
    case "clock": content = <><circle cx="12" cy="12" r="9" /><path d="M12 7v5l3 2" /></>; break;
    case "close": content = <><path d="m6 6 12 12M18 6 6 18" /></>; break;
    case "eye": content = <><path d="M2.5 12s3.5-6 9.5-6 9.5 6 9.5 6-3.5 6-9.5 6-9.5-6-9.5-6Z" /><circle cx="12" cy="12" r="2.5" /></>; break;
    case "globe": content = <><circle cx="12" cy="12" r="9" /><path d="M3 12h18M12 3a14 14 0 0 1 0 18M12 3a14 14 0 0 0 0 18" /></>; break;
    case "lock": content = <><rect x="4" y="10" width="16" height="11" rx="2" /><path d="M8 10V7a4 4 0 0 1 8 0v3" /></>; break;
    case "menu": content = <><path d="M4 7h16M4 12h16M4 17h16" /></>; break;
    case "monitor": content = <><rect x="3" y="4" width="18" height="13" rx="2" /><path d="M8 21h8M12 17v4" /></>; break;
    case "palette": content = <><path d="M12 3a9 9 0 1 0 0 18h1.5a2 2 0 0 0 0-4H12a2 2 0 0 1 0-4h5a4 4 0 0 0 4-4c0-3.3-4-6-9-6Z" /><path d="M7 10h.01M9 6h.01M14 6h.01M18 9h.01" /></>; break;
    case "plus": content = <><path d="M12 5v14M5 12h14" /></>; break;
    case "search": content = <><circle cx="11" cy="11" r="7" /><path d="m20 20-4-4" /></>; break;
    case "send": content = <><path d="m3 3 18 9-18 9 4-9-4-9Z" /><path d="M7 12h14" /></>; break;
    case "shield": content = <><path d="M12 2 4 5v6c0 5 3.4 9.2 8 11 4.6-1.8 8-6 8-11V5l-8-3Z" /><path d="m9 12 2 2 4-4" /></>; break;
    case "task": content = <><path d="M6 3h12v18H6z" /><path d="M9 8h6M9 12h6M9 16h3" /></>; break;
    case "user": content = <><circle cx="12" cy="8" r="4" /><path d="M4 21a8 8 0 0 1 16 0" /></>; break;
    default: content = null;
  }

  return <svg className={`ui-icon ${className}`.trim()} width={size} height={size} viewBox="0 0 24 24" aria-hidden="true" {...common}>{content}</svg>;
}

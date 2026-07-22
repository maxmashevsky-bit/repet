import { AppShell } from "../../features/app-shell/AppShell";

export default function ProductLayout({ children }: { children: React.ReactNode }) {
  return <AppShell>{children}</AppShell>;
}


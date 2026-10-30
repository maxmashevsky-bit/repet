"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { createContext, ReactNode, useContext, useEffect, useMemo, useRef, useState } from "react";
import { authApi, productApi, ApiError } from "../../shared/api/client";
import type { Dashboard } from "../../shared/api/types";
import { usePolling } from "../messages/usePolling";
import { NotificationsPopover } from "../notifications/NotificationsPopover";
import { Brand } from "../ui/Brand";
import { Icon } from "../ui/Icon";

type WorkspaceContext = {
  dashboard: Dashboard;
  refresh: () => Promise<void>;
  refreshConversations: (signal?:AbortSignal)=>Promise<void>;
};

const Context = createContext<WorkspaceContext | null>(null);

const nav = [
  { href: "/", label: "Главное" },
  { href: "/messages", label: "Сообщения" },
  { href: "/calendar", label: "Календарь" },
  { href: "/tasks", label: "Задания" }
];

export function useWorkspace() {
  const value = useContext(Context);
  if (!value) throw new Error("useWorkspace must be used inside AppShell");
  return value;
}

export function AppShell({ children }: { children: ReactNode }) {
  const [dashboard, setDashboard] = useState<Dashboard | null>(null);
  const [error, setError] = useState("");
  const [menuOpen, setMenuOpen] = useState(false);
  const pathname = usePathname();
  const router = useRouter();
  const notificationEpoch=useRef(0),conversationEpoch=useRef(0),dashboardEpoch=useRef(0);

  async function refresh() {
    const generation=++dashboardEpoch.current,notifications=++notificationEpoch.current,conversations=++conversationEpoch.current;
    setError("");
    try {
      const next=await productApi.dashboard();
      if(generation===dashboardEpoch.current)setDashboard(previous=>({...next,notifications:notifications===notificationEpoch.current||!previous?next.notifications:previous.notifications,conversations:conversations===conversationEpoch.current||!previous?next.conversations:previous.conversations}));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Не удалось загрузить данные");
    }
  }

  useEffect(() => {
    productApi.dashboard().then(setDashboard).catch((err: unknown) => { if(err instanceof ApiError&&err.code==="email_unverified"){router.replace("/verify-email");return;}setError(err instanceof Error ? err.message : "Не удалось загрузить данные"); });
  }, [router]);

  usePolling(async(signal)=>{
    const generation=notificationEpoch.current;
    const notifications=await productApi.notifications(signal);
    if(!signal.aborted&&generation===notificationEpoch.current)setDashboard(old=>old?{...old,notifications}:old);
  },Boolean(dashboard),20000);

  async function refreshConversations(signal?:AbortSignal){
    const generation=++conversationEpoch.current,conversations=await productApi.conversations(signal);
    if(!signal?.aborted&&generation===conversationEpoch.current)setDashboard(old=>old?{...old,conversations}:old);
  }

  async function logout() {
    await authApi.logout();
    router.push("/login");
  }

  const unread = useMemo(() => dashboard?.notifications.filter((item) => !item.read_at).length ?? 0, [dashboard]);

  if (!dashboard && !error) {
    return <div className="full-state">Загружаем «Репет»...</div>;
  }

  if (!dashboard) {
    return (
      <main className="full-state">
        <h1>Не удалось открыть кабинет</h1>
        <p>{error}</p>
        <Link className="button" href="/login">Войти заново</Link>
      </main>
    );
  }

  return (
    <Context.Provider value={{ dashboard, refresh, refreshConversations }}>
      <div className="app-frame">
        <a className="skip-link" href="#content">Перейти к содержимому</a>
        <header className="app-header">
          <Brand />
          <nav className="top-nav" aria-label="Главное меню">
            {nav.map((item) => (
              <Link key={item.href} href={item.href} className={pathname === item.href || item.href !== "/" && pathname.startsWith(item.href + "/") ? "active" : ""}>
                {item.label}
              </Link>
            ))}
          </nav>
          <div className="header-actions">
            <NotificationsPopover unread={unread} />
            <Link href="/settings" className="profile-link" title={dashboard.user.display_name} aria-label={`Профиль: ${dashboard.user.display_name}`}>
              <span className="avatar" aria-hidden="true">{dashboard.user.display_name.slice(0, 1)}</span>
              <span className="profile-copy">
                <strong>{dashboard.user.display_name}</strong>
                <small>{dashboard.user.role === "tutor" ? "Преподаватель" : "Ученик"}</small>
              </span>
              <Icon name="chevron" size={17} />
            </Link>
            <button className="icon-button mobile-only" type="button" aria-label="Открыть меню" onClick={() => setMenuOpen(true)}>
              <Icon name="menu" />
            </button>
          </div>
        </header>

        {menuOpen ? (
          <div className="drawer-layer" role="presentation" onClick={() => setMenuOpen(false)}>
            <aside className="drawer" role="dialog" aria-modal="true" aria-label="Мобильное меню" onClick={(event) => event.stopPropagation()}>
              <button className="icon-button drawer-close" type="button" aria-label="Закрыть меню" onClick={() => setMenuOpen(false)}><Icon name="close" /></button>
              {nav.map((item) => (
                <Link key={item.href} href={item.href} onClick={() => setMenuOpen(false)}>
                  {item.label}
                </Link>
              ))}
              <Link href="/settings" onClick={() => setMenuOpen(false)}>Настройки</Link>
              <button type="button" onClick={logout}>Выйти</button>
            </aside>
          </div>
        ) : null}

        <main id="content" className="content-shell">
          {error ? (
            <div className="notice error">
              {error}
              <button type="button" onClick={refresh}>Повторить</button>
            </div>
          ) : null}
          {children}
        </main>
        <footer className="app-footer">
          <p>Репет</p>
          {process.env.NEXT_PUBLIC_SUPPORT_CREATOR_URL ? (
            <a className="support-link" href={process.env.NEXT_PUBLIC_SUPPORT_CREATOR_URL}>Поддержать создателя ♥</a>
          ) : null}
          <nav aria-label="Ссылки в подвале">
            <Link href="/privacy">Конфиденциальность</Link>
            <Link href="/terms">Условия использования</Link>
            <span>© {new Date().getFullYear()} Репет</span>
          </nav>
        </footer>
      </div>
    </Context.Provider>
  );
}

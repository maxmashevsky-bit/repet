"use client";

import Link from "next/link";
import { useState } from "react";
import { productApi } from "../../shared/api/client";
import { formatDateTime } from "../../shared/lib/date";
import { useWorkspace } from "../app-shell/AppShell";
import { Icon } from "../ui/Icon";

export function NotificationsPopover({ unread }: { unread: number }) {
  const { dashboard, refresh } = useWorkspace();
  const [open, setOpen] = useState(false);
  const [loading, setLoading] = useState(false);

  async function markAll() {
    setLoading(true);
    await productApi.markAllNotificationsRead();
    await refresh();
    setLoading(false);
  }

  return (
    <div className="notifications">
      <button className="bell-button" type="button" aria-label="Открыть уведомления" onClick={() => setOpen((value) => !value)}>
        <Icon name="bell" size={25} />
        {unread > 0 ? <strong>{unread}</strong> : null}
      </button>
      {open ? (
        <section className="popover" aria-label="Уведомления">
          <header>
            <h2>Уведомления</h2>
            <button type="button" className="link-button" disabled={loading || unread === 0} onClick={markAll}>
              Отметить все
            </button>
          </header>
          {dashboard.notifications.length === 0 ? (
            <p className="empty">Пока нет уведомлений.</p>
          ) : (
            <ul className="stack-list">
              {dashboard.notifications.map((item) => (
                <li key={item.id} className={item.read_at ? "" : "unread"}>
                  <Link href={item.href} onClick={() => setOpen(false)}>
                    <strong>{item.title}</strong>
                    <span>{item.body}</span>
                    <small>{formatDateTime(item.created_at)}</small>
                  </Link>
                </li>
              ))}
            </ul>
          )}
        </section>
      ) : null}
    </div>
  );
}

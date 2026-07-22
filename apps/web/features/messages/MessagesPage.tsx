"use client";

import { FormEvent, useEffect, useMemo, useState } from "react";
import { useSearchParams } from "next/navigation";
import { productApi } from "../../shared/api/client";
import type { Message } from "../../shared/api/types";
import { formatTime } from "../../shared/lib/date";
import { useWorkspace } from "../app-shell/AppShell";
import { Icon } from "../ui/Icon";

export function MessagesPage() {
  const { dashboard, refresh } = useWorkspace();
  const search = useSearchParams();
  const firstConversation = dashboard.conversations[0]?.id ?? "";
  const [conversationID, setConversationID] = useState(search.get("conversation") ?? firstConversation);
  const [messages, setMessages] = useState<Message[]>([]);
  const [body, setBody] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const conversation = useMemo(
    () => dashboard.conversations.find((item) => item.id === conversationID) ?? dashboard.conversations[0],
    [dashboard.conversations, conversationID]
  );

  useEffect(() => {
    if (!conversation) return;
    setConversationID(conversation.id);
    setLoading(true);
    productApi.messages(conversation.id)
      .then(setMessages)
      .catch((err: unknown) => setError(err instanceof Error ? err.message : "Не удалось загрузить чат"))
      .finally(() => setLoading(false));
  }, [conversation?.id]);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!conversation || body.trim() === "") return;
    const optimistic: Message = {
      id: `pending-${Date.now()}`,
      conversation_id: conversation.id,
      author_id: dashboard.user.id,
      body,
      created_at: new Date().toISOString()
    };
    setMessages((items) => [...items, optimistic]);
    setBody("");
    try {
      const saved = await productApi.sendMessage(conversation.id, optimistic.body);
      setMessages((items) => items.map((item) => (item.id === optimistic.id ? saved : item)));
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Сообщение не отправлено");
      setMessages((items) => items.filter((item) => item.id !== optimistic.id));
      setBody(optimistic.body);
    }
  }

  return (
    <section className={dashboard.user.role === "tutor" ? "messages-layout with-context" : "messages-layout"}>
      <div className="page-title full"><h1>Сообщения</h1><p>Общайтесь и получайте материалы к занятиям</p></div>
      <aside className="panel dialog-list">
        <div className="heading-with-count"><h2>Диалоги</h2><span>{dashboard.conversations.length}</span></div>
        <label className="search-field"><Icon name="search" size={20} /><input aria-label="Поиск по диалогам" placeholder="Поиск" /></label>
        {dashboard.conversations.length === 0 ? <p className="empty">Диалоги появятся после подтверждения связи.</p> : dashboard.conversations.map((item) => (
          <button key={item.id} type="button" className={item.id === conversation?.id ? "dialog-row active" : "dialog-row"} onClick={() => setConversationID(item.id)}>
            <span className="avatar small">{(dashboard.user.role === "tutor" ? item.student_name : item.tutor_name).slice(0, 1)}</span>
            <span className="dialog-copy"><strong>{dashboard.user.role === "tutor" ? item.student_name : item.tutor_name}</strong><small>{item.last_message || "Нет сообщений"}</small></span>
            {item.unread_count > 0 ? <em>{item.unread_count}</em> : null}
          </button>
        ))}
      </aside>
      <section className="panel chat-panel" aria-label="Активный чат">
        {conversation ? (
          <>
            <header className="chat-head">
              <div className="chat-person">
                <span className="avatar">{(dashboard.user.role === "tutor" ? conversation.student_name : conversation.tutor_name).slice(0, 1)}</span>
                <span><h2>{dashboard.user.role === "tutor" ? conversation.student_name : conversation.tutor_name}</h2><p>Математика <i /> в сети</p></span>
              </div>
              <div className="chat-actions"><button className="icon-button" type="button" aria-label="Поиск в переписке"><Icon name="search" /></button><a className="button secondary" href="/calendar?view=day">Открыть урок</a></div>
            </header>
            <div className="lesson-note"><span className="soft-icon"><Icon name="calendar" /></span><span><strong>Следующее занятие</strong><small>Сегодня, 17:00 · материалы доступны внутри урока</small></span><a className="button secondary" href="/calendar?view=day">Открыть урок</a></div>
            <div className="message-feed">
              {loading ? <p className="empty">Загружаем сообщения...</p> : null}
              {messages.length === 0 && !loading ? <p className="empty">История пуста. Напишите первое сообщение.</p> : null}
              {messages.map((message) => (
                <div key={message.id} className={message.author_id === dashboard.user.id ? "bubble own" : "bubble"}>
                  <p>{message.body}</p>
                  <small>{formatTime(message.created_at)}</small>
                </div>
              ))}
            </div>
            {error ? <p className="field-error">{error}</p> : null}
            <form className="composer" onSubmit={submit}>
              <button type="button" className="icon-button" aria-label="Прикрепить файл"><Icon name="attach" /></button>
              <textarea value={body} onChange={(event) => setBody(event.target.value)} placeholder="Сообщение..." rows={1} />
              <button className="send-button" type="submit" aria-label="Отправить сообщение" disabled={body.trim() === ""}><Icon name="send" /></button>
            </form>
          </>
        ) : <p className="empty">Нет доступных диалогов.</p>}
      </section>
      {dashboard.user.role === "tutor" && conversation ? (
        <aside className="panel student-context">
          <h2>Об ученике</h2>
          <strong>{conversation.student_name}</strong>
          <div className="progress"><span style={{ width: "64%" }} /></div>
          <p>Ближайшее занятие и задания доступны в соседних разделах.</p>
          <a className="inline-link" href="/tasks">Открыть задания</a>
        </aside>
      ) : null}
    </section>
  );
}

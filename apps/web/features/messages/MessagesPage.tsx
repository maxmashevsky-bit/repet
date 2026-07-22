"use client";

import { FormEvent, useEffect, useMemo, useState } from "react";
import { useSearchParams } from "next/navigation";
import { productApi } from "../../shared/api/client";
import type { Message } from "../../shared/api/types";
import { formatTime } from "../../shared/lib/date";
import { useWorkspace } from "../app-shell/AppShell";

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
    <section className="messages-layout">
      <div className="page-title full"><h1>Сообщения</h1><p>Общайтесь и сохраняйте контекст занятий.</p></div>
      <aside className="panel dialog-list">
        <h2>Диалоги</h2>
        <input aria-label="Поиск по диалогам" placeholder="Поиск" />
        {dashboard.conversations.length === 0 ? <p className="empty">Диалоги появятся после подтверждения связи.</p> : dashboard.conversations.map((item) => (
          <button key={item.id} type="button" className={item.id === conversation?.id ? "dialog-row active" : "dialog-row"} onClick={() => setConversationID(item.id)}>
            <strong>{dashboard.user.role === "tutor" ? item.student_name : item.tutor_name}</strong>
            <span>{item.last_message || "Нет сообщений"}</span>
          </button>
        ))}
      </aside>
      <section className="panel chat-panel" aria-label="Активный чат">
        {conversation ? (
          <>
            <header className="chat-head">
              <div>
                <h2>{dashboard.user.role === "tutor" ? conversation.student_name : conversation.tutor_name}</h2>
                <p>В сети · следующий урок в календаре</p>
              </div>
              <a className="button secondary" href="/calendar?view=day">Открыть урок</a>
            </header>
            <div className="lesson-note">Следующее занятие: сегодня, 17:00 · материалы доступны внутри урока и сообщений.</div>
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
              <button type="button" className="icon-button" aria-label="Прикрепить файл">+</button>
              <textarea value={body} onChange={(event) => setBody(event.target.value)} placeholder="Сообщение..." rows={1} />
              <button type="submit" disabled={body.trim() === ""}>Отправить</button>
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


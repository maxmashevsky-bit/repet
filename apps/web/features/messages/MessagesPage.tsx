"use client";

import { useMemo, useState } from "react";
import { useSearchParams } from "next/navigation";
import Link from "next/link";
import { useWorkspace } from "../app-shell/AppShell";
import { Icon } from "../ui/Icon";
import { ConversationThread } from "./ConversationThread";
import { usePolling } from "./usePolling";

export function MessagesPage() {
  const { dashboard, refreshConversations } = useWorkspace();
  const search = useSearchParams();
  const firstConversation = dashboard.conversations[0]?.id ?? "";
  const [conversationID, setConversationID] = useState(search.get("conversation") ?? firstConversation);
  const [drafts, setDrafts] = useState<Record<string,string>>({});
  const [error, setError] = useState("");
  const [query, setQuery] = useState("");

  const conversation = useMemo(
    () => dashboard.conversations.find((item) => item.id === conversationID) ?? dashboard.conversations[0],
    [dashboard.conversations, conversationID]
  );
  usePolling(async signal=>{
    await refreshConversations(signal);
    if(!signal.aborted)setError("");
  },true,15000,err=>setError(err instanceof Error?err.message:"Не удалось обновить диалоги"));

  return (
    <section className={dashboard.user.role === "tutor" ? "messages-layout with-context" : "messages-layout"}>
      <div className="page-title full"><h1>Сообщения</h1><p>Общайтесь с преподавателем или учеником.</p></div>
      <aside className="panel dialog-list">
        {error?<p className="field-error" role="alert">{error}</p>:null}
        <div className="heading-with-count"><h2>Диалоги</h2><span>{dashboard.conversations.length}</span></div>
        <label className="search-field"><Icon name="search" size={20} /><input aria-label="Поиск по диалогам" placeholder="Поиск" value={query} onChange={(event) => setQuery(event.target.value)} /></label>
        {dashboard.conversations.length === 0 ? <p className="empty">Диалоги появятся после подтверждения связи.</p> : dashboard.conversations.filter((item) => `${item.tutor_name} ${item.student_name}`.toLowerCase().includes(query.toLowerCase())).map((item) => (
          <button key={item.id} type="button" title={dashboard.user.role === "tutor" ? item.student_name : item.tutor_name} className={item.id === conversation?.id ? "dialog-row active" : "dialog-row"} onClick={() => {setConversationID(item.id);setDrafts(previous=>item.id in previous?previous:{...previous,[item.id]:""});}}>
            <span className="avatar small">{(dashboard.user.role === "tutor" ? item.student_name : item.tutor_name).slice(0, 1)}</span>
            <span className="dialog-copy"><strong>{dashboard.user.role === "tutor" ? item.student_name : item.tutor_name}</strong><small>{item.last_message || "Нет сообщений"}</small></span>
            {item.unread_count?<em aria-label={`Непрочитанных сообщений: ${item.unread_count}`}>{item.unread_count>99?"99+":item.unread_count}</em>:null}
          </button>
        ))}
      </aside>
      <section className="panel chat-panel" aria-label="Активный чат">
        {conversation ? (
          dashboard.conversations.filter(item=>item.id===conversation.id||item.id in drafts).map(item=><div key={item.id} className="conversation-thread" hidden={item.id!==conversation.id}><ConversationThread active={item.id===conversation.id} conversation={item} body={drafts[item.id]??""} setBody={body=>setDrafts(previous=>({...previous,[item.id]:body}))}/></div>)
        ) : <p className="empty">Нет доступных диалогов.</p>}
      </section>
      {dashboard.user.role === "tutor" && conversation ? (
        <aside className="panel student-context">
          <h2>Об ученике</h2>
          <strong>{conversation.student_name}</strong>
          <p>Занятия и задания доступны в соседних разделах.</p>
          <Link className="inline-link" href="/tasks">Открыть задания</Link>
        </aside>
      ) : null}
    </section>
  );
}

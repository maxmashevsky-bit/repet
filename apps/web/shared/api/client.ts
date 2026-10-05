import type { Assignment, Conversation, Dashboard, Lesson, Message, NotificationItem, Role, User } from "./types";

const API_BASE = "";

type JsonBody = Record<string, unknown>;

function internalReturnTo(value: string | null): string {
  if (!value || !value.startsWith("/") || value.startsWith("//")) return "/";
  return value;
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const response = await fetch(`${API_BASE}${path}`, {
    ...init,
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      ...(init.headers ?? {})
    },
    cache: "no-store"
  });
  const payload = (await response.json().catch(() => ({}))) as unknown;
  if (!response.ok) {
    let message = "Не удалось выполнить запрос";
    if (typeof payload === "object" && payload && "error" in payload) {
      const err = payload.error;
      if (typeof err === "object" && err && "message" in err && typeof err.message === "string") {
        message = err.message;
      }
    }
    throw new Error(message);
  }
  return payload as T;
}

function post<T>(path: string, body: JsonBody, init: RequestInit = {}) {
  return request<T>(path, {
    ...init,
    method: "POST",
    body: JSON.stringify(body)
  });
}

export const authApi = {
  register(input: { displayName: string; email: string; password: string; role: Role; acceptTerms: boolean }) {
    return post<{ user: User }>("/api/v1/auth/register", {
      display_name: input.displayName,
      email: input.email,
      password: input.password,
      role: input.role,
      accept_terms: input.acceptTerms
    });
  },
  login(input: { email: string; password: string }) {
    return post<{ user: User }>("/api/v1/auth/login", input);
  },
  logout() {
    return post<{ status: string }>("/api/v1/auth/logout", {});
  },
  forgotPassword(email: string) {
    return post<{ status: string }>("/api/v1/auth/forgot-password", { email });
  },
  resetPassword(token: string, password: string) {
    return post<{ status: string }>("/api/v1/auth/reset-password", { token, password });
  },
  safeReturnTo: internalReturnTo
};

export const productApi = {
  me: () => request<User>("/api/v1/me"),
  dashboard: () => request<Dashboard>("/api/v1/dashboard"),
  updateProfile: (input: { displayName: string; timezone: string; locale: string }) =>
    post<User>("/api/v1/profile", {
      display_name: input.displayName,
      timezone: input.timezone,
      locale: input.locale
    }),
  createInvitation: (studentEmail: string, studentName: string) =>
    post<{ token: string; accept_url: string }>("/api/v1/invitations", {
      student_email: studentEmail,
      student_name: studentName
    }),
  acceptInvitation: (token: string) => post<{ id: string }>(`/api/v1/invitations/${encodeURIComponent(token)}/accept`, {}),
  createLesson: (input: { relationID: string; title: string; startsAt: string; endsAt: string }) =>
    post<Lesson>("/api/v1/lessons", {
      relation_id: input.relationID,
      title: input.title,
      starts_at: input.startsAt,
      ends_at: input.endsAt
    }),
  conversations: async () => (await request<{ conversations: Conversation[] }>("/api/v1/messages/conversations")).conversations,
  messages: async (conversationID: string, offset = 0) =>
    (await request<{ messages: Message[] }>(`/api/v1/messages/conversations/${conversationID}?offset=${offset}`)).messages,
  sendMessage: (conversationID: string, body: string) =>
    post<Message>(
      `/api/v1/messages/conversations/${conversationID}`,
      { body },
      { headers: { "Idempotency-Key": crypto.randomUUID() } }
    ),
  assignments: async () => (await request<{ assignments: Assignment[] }>("/api/v1/assignments")).assignments,
  createAssignment: (input: {
    studentID: string;
    title: string;
    subject: string;
    topic: string;
    body: string;
    dueAt: string;
    maxScore: number;
  }) =>
    post<Assignment>("/api/v1/assignments", {
      student_id: input.studentID,
      title: input.title,
      subject: input.subject,
      topic: input.topic,
      body: input.body,
      due_at: input.dueAt,
      max_score: input.maxScore
    }),
  submitAssignment: (assignmentID: string, answer: string) =>
    post<Assignment>(`/api/v1/assignments/${assignmentID}/submit`, { answer }),
  gradeAssignment: (assignmentID: string, score: number, feedback: string, revision: boolean) =>
    post<Assignment>(`/api/v1/assignments/${assignmentID}/grade`, { score, feedback, revision }),
  notifications: async () => (await request<{ notifications: NotificationItem[] }>("/api/v1/notifications")).notifications,
  markNotificationRead: (notificationID: string) => post<{ status: string }>(`/api/v1/notifications/${notificationID}/read`, {}),
  markAllNotificationsRead: () => post<{ status: string }>("/api/v1/notifications/mark-all-read", {})
};

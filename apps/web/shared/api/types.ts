export type Role = "tutor" | "student" | "admin";

export type User = {
  id: string;
  organization_id: string;
  email: string;
  display_name: string;
  role: Role;
  created_at: string;
};

export type Relation = {
  id: string;
  tutor_id: string;
  student_id: string;
  status: string;
  created_at: string;
};

export type Lesson = {
  id: string;
  relation_id: string;
  tutor_id: string;
  student_id: string;
  title: string;
  starts_at: string;
  ends_at: string;
  status: string;
  created_at: string;
};

export type Conversation = {
  id: string;
  tutor_id: string;
  tutor_name: string;
  student_id: string;
  student_name: string;
  last_message: string;
  unread_count: number;
  created_at: string;
};

export type Message = {
  id: string;
  conversation_id: string;
  author_id: string;
  body: string;
  created_at: string;
};

export type Assignment = {
  id: string;
  tutor_id: string;
  student_id: string;
  student_name: string;
  title: string;
  subject: string;
  topic: string;
  body: string;
  status: "draft" | "assigned" | "submitted" | "revision" | "done" | "overdue" | "archived";
  due_at: string;
  max_score: number;
  score?: number;
  feedback: string;
  answer: string;
  created_at: string;
};

export type NotificationItem = {
  id: string;
  title: string;
  body: string;
  href: string;
  read_at?: string;
  created_at: string;
};

export type Dashboard = {
  user: User;
  relations: Relation[];
  lessons: Lesson[];
  assignments: Assignment[];
  conversations: Conversation[];
  notifications: NotificationItem[];
};


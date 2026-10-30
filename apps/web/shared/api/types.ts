export type Role = "tutor" | "student" | "admin";

export type User = {
  id: string;
  organization_id: string;
  email: string;
  display_name: string;
  role: Role;
  timezone: string;
  locale: string;
  created_at: string;
  email_verified?: boolean;
  email_verification_required?: boolean;
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
  unread_count?: number;
  created_at: string;
};

export type Message = {
  attachment?: ChatAttachment;
  id: string;
  conversation_id: string;
  sequence?: number;
  author_id: string;
  body: string;
  created_at: string;
};

export type ChatFile = { id:string;filename:string;content_type:string;size_bytes:number };
export type ChatAttachment = { kind:"file"|"assignment"|"lesson";id?:string;title?:string;available:boolean;file?:ChatFile };
export type MessageAttachmentInput = { file_id?:string;assignment_id?:string;lesson_id?:string };

export type Assignment = {
  html?: {lesson_id:string;version_id:string};
  kind: "text" | "material";
  material?: {
    lesson_id:string; version_id:string; exercise_ids:string[];
    calendar_lesson_id:string|null; repeat_of:string|null; mode:"practice"|"exam";
    attempt:import("./learning").LearningAttempt|null;
  };
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

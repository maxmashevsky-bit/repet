# Карта 15 дизайн-референсов «Репет»

## Авторизация

| Файл | Маршрут/состояние | Главная проверка |
| --- | --- | --- |
| `screens/auth/01-registration-role-selection.png` | `/register`, частично `/login` | Выбор роли, форма, Google fallback, отсутствие product nav |

## Ученик

| Файл | Маршрут/состояние | Главная проверка |
| --- | --- | --- |
| `screens/student/01-main-dashboard.png` | `/` | Ближайший урок, задача, сообщение, прогресс |
| `screens/student/02-messages.png` | `/messages` | Диалоги, чат, вложения, composer |
| `screens/student/03-calendar-week.png` | `/calendar?view=week` | Недельная сетка и события |
| `screens/student/04-calendar-day.png` | `/calendar?view=day` | Дневная временная шкала |
| `screens/student/05-calendar-month.png` | `/calendar?view=month` | Месячная сетка и выбранная дата |
| `screens/student/06-tasks.png` | `/tasks` | Статусы, дедлайн, прогресс и продолжение |
| `screens/student/07-notifications.png` | Popover колокольчика | Unread и переход к ресурсу |
| `screens/student/08-settings-account.png` | `/settings` | Профиль, security, sessions, preferences |

## Преподаватель

| Файл | Маршрут/состояние | Главная проверка |
| --- | --- | --- |
| `screens/teacher/01-main-dashboard.png` | `/` | Несколько учеников, уроки, проверки, внимание |
| `screens/teacher/02-messages.png` | `/messages` | Список учеников, чат, контекст ученика |
| `screens/teacher/03-calendar-week.png` | `/calendar?view=week` | Недельная загрузка и filters учеников |
| `screens/teacher/04-calendar-day.png` | `/calendar?view=day` | Актуальная спокойная структура как у ученика |
| `screens/teacher/05-calendar-month.png` | `/calendar?view=month` | Месячная загрузка и сводка |
| `screens/teacher/06-tasks.png` | `/tasks` | Создание, назначение, проверка и просрочки |

## Общие правила

- Последние продуктовые решения и `PROMPT_FOR_CODEX_V2.md` приоритетнее старого PNG.
- Отдельный раздел «Материалы» запрещён.
- Footer «Поддержать создателя ♥» обязателен на полноэкранных страницах.
- Роль получает backend; смена query string не меняет доступ.
- PNG служат для сравнения, а не являются runtime-ассетами.
- Все видимые controls должны работать и иметь loading/error/disabled состояния.

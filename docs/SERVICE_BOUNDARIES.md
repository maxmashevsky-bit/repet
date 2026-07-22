# Границы сервисов «Репет»

Этот документ кратко дополняет мастер-промт и помогает не превратить микросервисную архитектуру в распределённый монолит.

## Page BFF

Page BFF владеет формой ответа страницы, но не доменными данными. Он может агрегировать сервисы, строить read model и кэшировать результат. Он не изменяет чужие таблицы и не реализует доменные переходы статусов.

| BFF | Потребители | Основные зависимости |
| --- | --- | --- |
| Auth | login/register | Identity, Profile |
| Dashboard | student/teacher home | Lessons, Schedule, Messaging, Assignments, Notifications, Relationships |
| Messages | `/messages` | Messaging, Profile, Relationships, Lessons, Assignments, Files |
| Calendar | `/calendar` | Schedule, Lessons, Relationships |
| Tasks | `/tasks` | Assignments, Profile, Relationships, Files, Notifications |
| Settings | `/settings` | Profile, Identity, Notifications |

## Domain services

Каждый domain service:

- является единственным владельцем своих таблиц;
- проверяет собственные инварианты и resource authorization;
- публикует versioned domain events через outbox;
- предоставляет узкие HTTP/event contracts;
- может быть развёрнут и масштабирован независимо;
- не импортирует внутренний код другого сервиса.

## Что не является микросервисом

- кнопка «Отправить»;
- React-компонент карточки;
- отдельный SQL query;
- formatter даты;
- getter/setter;
- функция без собственной бизнес-ответственности и данных.

Такие элементы остаются маленькими чистыми функциями или use cases внутри правильной границы. Это обеспечивает изменяемость без сетевого шума, дублирования данных и каскадных сбоев.

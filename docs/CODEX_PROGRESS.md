# Репет 2.0: прогресс разработки

Дата аудита: 2026-10-19 (Europe/Moscow). Это снимок локального состояния, не подтверждение production-готовности. Нумерация 00–19 введена этим аудитом; старый `EXECUTION_PLAN.md` — исторический общий план, а не источник актуальных статусов.

Статусы: `TODO` — работа не начата; `IN_PROGRESS` — есть частичная реализация или не закрыта проверка; `DONE` — выполнены все критерии соответствующего этапа в ROADMAP; `BLOCKED` — конкретное препятствие не позволяет закончить этап. Дата означает последнюю указанную проверку, а не срок завершения. Наличие кода не равно успешной проверке.

## Задачи

Не менять порядок столбцов: таблицу читает `scripts/repet-progress.sh`. Ровно одна строка на ID, без символа вертикальной черты внутри ячеек.

| ID | Название | Статус | Дата проверки | Результат | Блокеры |
| --- | --- | --- | --- | --- | --- |
| 00 | Аудит и система прогресса | DONE | 2026-10-19 | Аудит, 20 этапов, roadmap/workflow и скрипт созданы; позитивные и негативные проверки сводки прошли | Нет; ограничения продукта перечислены отдельно |
| 01 | Backend каталога учебных материалов | DONE | 2026-10-19 | Миграция 0005, модели/store, 25 HTTP-операций, поиск/ACL и закрытые ключи; API-сценарий PostgreSQL и race прошли, неизменяемость публикаций подтверждена | Нет по backend-критерию; ограничение нативного frontend lint описано ниже |
| 02 | Интерфейс библиотеки учебных материалов | DONE | 2026-10-19 | Каталог внутри Заданий, поиск/фильтры, страница урока, редактор, черновики/preview/публикация через API 01; browser E2E двух ролей прошёл | Нет по критерию 02; ограничения API и нативного lint описаны ниже; исторические проверки миграций сохранены |
| 03 | Универсальный движок упражнений | DONE | 2026-10-19 | Пять типов в общем renderer, строгий JSON, серверная проверка и ручная проверка решения; PostgreSQL/race и browser E2E двух ролей прошли | Нет по критерию 03; нативный lint ограничен локальным зависанием, контейнерный прошёл |
| 04 | Визуальные математические инструменты | DONE | 2026-10-19 | Пять визуальных типов, три режима geometry, строгие схемы, серверные ключи; unit/API/race и browser mouse/touch/keyboard прошли | Нет по критерию 04; нативный lint ограничен локальным зависанием, контейнерный прошёл |
| 05 | Прохождения и сохранение результатов | DONE | 2026-10-20 | learning_attempts и история exercise_attempts, серверные номера/время/проверка, подсказки и закреплённая версия; три ответа восстановлены после закрытия вкладки; PostgreSQL/race и 4 browser E2E прошли | Нет по новому критерию 05; нативный ESLint зависает, контейнерный lint прошёл; прежние auth-ограничения не закрыты |
| 06 | LearningPlayer | DONE | 2026-10-20 | Последовательные блоки, авторский пример, сохранение шага/ответов, тренировка и закрытая контрольная, допустимые разборы и результаты; PostgreSQL/race, 15 unit-групп и desktop/mobile browser E2E прошли | Нет по критерию 06; native ESLint зависает, Docker lint проходит; прежние browser-сценарии 03–05 не мигрированы на новый UI |
| 07 | Интерактивные домашние задания | DONE | 2026-10-29 | Назначение уроков/подборок и личные наборы, срок/связь с календарём, главная ученика, результаты/ошибки/попытки и отдельное повторение; PostgreSQL/race, 16 unit-групп и desktop/mobile E2E прошли; text сохранён | Нет по критерию 07; обычный git diff недоступен из-за файлового timeout индекса, проверка файлов без индекса прошла; прежняя приёмка профиля не закрыта |
| 08 | Импорт и безопасный просмотр HTML-уроков | DONE | 2026-10-29 | Загрузка/метаданные/классификация/карточки, отдельный opaque-origin preview, версии/оригинал/BOM, назначение snapshot и ручная конвертация; PostgreSQL/race/build/lint и 3 browser E2E прошли | Нет по критерию 08; ограничения legacy и хранения описаны в HTML_LESSONS; прежний календарный этап не закрыт сменой номера |
| 09 | PDF → интерактивный урок | DONE | 2026-10-29 | Приватный импорт/Poppler/очередь, исходник/права, текст/PNG/рисунки/crop, ручная сверка, native editor и revision review gate; PostgreSQL/race/unit/build/lint и desktop/mobile browser прошли,3 задачи решены | Нет по критерию09; OCR ручной, границы извлечения/карантина/хранения описаны в PDF_IMPORT; обычный Git index недоступен |
| 10 | Общий банк упражнений | DONE | 2026-10-30 | Пять категорий5–11, источники/права, фильтры/FTS/индексы/пагинация, immutable подборки и массовое назначение; текущие PostgreSQL/race/lint/build и реальный browser банка прошли | Нет по критерию10; только разрешённые авторские демонстрации, не официальный банк; Git index недоступен |
| 11 | Тренировки и пробные экзамены | DONE | 2026-10-30 | Версии программ/воспроизводимые варианты, серверный deadline, mobile навигация/сохранение/исправление/сдача, auto/manual результат, повтор ошибок/история; PostgreSQL/race/27unit/build/lint и browser прошли | Нет по критерию11; raw баллы, без официального перевода/частичного оценивания; expiry материализуется при чтении, поздние ответы запрещены БД |
| 12 | Аналитика успеваемости | DONE | 2026-10-30 | Панели двух ролей, темы/динамика/подсказки/домашние/пробники, sample size и pending отдельно; объяснимые рекомендации/конкретные ошибки и pinned повторение слабой темы; Go/race/29unit/lint/build/mobile browser прошли | Нет по критерию12; определения/границы метрик в LEARNING_ANALYTICS; без LLM, психометрических и официальных экзаменационных баллов |
| 13 | Повторение и достижения | DONE | 2026-10-30 | Интервалы1/3/7/14/30, private source-granted квизы/продолжение, ошибки/освоение/недельные цели, XP ledger/бейджи и off/on; полные Go/race/31unit/lint/TS/build, миграции и real browser прошли | Нет по критерию13; эвристика памяти/освоения, консервативный XP только за подтверждённое самостоятельное повторение, legacy HTML вне статистики |
| 14 | Родительский доступ | DONE | 2026-10-30 | Отдельный подтверждённый/отзывной grant, календарь/домашние/результаты/пробники/графики/разрешённые notes/события без личного чата; Go/race/33unit/lint/build/PostgreSQL и mobile browser прошли | Нет по критерию14; техническое подтверждение по личной ссылке, не независимая проверка родства/email; event feed без push/email, пределы в FAMILY_ACCESS |
| 15 | Мессенджер, файлы и уведомления | DONE | 2026-10-30 | 0020polling/read/unread/retry и0021private PNG/JPG/TXT CDR/quota/ACL, assignment preview/native lesson links, submit/grade notification deep links; полный Go/race/lint/build и browser двух клиентов прошли | Нет по критерию15; узкая allowlist, не универсальный AV, storage/offline ограничения в MESSAGING/ADR0004 |
| 16 | Независимые преподаватели | IN_PROGRESS | 2026-10-30 | 0023preferences/UI и закрытие standalone leaks; дополнительно0025backend pinned sharing/revoke/server-copy/provenance/idempotence, focused PostgreSQL/full Go/lint PASS | Sharing UI, race025/OpenAPI, private visibility и полная матрица изоляции ещё нужны; preview пока0024; не DONE |
| 17 | Подготовка mini-PC | IN_PROGRESS | 2026-10-30 | По прямому запросу: общий Dockerfile с3целями, production Compose/Caddy/secrets/health/resources, SMTP verification, архив и backup/restore; локальные сборки и roundtrip проверены | Публичный запуск не выполнялся; нужны домены/SMTP владельца, реальный TLS/delivery, установка расписания копий/off-site/мониторинга и финальный security18; не DONE |
| 18 | Сквозная приёмка MVP | TODO | 2026-10-29 | Новый критерий очереди 08–19 сохранён | Сначала реализовать критические сценарии; затем MVP_READINESS с реальными блокерами |
| 19 | Устойчивость и план расширения | TODO | 2026-10-29 | Платежи и подписки не реализуются | Нужна оценка после 00–18, зависимости, затраты и критерии запуска |

## Доказательства текущего аудита

### Прямой Запрос: Сервер, Почта И Интерфейс — 2026-10-30

Это приоритетный запрос владельца поверх очереди, не объявление завершения16–19. Общий Dockerfile api/web/pdf, standalone Next, закрытый PostgreSQL, изолированный HTML host, Caddy HTTPS, file secrets, лимиты/health/log rotation. `scripts/package-server.sh` собирает переносимый source archive без secrets/БД/private files/runtime; команды и ограничения в SERVER_LAUNCH.md. Backup/restore скрипты проверены на отдельном локальном Compose project `repet-release-checks`: реальная строка БД и удалённый после backup приватный файл восстановились, SHA256 файла совпал `412bd74052c859ece6f5c4a66e533f30bcef287e4a5f264cdba256c3469cf661`. Копия только synthetic в ignored backups/20261030T085513Z, не пользовательская БД.

0024email verification: хеши токенов, DB clock/час, bound user+email, row-lock/cooldown60с, replay до expiry; production запрещает отключение, STARTTLS обязателен. GETme/logout/resend/confirm доступны до подтверждения, остальные authenticated API403email_unverified. Register201 сохраняет аккаунт при SMTP failure, не утверждает доставку; /verify-email UI умеет resend/ошибку/confirmation, login учитывает persisted flag. SMTP credentials и настоящая доставка владельцу не настраивались.

Единый Select заменил native popup во всех21React-модулях: собственный listbox/portal в dialog, keyboard/typeahead/touch, inset arrow16px, реальные select для FormData/required и совместимости. Оригинальные загруженные HTML не переписывались. Недельный календарь: общая сетка72px+7колонок, sticky header62px, реальные даты/today, старт08:00, собственный bounded scroll; устранён collision со стилями недели dashboard. Mobile320/375/768/1440, no horizontal page overflow, header/cells delta<1px; screenshots просмотрены.

Проверки: `make test` fresh PostgreSQL `repet_release_final_checks` PASS HTTP75,088с/store9,292с; первый запуск до создания БД и первый verification/login regression падали, исправлены и полный прогон повторён. Focused verification/SMTP PASS2,567с; `go test -race` verification/SMTP fresh `repet_release_race` PASS15,614с. `make lint` nativeGo + frontend current image/tests PASS, 5 прежних img warnings; отдельный tsc PASS.36frontendunitPASS/0SKIP9,586с. Корневые Docker targets API/PDF/web собраны с официальных баз и frozen lock; standalone web Next196,6с/TS24,4с PASS (исправлены отсутствующий public и storage mode0700). Compose config PASS, Caddy validate PASS,15OpenAPI YAML/481refs PASS (не full validator).0001–0024up/empty24down-up PASS, populated24down expected exit3.

Финальный browser текущих3020+новогоAPI: integrated tools32,336с + release UI10,951с,2PASS32,836с, no JS errors. Email browser на root standalone3021 + собственный Mailpit18125: обе роли register→mail→blocked403→confirm→dashboard200→logout/login, PASS6,699с; ранние test-locator ожидания скрытого mobile navigation/неверного relation ID исправлены, не product ACL. Source/current web image digest **f8d6496124d3e24e4db7d4a3837e2df46e2bc2c464f072a21ce2cd3ff3601f49** совпал. Критерии публичного запуска НЕ подтверждены: нет DNS/серверных реквизитов/реального SMTP,16 и18 не завершены. Git index не ремонтировался. Финальный package/whitespace/shell итог указан в checkpoint.

### Задача16: Первый Срез — IN_PROGRESS, 2026-10-30

Teaching preferences/0023 GET+PUT, собственная session scope, CAS/валидаторы, UI «Преподавание». Public автор не видит чужих standalone attempts и не получает personal manual notification; manual требует active relation перед записью, review после revoke404. Подробные условия/оставшиеся шаги в TEACHER_ISOLATION.md. Никаких новых grants только по факту авторства/роли, ученических аккаунтов или организационных прав от клиента.

Проверки: focused isolation2,854с/exercise6,875с; focused race обеих HTTP35,915с exit0. Полный make test в новой repet_tenants16_final_checks HTTP28,861/store2,414с exit0, cached остальных явно. Первый full run упал на старом len=1 assertion ChatAttachments при наличии других разрешённых public fixtures; поиск стал уникальным, финальный fresh прогон прошёл. make lint nativeGo/current Docker source+tests exit0,5прежних imgwarnings. ProductionNext79,2/TS19,0сPASS, digest host/image2aaac635097872d5a4fac6221d660440eb97940f74019a72762c51748165eded. Browser2PASS44,880с: основной integration36,387/preferences7,248с,320–1440/nooverflow/noJSerrors, screenshots просмотрены.0001–0023up,23emptydown/up PASS; populatedguardexpected3.15YAML475refsresolvedRuby.

Preview3020 теперь repet-tenants16-web/api в прежних собственных containers repet-messages15-web/api, прежняя development DB/volumes сохранены; пользовательский volume не изменён. Checks/final_checks/race/migrations DB repet_tenants16_* только synthetic.17 не начата.16 не DONE: private/sharing/copy/licensing и полная resource isolation matrix ещё нужны. Эти проверки не подтверждают будущие отсутствующие функции.

Финальный визуальный проход исправил inherited padding/min-height checkbox и перенос чисел10/11: browser assert nowrap+реальные label bounds320–1440. FinalNext48,4с/TS15,9сPASS; image repet-tenants16-web-final (alias repet-tenants16-web) соответствует source digest de9ac0efac443053c14f5391088431190f81d4dc10693bc8cc8dfb7f66553b61. Finalpreferences browser12,501сPASS, screenshots просмотрены.36unitPASS0SKIP1,457с, final make lint exit0/5прежних warnings, final make test exit0cached.21files no-indexwhitespacePASS;20строк/16DONE/1IN_PROGRESS/3TODO. Длительных проверочных процессов не осталось. Обычныйgitdiffcheck128 из-за прежнего index timeout, индекс не изменялся.

### Интеграция Инструментов И Завершение15 — 2026-10-30

Последний прямой запрос: сделать доступными перечисленные функции в одном кабинете. На главной преподавателя8явных переходов: план/библиотека/PDF/банк/пробники/успеваемость/повторение/семья; верхняя навигация по-прежнему4раздела. На320px сокращено место декоративной стрелки, чтобы длинные названия не дробились на отдельные буквы.

Planner: relation_id в карточке — только действующая собственная связь, не browser-owned grants; миграция0022 материализует типовую неделю в обычные lessons и уведомления, source slot/date дедупликация + payload nonce receipt, server timezone, pause/unlinked counters, overlap/stale409/rollback. Существующие calendar lessons не меняются после правок плана. Публичных приглашений без согласия и автоматического создания аккаунтов нет. Мастерская получила два действия импорта/конвертации; исходник по allowlisted slug только преподавателю, оригинал сохраняется в HTML version, native draft требует авторской сверки/упражнений/ключей. Старые JS/localStorage scores не импортируются. Новый native ответ после конвертации вошёл в реальную аналитику:1attempt/1correct.

Chat:0021/private ObjectStorage, проверка/reconstruction PNG/JPEG, строгий TXT, source/canonical4МиБ/4Мпикселя и50МиБ/100owner quota, upload/send nonce, expiry/cancel, download attachment/nosniff/no-store/sandbox, no keys. Only active participant; pending uploader-only, recipient после send, чужая переписка/peer/admin/guardian без доступа. Assignment exactly samepair/notdraft/archive; lesson обеим сторонам publishedcourse/ancestors, no assignment_only/private other tutor. Preview allowlist без answers/feedback/answerkeys. Архивирование скрывает metadata ссылки. Новая TextAssignmentPage обслуживает `/tasks/{id}` и прежние `/tasks?task={id}`, submission→teacher notification→grade→student notification/result проверены в браузере.

| Проверка | Результат |
| --- | --- |
| `TEST_DATABASE_URL=…repet_integration_checks_v2 make test` | exit0: HTTP28,632с/store3,350с, PostgreSQL выполнен; финальный повтор exit0 cached явно |
| Четыре focused HTTP planner/calendar/chat/read | exit0,3,382с; chat с MaxConns1, ранее2,228с |
| `go test -race ./... -count=1 -timeout=5m`, отдельная repet_integration_race | exit0, HTTP150,030с/store18,554с, остальные пакеты также PASS |
| `make lint PNPM=…/tmp/integrated-pnpm` | exit0, native vet/gofmt и контейнерный текущий frontend+tests;0errors/5прежних img warnings |
| Frontend unit в образе | 36PASS/0SKIP,2,452с; финальный текущий образ+tests36PASS/0SKIP,1,140с |
| Production Next/TypeScript, final repet-integrated-web | exit0, RUN77,1с/TypeScript25,8с. Host/image source digest c3a965645bfba34e8a2eb0314a0d6a52aa119680adcd71b624103f8c134a51b2 совпал |
| `integrated-tools.e2e.mjs` в финальном образе | exit0,34,182с:8входов, account→planner→calendar, private file→download/foreign404, assignment preview/lesson, submit/notification/grade/result, workshop→native publish→player→analytics;320/375/768/1440 без overflow/JSerrors |
| Последовательная browser-регрессия10разделов | 10PASS/0SKIP,126,541с: analytics/bank/family/HTML/material/messages/PDF/reviews/study/planner. Предыдущий запуск9PASS/1FAIL был отсутствующим dedicated fixture DSN для reviews, не принят за PASS; отдельный повтор reviews11,432с и полный повтор успешны |
| PostgreSQL миграции |0001–0022 up, пустые22down/21down/21up/22up exit0; populated21/22down ожидаемоexit3 с export guard |
| Контракты |14YAML/471local refs разрешены Ruby, включая external exercise JSON schema; не полный OpenAPI-linter |
| Финальная сводка/whitespace |20строк/16DONE/4TODO, exit0;50файлов no-index whitespace exit0. Обычный git diff --check недоступен из-за старого timeout индекса; индекс не изменялся |

Скриншоты tmp/integrated-tools (home/planner/chat), включая mobile320, просмотрены; old planner/HTML/PDF/browser артефакты также обновлены тестами. Нет публичного deployment/push/PR/платежей/AI-вызовов/production secrets.

Docker был остановлен, прежний disposable PG15437 исчез. Создан собственный `repet-integration-postgres` (postgres18), loopback15437, named volume `repet-integration-postgres-data`; preview DB `repet_integration_preview`. Собраны `repet-integrated-api`/`repet-integrated-web`; только собственные preview `repet-messages15-api/web` заменены, URL3020/API18108/HTML18109, object volume15 сохранён; API подключён к task07/pdf-private networks с прежним extractor. **Пользовательский том repet_postgres-data и исходные аккаунты других баз не менялись/не импортировались**, preview — отдельная база. Checks_v2/race/migrations — только synthetic и сохранены до очистки владельцем. Git index по-прежнему128; не чинить reset/rebuild, whitespace проверять no-index. Задачи16–19 этим запросом не реализованы; текущая функциональная интеграция не объявляет MVP/public readiness.

### История: Задача15, Первый Шаг — 2026-10-30

0020message sequence/cursors, обратная миграция с export guard. Упорядочивание под row lock диалога и DB timestamp, unread по реальным чужим строкам после server cursor; late-commit не теряется. Новый POST conversations/{id}/read принимает только through_message_id; чужие/future IDs404, client sequence/user/time отклонены. Monotone cursor/DB participant guard. Чат теперь только actual participant с активной связью/организацией, не global admin/guardian. Send replay сохраняет ID/sequence, изменённый body409; лимит4000Unicode вместо прежнего byte-count для русского текста.

Scheduler с явным окружением, React adapter, poll5s/15s/20s active chat/conversations/notifications, visibility abort/pause, no overlap, backoff≤60s, cleanup. Merge ID/sequence, per-dialog drafts, скрытые уже использованные threads сохраняют failed-send/nonce, но не poll/read. Чтение после layout только visible/focused/bottom; stale conversation/notification response не отменяет новое обновление. Существующий AppShell/NotificationsPopover/API client переиспользованы. Пока только текстовый контур: ADR0004 прочитан, files/assignment preview/lesson attachments ещё не реализованы. Полные ограничения и следующий шаг в MESSAGING.md.

Проверки foundation: make test PostgreSQL repet_messages15_checks exit0 HTTP24,199/store1,703с; финальный после Unicode-boundary additions exit0 HTTP14,010с, остальные cached. Fullfresh race repet_messages15_race exit0 HTTP58,926/store9,422с; focused Unicode/read HTTP1,845/race4,413с PASS. TestMessageReadsPostgres: поздний commit после read, smaller ack не откатывает cursor, body replay/conflict, own/incoming unread, foreign/admin/ended404, direct DB foreign cursor отказ,4000ЖPASS/4001Ж400.36unitPASS0SKIP2,325с, make lint nativeGo + current-image ESLint exit0,5прежних img warnings. Синхронный effect lint исправлен правильным layout-frame read. Final productionNext33,9с/TS8,7с PASS, host/image digest a4b52ee51d67b352e14a3da4095e343f3b3d4594e8e007d757671f18f498a4fd.

Browser foundation current source28,295с PASS: две сессии без reload, потерянный ответ→retry→одна server row, failed bubble переживает переключение, drafts разделены, unread другого диалога и bell. Финальный screenshot-run26,172с PASS на320/375/768/1440px,0overflow/0JSerrors/header overlap; screenshots просмотрены. Instant scroll устранил артефакт capture из промежуточной позиции CSS smooth scroll; DOM содержит одну оболочку/header.0020emptydown/up,0001–0020up и backfill двух настоящих pre0020messages PASS (seq1/2, conversation counter2); populateddownexpected3. YAML11files442refsresolvedRuby;24files whitespacePASS. Preview3020/messages, web repet-messages15-web из repet-messages15-web-current; API repet-messages15-api из repet-messages15-api на18108/HTML18109, DB repet_messages15_preview/volume repet-messages15-objects.08–14previews не менялись. Все проверки/exec sessions завершены;15 не DONE, attachments ещё нет.

### Задача14 завершена — 2026-10-30

0019guardian_links/guardian_notes/guardian_events,10HTTPопераций, отдельный академический allowlist report без расширения глобальных ролей/старых product ACL. Ученик явно приглашает, секрет256бит hashed и только fragment+POSTbody; адресат подтверждает в email-bound аккаунте, участники могут отозвать. Собственные новые server-owned поля не принимаются. После revocation родитель не видит ни отчёт, ни read, ни меняющиеся имена через links. Ресурсная/организационная изоляция, pending/active/revoked/expiry, DB immutable invite и idempotent accept/revoke/note. Родитель A не видит ребёнка B при двух реально подтверждённых семьях. Старые chat/lesson/assignment/analytics endpoints по-прежнему404 для чужого student actor с guardian grant.

`/family`, `/family/[studentId]`, `/family-invite`; вход через Настройки→Семья, четыре topnav сохранены. Самостоятельное согласие ученика/приглашение, кабинет родителя с периодом/7пагинациями/графиком и sample size, отдельные pending. Только опубликованные для родителей комментарии активного преподавателя, withdrawal; обычный private feedback не открывается. Dedicated academic event feed с unread/server read watermark без детских сообщений/уведомлений. Активный пробник не виден через metrics/results/exams; после сдачи только реальные raw points. Общие SQL facts/метрики и DailyChart12 переиспользованы, без impersonation и обхода grant. Ограничения идентификации/доставки в FAMILY_ACCESS.md.

Финальный `make test` PostgreSQL repet_family14_checks exit0 HTTP22,834с, остальные пакеты cached отмечены; предыдущий полный13,750/store1,579с. Полный fresh race repet_family14_race exit0 HTTP72,416/store8,096с; после ужесточения links focused HTTP1,708с и race6,898с exit0. make lint native Go + Docker current frontend/tests exit0,0errors/5прежних img warnings.33unitPASS0SKIP3,281с. Final Next production38,2с/TS9,8с PASS, digest host/imageed0163db8fa51bcf5a1acc27e50216a905b9eafe6bf4b3c0d8e31eeb708a576d.

Final real Chrome family E2E10,359с PASS: student consent UI→email-bound parent→academic fixtures/native graded+pending→teacher explicit note→schedule/homework/results/graph/comments/events/read→withdraw→student revoke→parent404.320/375/768/1440px без overflow/JS errors, screenshots просмотрены; token отсутствует в request URLs. Сбои тестовых selector Next announcer/read-busy ожидания исправлены до итогового PASS; Node unit import с расширением .ts приведён к существующему паттерну. Browser analytics+study2PASS71,719с на API14.0019emptydown/up и0001–0019up PASS, populateddown expected3/export guard;11YAML440refs resolvedRuby (не полный validator);31file whitespace PASS без индекса. Preview3019, web repet-family14-web из repet-family14-web-complete, API repet-family14-api из repet-family14-api-complete18106/HTML18107, DB repet_family14_preview/volume repet-family14-objects. Старые08–13preview сохранены. Все exec checks14 завершены, push/deploy не делались.

### Задача13 завершена — 2026-10-30

0018/reviews domain/store/API и frontend `/tasks/review`, `/tasks/review/[sessionId]`, `/tasks/review/check`. Повторение по latest logical source с интервалами1/3/7/14/30 и source-run grant, неизменяемый выбор упражнений/версии, отдельные unanswered/pending, проверка исходного доступа после revocation. Нельзя получить другие задания закрытой банковской подборки, подменить преподавателя или раскрыть active exam. Native player/progress/hints/manual review переиспользованы; ручная очередь только контекстному преподавателю. Пагинация плана и active независима. Frontend ошибок/цели3внеделю/реального освоения/XP/бейджей/off-on, reduced-motion и mobile. Правила и API в LEARNING_REWARDS.md;4новых пути/4операции learning-reviews.yaml.

Полный `make test` PostgreSQL repet_review13_checks exit0 HTTP29,737/store1,909с; fresh full race repet_review13_race exit0 HTTP54,980/store10,974с. После последних дополнительных paging/filter тестов focused HTTP1,668/store0,721с и focused race HTTP6,820/store2,167с exit0. MaxConns1 доказывает отсутствие pool starvation при старте квиза. Интервалы проверены реальным SQL по границам1/3/7/14/30; автор shared source не видит private manual review другого преподавателя, bankcopy/original не дублирует XP, настройки двух контекстов независимы. Нативные Go vet/gofmt + Docker current-source ESLint `make lint PNPM=.../tmp/review13-pnpm` exit0,0errors/5прежних img warnings.31unit PASS0SKIP2,832с. Production Next/TS построены в repet-review13-web-complete, отдельный `npm run typecheck` exit0; digest host/image23273ea717b9695c10513f06bcae2f5c4805b339ebec571bad60d08495906a08.

Real Chrome review E2E exit0,12,431с: private4из5→три ошибки→mobile квиз→закрытие вкладки→resume2из3→завершение→same-day0XP→серверный fixture48ч→30XP/3бейджа→tutor off/on→manual quiz→review→correct.320/375/768/1440px,0overflow/JS errors, финальные screenshots просмотрены. Найденная controlled checkbox race исправлена; второй сбой был selector теста, не продукт. Регрессионные browser analytics+study2PASS10,589с на текущем13API, включая20ответов/повторение темы/контрольную/ручную проверку/историю.

0001–0018up и0018empty down/up exit0; populateddown expected3/export guard.10YAML/408refs разрешены Ruby (не полный validator).45files whitespace PASS без индекса. Preview13 http://127.0.0.1:3018/tasks/review: repet-review13-web/api из *-complete, API18104/HTML18105, DB repet_review13_preview, volume repet-review13-objects. Только собственные13контейнеры заменены;08–12preview сохранены. Git index timeout не исправлялся, push/deploy не выполнялись.

### Начало задачи13 — 2026-10-30, IN_PROGRESS

0017/domain/store/httpapi rewards foundation,3операции на2paths, опциональная настройка контекста преподаватель→ученик и неизменяемый XP ledger. Реальное завершённое самостоятельное первое решение без hints на двух UTC-днях с≥24часа, только numeric/text/manual-correct long_text;10XP один раз за logical source. Выбор вариантов/визуальные клики, ошибка→исправление и replay не награждаются. Авторский bank provenance дедуплицирует исходники. Уровни/неделя UTC вычисляются сервером; off скрывает контекст, повторный PUT не сдвигает время, reenable сохраняет старые достижения без нового backfill за disabled interval. API не принимает XP/время/роль от браузера. Подробности и ограничения в LEARNING_REWARDS.md, контракт learning-rewards.yaml.

Проверено на отдельной synthetic repet_rewards13_checks: focused rewards HTTP2,124/store1,380с exit0; затем полный make test exit0, HTTP10,473/store1,086с (другие пакеты cached явно). Focused race на отдельной repet_rewards13_race exit0 HTTP5,622/store1,864с. make lint exit0: native vet/gofmt, прежний frontend-complete ESLint с текущими tests0ошибок/5img warnings; frontend13 не менялся и не выдаётся за готовый. Unit уровней0/10/90/100/210XP PASS. HTTP:0XP в один день→синтетически перенесённая серверная fixture первая дата≥48ч→только clean numeric10XP; повторGET остаётся10; pending manual не награждён, после двух реальных tutor-review20XP; выбор/hints/ошибки→исправление всё ещё исключены. Immutable ledger update запрещён; peer/foreign404, student PUT403, server-owned points400; off0/hidden, PUT replay same timestamp, on20 сохранены, revoked tutor404. Первоначальный сбой был только cast $2 timestamptz в fixture, исправлен перед успешными проверками.

0001–0017 up,0017empty down/up exit0; populated down ожидаемо3/export guard. YAML2paths/3ops118refs rewards+compatibility разрешены Ruby, не полный validator. **Это не DONE13:** ещё нужны интервальная очередь, короткие квизы по source-run grant, личная работа над ошибками/освоение/награды/недельные цели в UI и browser-проверки. Нельзя создавать student quiz под подставленным tutor actor. Рабочий preview остаётся12 на3017/API0001–0016; текущий backend source13 не запущен в этом preview.

### Задача 12: аналитика — 2026-10-30

Миграция0016 добавляет индексы, не новую параллельную систему результатов.3APIоперации и `/tasks/progress` внутри Заданий: teacher roster, период/пагинация тем и пробников, сводка/дневной график с таблицей/темы/домашние/pending/история и до5 объяснимых рекомендаций с конкретными упражнениями. Tutor видит только активного ученика и свои assignment/spec/course контексты; student только себя. Read-only repeatable snapshot после server expiry. Повторение выбирает только последние incorrect одной темы/версии и использует существующую транзакцию native assignment/уведомления, nonce replay/conflict.

| Проверка | Фактический результат |
| --- | --- |
| Финальный TEST_DATABASE_URL=…/repet_analytics12_checks make test | exit0, HTTP24,914/store1,878с, PostgreSQL не SKIP; cached прочие пакеты не выданы за новые |
| Свежий go test -race ./... -count=1, новая repet_analytics12_final_race | exit0, HTTP49,778/store12,238с. Предыдущий полный race57,000/9,743с; focused после списка ошибок13,052/2,072с — исторические промежуточные, финальный прогон новее |
| make lint PNPM='sh …/tmp/analytics12-pnpm' | exit0, native vet/gofmt + Docker ESLint на current source/tests,0ошибок/5прежних img warnings; не native pnpm |
| Production Next build --webpack + TypeScript | exit0, RUN35,7с/TS8,4с; repet-analytics12-web-complete. Host/image app source digest51bd1871f67c3183d49566de66fbd7d74af47c6ed8c3b62b09b024d011a94c52 совпал |
| Frontend units в complete image с актуальными tests | exit0,29PASS/0SKIP/2,236с; runtime analytics/периоды и прежние27групп |
| Final analytics + study real browser / Chrome | exit0,2PASS/0SKIP/16,834с; analytics8,245с, study7,688с. Новый API-complete/frontend-complete с миграциями0001–0016, отдельная synthetic preview DB |
| Миграции/контракт/whitespace | 0001–0016 up и0016down/up exit0; migration DB удалена. YAML3analytics paths/3ops и115refs analytics+compatibility разрешены Ruby, не полный OpenAPI validator.24файла no-index whitespace PASS |

Проверенные endpoints: `GET learning-analytics/students`, `GET learning-analytics/{studentID}`, `POST learning-analytics/{studentID}/repeat`. HTTP fixture:20оценённых=15верных/5неверных→75%,1manual pending отдельно; hints1; повтор nonce не увеличивает отправки. Synthetic даты20ответов проверяют две половины:0%→100%, по5оценок,3дневных точки. Пустой период даёт nullaccuracy; неизвестные/повторные параметры/неверные даты400. Foreign/student/неактивная связь404/403, два tutor одного student видят20и1своих оценённых ответов, student21. Повторение pinned только5ошибок, replayтотжеID, изменённый payload409; чужой run закрыт. Study integration дополнительно подтверждает, что active exam не раскрывает accuracy/ошибки/баллы, после сдачи/ручной проверки raw5/7 и3оценённых ответа корректны, bank provenance тема сохранена.

Browser:20objective+1pending → tutor UI75%/слабая и сильная темы/пять условий ошибок → назначение пяти ошибок → student UI/таблица/пустой период;320/375/768/1440, длинное имя,0page errors/0overflow. Скриншоты final просмотрены. Повторён полный study UI с edit/resume/manual/repeat/history. Первый analytics browser17,052с также PASS, но предшествовал финальным подписям/списку ошибок; первый SQL alias day и два неверных тестовых пути исправлены до успешных проверок, не скрытые PASS.

Preview12 http://127.0.0.1:3017/tasks/progress: repet-analytics12-web из repet-analytics12-web-complete, API repet-analytics12-api из repet-analytics12-api-complete (:18102/HTML:18103), synthetic repet_analytics12_preview, volume repet-analytics12-objects. Старые preview не заменены. Все проверочные процессы завершены; Git index128 timeout сохраняется, индекс не исправлялся. Запрет push/PR/deployment соблюдён. Подробные определения/ограничения: LEARNING_ANALYTICS.md; рекомендации не являются диагностикой и не включают ещё непроверенные решения.

### Приёмка 10–11 — 2026-10-30

Завершена отложенная приёмка10 и полный цикл11 на текущем backend. Добавлены три study routes внутри Заданий, создание/публикация версионируемой программы, отдельный exam player с серверным временем, восстановлением/редактированием ответов и нейтральным статусом до сдачи, результаты/repeat ошибок/история и ручная очередь tutor. Practice переиспользует LearningPlayer. БД фиксирует snapshot/выбранные упражнения/баллы/срок; DB-trigger отклоняет поздний ответ. Выбор SHA256 детерминирован, логические дубликаты из разных версий исключены. Состояние и очередь tutor материализуют expiry. Календарные lessons не заменены, старые назначения07 сохранены. Ограничения в STUDY_SESSIONS.md.

| Проверка | Фактический результат |
| --- | --- |
| TEST_DATABASE_URL=…/repet_study11_checks make test | exit0, HTTP11,344/store2,148с; PostgreSQL не SKIP. Focused TestStudy отдельно HTTP2,322/store1,264с |
| go test -race ./... -count=1 с отдельной repet_study11_race | exit0, HTTP37,497/store8,442с; все пакеты выполнены заново |
| make lint PNPM='sh …/tmp/study11-pnpm' | exit0, native vet/gofmt и ESLint final Docker source с актуальными tests;0ошибок/5прежних img warnings, не native pnpm |
| Production Next build --webpack + TypeScript | exit0, RUN58,2с, TypeScript11,3с; repet-study11-web-final. Host/image app source digest одинаков db6322aa1439d2b7763647cd3aa8b7fdd5168606f56555038ad561f081b47495 |
| npm run test:exercise-engine в final image | exit0,27PASS/0SKIP,1,500с; runtime/clock/repeat, engine/player/visual/bank/calendar/PDF/planner/workshop |
| exercise-bank/material-assignments/html-lessons/pdf-import real browser на API11 | exit0,4PASS/0SKIP,59,604с; bank11,885с, assignments13,471с, HTML11,939с, PDF20,243с. Изолированная synthetic preview DB |
| Финальный study.e2e.mjs / Chrome на final image | exit0,1PASS/0SKIP,15,159с. Tutor через UI создаёт и публикует программу; student375 сохраняет, исправляет, закрывает/продолжает и сдаёт; ручная проверка меняет2/7 на5/7; повтор1/1 и история2; peer404, private keys не раскрыты. Clock/overflow также320px;0JS errors; screenshots просмотрены |
| Миграции/контракт/whitespace | 0001–0015 up и0015 empty down/up exit0; populated down ожидаемо exit3 с export guard. Ruby YAML7study paths/9operations,127refs study+compatibility разрешены (не полный OpenAPI validator).37затронутых файлов no-index whitespace PASS |

HTTP integration подтверждает teacher/student/foreign ACL, draft visibility, nonce replay/conflict, закреплённую версию/порядок/баллы, скрытые результаты, resave, запрет поздней записи и expiry из tutor queue, latest-only manual review, новые версии без изменения старого прохождения. Временной deadline в тесте сдвигается только test-fixture транзакцией; реальное ожидание5мин в browser не заявляется. Промежуточный expected404 на закрытом answer-keys исправлен на фактический403, затем полный прогон PASS. Node js-yaml недоступен; успешная проверка refs выполнена Ruby, не скрытый PASS первого вызова.

Новый loopback preview http://127.0.0.1:3016/tasks/study: repet-study11-web (final image), repet-study11-api (:18100/HTML:18101), synthetic repet_study11_preview и volume repet-study11-objects. Старые preview, включая3015, не заменены. Нет push/PR/deployment. Прежние audit security findings не закрыты этими номерами.

### Отдельный запрос: визуальная полировка frontend — 2026-10-29

Выполнен, не новый номер roadmap. Общая сетка/отступы/типографика, ограничение длинных строк, адаптивные формы и карточки, сообщения/composer, уведомления, auth и профессиональные подписи приведены к общему стилю. Общая тема всех20HTML-уроков обновлена без переписывания исходников. Пересечения календаря разнесены по колонкам; дневной календарь помещается на телефоне; мобильные фильтры библиотеки/банка сворачиваются. Исправлен обнаруженный браузером overflow review-card, растягивавший страницу заданий. Продуктовые функции11 в этом проходе не добавлялись.

| Проверка | Фактический результат |
| --- | --- |
| TEST_DATABASE_URL=…/repet_frontend_polish_test make test | exit0, HTTP16,603/store3,978с, PostgreSQL не SKIP; регрессия текущих существующих сценариев, не полная приёмка11 |
| Финальный make lint PNPM='sh …/tmp/polish-complete-pnpm' | exit0: native Go vet/gofmt, Docker package.json ESLint на final source image с текущими tests read-only;0ошибок/5прежних no-img-element warnings для приватных PNG |
| node --experimental-strip-types --test tests/*.test.mjs в final image | exit0,24PASS/0SKIP,2,254с; calendar collision/minimum text height включён |
| Production Next build --webpack + TypeScript | exit0, RUN113,8с; repet-polish-web-complete. SHA256 app/features/shared/lesson-content/config совпал с host:2201e4316a8fb5a0f49803c0c68e619d83f77aad33659108924cf3def7503fac |
| layout.e2e.mjs / Chrome на final image | exit0,84,263с:125 сочетаний/25экранов и ролей/5ширин320,375,768,1024,1440;0overflow/0page errors с длинными русскими именами, словами и URL. Дополнительно меню, уведомления, открытые фильтры, диалог курса, дневной календарь и форма |
| Реальные auth-navigation + exercise-bank + workshop browser | exit0,3PASS/0SKIP,86,278с; auth15,201с, bank34,259с, workshop33,934с. Создание/поиск/порядок подборки/retry, назначение двум ученикам, pinned mobile player, скрытые ключи и результат проверены через API10 |
| Расширенный workshop отдельно | exit0,38,485с: все20уроков1440/375 и общий фон; для10новых trainer/filter/touch, source ACL401/403. Скриншоты tmp/frontend-polish, tmp/bank10 и tmp/workshop-additions просмотрены |

Промежуточные сбои не засчитаны: native tsc остановлен143 после зависания; dev tsc упал на частично сгенерированных .next/dev/types; dev HMR блокировал гидратацию127.0.0.1 (local origins добавлены, preview production). Mocked server-auth мастерская перенаправилась из-за fake cookie и проверена отдельным real E2E. Bank test исправлен на role-based имена select и ожидание нового prompt после асинхронной навигации; iframe test ждёт document.complete до касания. Прерванная промежуточная сборка130 не PASS. Git status128: прежний timeout индекса, индекс не изменён; no-index whitespace21файла PASS. Сводка трекера20строк exit0.

Отдельный loopback preview http://127.0.0.1:3015 использует synthetic repet_polish_preview и API binary10/migrations0001–0014 (:18098,HTML:18099), контейнеры repet-polish-web/repet-polish-api и volume repet-polish-objects. Проверочная repet_frontend_polish_test удалена. Старые preview и пользовательские данные не заменены, push/PR/deployment не выполнялись. UI image соответствует текущему frontend; browser не подтверждает незавершённые study11 API. Статусы10/11 не повышены этой UI-работой.

### Контрольная точка 10–11 — 2026-10-29

Последнее указание пользователя: приоритет коду, проверки отдельным поздним проходом. Это не разрешение отмечать непроверенные функции DONE. Код11 начат до отложенной приёмки10; статус обоих IN_PROGRESS.

До11: первый bank PostgreSQL6,24с; расширенный112,270с; полный race376,150с HTTP/33,728с store прошёл до последних правок replay/course locks, затем focused bank race50,496с прошёл после них. Финальный make test на отдельной repet_bank10_final_test exit0, HTTP207,659/store27,118с; другие пакеты cached явно сохранены. Промежуточные make test не прошли из-за пересечения fixtures/общей PDF queue в одной test DB при параллельных прогонах; bank search изолирован маркером, финальный полный прогон выполнен отдельно. Frontend23unit PASS0SKIP до11. Web build/TypeScript repet-bank10-web-verified exit0, Next compile11,6мин + TypeScript5,2мин, весь build1203,7с. Это образ10, не текущий код11. Нативная cross-build API10 `-buildvcs=false` exit0 (обычная упала на недоступном Git index); runtime image ещё не собран. Обычный Docker API build прерван SIGINT, exit130, не PASS. 0014 empty down/up exit0; OpenAPI YAML7paths/8refs разобран Ruby, не полный validator. No-index whitespace28files PASS до11; обычный Git diff exit128 сохранён. Browser10 написан, но не запущен. Финальный make lint10 пока не запущен. Все длительные процессы на момент последнего просмотра завершены.

11: набор упражнений хранится в существующей immutable bank collection; study_sessions выбирает только закреплённые exercise keys. Переиспользованы native learning_attempts/exercise_attempts/закрытые ключи/серверная проверка. Добавлены student-specific scope, произвольная навигация и редактирование сохранённых ответов только для новых study exams, сдача с пропусками, deadline clamp и DB trigger запрета позднего ответа. Native assignment07 semantics должны остаться прежними, но новая регрессия ещё не запускалась. Конфигурации принадлежат преподавателю, версия параметров неизменяема; raw points не официальный балл. Публичный вариант с manual задачами требует подтверждённой связи с проверяющим. Результаты11 не объявляются проверенными.

### Задача 09: PDF — 2026-10-29

Миграция0013,9paths/10операций (`pdf-imports`, страницы/crop/исходник/черновик, `pdf-assets`, lesson pdf-source/pdf-review). ObjectStorage filesystem0700/0600, immutable source/PNG, квоты и отдельный Poppler25.12.0 Alpine3.24: nonroot/read-only/internal network/cap_drop/no-new-privileges/tmpfs/CPU/memory/pids, без DB/secrets/public port. Загрузка5МиБ/10страниц, durable lease/replay, нормализованные PNG, raw/corrected text отдельно, ручной OCR-этап, источник/права. Условия не обрезаются; предложенные типы редактируются. Native ExerciseRenderer/редактор/закрытые ключи/hints/разборы/публикации/назначения переиспользованы. Server review привязан к показанному revision; правка страницы/урока снимает его, публикация без проверки запрещена. PNG доступен только автору либо через доступный snapshot/выбранное назначение; страницы и original закрыты ученику. Отмена загрузки очищает staged object с независимым контекстом и разрешением COMMIT под author lock.

- Финальный `TEST_DATABASE_URL=.../repet_pdf09_final_test make test`: exit0, httpapi22,859с; остальные результаты, включая cached, не выданы за свежие. БД отдельная synthetic, затем удалена.
- Свежий `go test -race ./... -count=1` после stale-review: exit0, httpapi260,206/store52,737с. После последних cleanup/list изменений `go test -race ./internal/httpapi -run TestPDFImportsPostgres -count=1`: exit0,45,815с; проверены роли/чужие ID, metadata paging, original, crop bounds, immutable snapshots/assets, replay/concurrent upload, expired lease, stale revision и отмена сразу после Put без orphan.
- `make lint PNPM='sh .../tmp/pdf09-pnpm'`: exit0; Go vet/gofmt нативно, frontend `npm run lint` в текущем Docker web-complete.5предупреждений no-img-element: приватные cookie-auth PNG намеренно не передаются публичному Next image optimizer. Native pnpm lint по-прежнему не подтверждён.
- Frontend `node --experimental-strip-types --test tests/*.test.mjs`:22PASS,0SKIP. Новый focused PDF unit после запрета обрезания больших условий:2PASS. Последний production Next build+TypeScript из текущего app source: exit0,25,2с. Host/image SHA256 дерева app/features/shared/config одинаков: `2ad290905054f5b4b0ab908add741a451e9b16bcdba11631036c2cb782a2e71c`.
- Последний реальный browser PDF на API-ready/web-complete:1PASS42,001с. Собственный двухстраничный PDF → raw text/scan → ручная сверка → crop →3native упражнения → preview → отказ устаревшего подтверждения из другой вкладки → публикация → student проходит/получает2auto +1manual pending. Teacher1440/375 и student375, PNG nonzero, horizontal overflow отсутствует, screenshots просмотрены. Auth-navigation/HTML/material assignments также3PASS; промежуточный PDF locator совпал с Next announcer, исправлен и повторно прошёл, это не скрытый PASS упавшего запуска.
- `0013 down/up` на пустой migration DB exit0; populated down ожидаемо exit3 с требованием экспорта. Compose config exit0. OpenAPI YAML +8refs разрешены,9paths/10ops; это не полный OpenAPI validator.
- `git diff --check` exit128: файловый timeout индекса сохранён. `tmp/pdf09-whitespace.mjs`:45файлов PASS через no-index. Ничего не push/deploy/PR. Старые synthetic БД09 и старый volume удалены; current preview на3011 с отдельной БД/volume оставлен.

Ограничения: автоматический OCR не заявлен; извлечение не безошибочно, vector figures сохраняются явным crop; quarantine не является антивирусной сертификацией. При недоступной БД неопределённый COMMIT сохраняет объект до reconciliation; общая GC/backup ещё не реализованы. Подробности и команды в `docs/PDF_IMPORT.md`, контракт `packages/contracts/openapi/pdf-imports.yaml`. Это не подтверждение готовности MVP.

### Задача 08: HTML — 2026-10-29

Миграция 0012, 11 API-операций (9 paths) и отдельный content listener. Загрузка UTF-8 до 512 КиБ, метаданные/права, courses/topics/классы, фильтры/пагинация, авторские версии и оригинал. Изолированный hostname и CSP + iframe sandbox без allow-same-origin; source не исполняется на основном origin. Назначение фиксирует публикацию, manual text workflow не записывает exercise_attempts. Новая версия/архив не меняют выданную. Конвертация — отдельный draft + provenance одной транзакцией, без JS/ключей; публикация только после добавления native упражнений. Критерий загрузка → preview → публикация → назначение → ученик проверен. См. `docs/HTML_LESSONS.md`, `packages/contracts/openapi/html-lessons.yaml`.

| Проверка | Результат |
| --- | --- |
| `TEST_DATABASE_URL=… make test`, отдельная repet_html08_test на 127.0.0.1:15437 | exit 0; HTTP 25,183 с, store 8,530 с; auth/exercise cached, PostgreSQL не SKIP |
| `TEST_DATABASE_URL=… go test -race ./... -count=1` | exit 0; HTTP 48,894 с, store 22,160 с; все unit выполнены заново |
| После добавления positive native-conversion assertion: `go test -race ./internal/httpapi -run TestHTMLLessonsPostgres -count=1` | exit 0, 16,967 с; 0 native упражнений → publish 422, добавлено авторское numeric → publish 200; legacy submission всё ещё даёт 0 exercise_attempts |
| `make lint PNPM='/usr/local/bin/docker run --rm -v /Users/maksimmasevskij/Desktop/репетитор/apps/web/tests:/app/tests:ro repet-html08-web-final npm run'` | exit 0; native vet/gofmt, контейнерный ESLint, не native pnpm |
| `node --experimental-strip-types --test apps/web/tests/*.test.mjs` | exit 0; 20 групп / 0 SKIP |
| `docker build -f tmp/html08-web-final.Dockerfile -t repet-html08-web-final apps/web` | exit 0; Next production build + TypeScript, имеющиеся зависимости без новых npm packages |
| `REPET_E2E_URL=http://127.0.0.1:3010 REPET_PLAYWRIGHT_MODULE=…/playwright/test REPET_CHROME_PATH='/Applications/Google Chrome.app/Contents/MacOS/Google Chrome' node --test --test-concurrency=1 apps/web/tests/html-lessons.e2e.mjs apps/web/tests/material-assignments.e2e.mjs apps/web/tests/auth-navigation.e2e.mjs` | exit 0; 3 теста, 47,226 с: HTML 20,562 с, material assignment 16,207 с, auth navigation 8,831 с; desktop1440/mobile375, реальный JS-counter, parent/cookie/storage/popup/fetch denied, forged postMessage не создаёт статистику, точное сохранение BOM/исходника, публикация/выдача/v1 после v2, native conversion |
| 0012 down/up на пустой repet_html08_migration | exit 0 / exit 0; populated down на test-БД ожидаемо отклонён exit 3 без удаления |
| `docker compose config --quiet`, также `WEB_PORT=3108 HTML_PREVIEW_PORT=18108 docker compose config --quiet` | exit 0; preview hostname, build args и CORS используют соответствующие порты |
| Ruby YAML parser и локальные refs html-lessons.yaml | exit 0; 9 paths / 13 refs; не полный OpenAPI-linter |

Frontend source digest совпал с проверенным образом: `b797cc6be415a7bd2a58e10134a4209b76d47753fe06e4f1acf1e297c35fc010` (app/features/shared/lesson-content, package/lock/config; без node_modules/build/tests). Скриншоты `tmp/html08/teacher-html-desktop.png`, `student-html-mobile.png` просмотрены. API/frontend preview 18090/3010, изолированный HTML `http://html.localhost:18091`, отдельная синтетическая repet_html08_preview; исходные planner/task07 ресурсы не заменялись. Test/migration БД удалены. Git index остаётся недоступен; no-index whitespace проверка 43 файлов прошла. Push/deploy/PR не выполнялись.

Ограничения: CDN/сеть/storage/popups/eval/внешние файлы старых HTML не поддержаны; встроенные JS-ключи не являются секретами, баллы недостоверны. Конвертация требует ручной проверки текста и создания упражнений, рисунки/интерактивы не переводятся автоматически. Quota 16 МиБ/1000 версий; это небольшой versioned content, не production ObjectStorage/AV. Существующая репозиторная мастерская не была автоматически импортирована в аккаунты, её исторические ограничения остаются отдельно. Родители/экзамены/аналитика/публичный deployment этим DONE не заявляются.

Новая нумерация 08–19 утверждена пользователем 2026-10-29; TODO в новой строке не удаляет прежнюю реализацию. Прежние незакрытые проверки календаря/доработки текстовых заданий/сообщений/уведомлений/UI/CI/backup/V2 остаются ограничениями аудита. Исторический срез мастерской: 20 тем из 19 автономных HTML, teacher-only browser-проверки от 2026-10-20; он не равен новому импортёру и аналитике. Рабочий план из Excel проверен отдельно ниже. Новая очередь и возобновление описаны в `CODEX_NEXT_RUN.md`.

### Рабочий план преподавателя из Excel — DONE по функциональному срезу, 2026-10-29

Источник изучен только для функционального переноса: ученики, тариф/длительность/контакт, типовая неделя, пауза, пересечения, рабочие окна, плановая выручка, калькулятор цели и сравнение форматов. Персональные данные из книги не импортируются в исходники или тесты. Это дополнение к календарю, не закрытие всех критериев этапа 08. Карточки плана не создают аккаунты, а типовые занятия не создают датированные lessons; приглашения и прежний календарь сохранены. Доступ из главной преподавателя и календаря; отдельный пункт верхнего меню не добавлен. См. `docs/TUTOR_PLANNER.md`.

Миграция 0011 хранит отдельный типизированный JSON-документ с ревизией в `tutor_plans`; организация/владелец определяются из сессии. GET/PUT `/api/v1/tutor-planner` проверены: 200, 400 на невалидные/неизвестные поля, 401 без сессии, 403 ученику, 409 при устаревшей ревизии. Другой преподаватель получает только свой пустой план, другая организация не изменяет чужой. Конкурентные PUT одной ревизии дают ровно один успех. Локальные ключи карточек не являются account IDs.

Проверки этого среза (все данные синтетические):

| Команда / окружение | Фактический результат |
| --- | --- |
| `TEST_DATABASE_URL=… make test`, отдельная БД `repet_planner_test`, PostgreSQL 127.0.0.1:15437 | exit 0; httpapi 93,928 с, store 8,801 с; auth/config/exercise cached, PostgreSQL-тесты выполнены без SKIP |
| `go test -race ./internal/httpapi ./internal/store -run TestTutorPlan -count=1` с той же TEST_DATABASE_URL | exit 0; HTTP 11,171 с, store 3,534 с; сохранение/удаление, валидация, ACL/org и конкурентная ревизия |
| `make lint PNPM='/usr/local/bin/docker run --rm -v /Users/maksimmasevskij/Desktop/репетитор/apps/web/tests:/app/tests:ro repet-planner-web-fixed npm run'` | exit 0; нативные go vet/gofmt и контейнерный ESLint текущего frontend; это не проверка нативного pnpm. Первая команда с урезанным PATH не нашла docker (127), повтор с абсолютным путём успешен |
| `node --experimental-strip-types --test apps/web/tests/*.test.mjs` | exit 0; 20 групп, без SKIP, включая 4 новые: пауза/пересечения, разрозненные окна/подготовка, форматы/4,33 недели/нулевая загрузка и минимальная положительная цель, безопасные контакты |
| `docker build -f tmp/planner-web-final.Dockerfile -t repet-planner-web-fixed apps/web` | exit 0; Next.js production build и TypeScript; использованы имеющиеся зависимости из локального task07-образа, новых пакетов/install scripts нет |
| `REPET_E2E_URL=http://127.0.0.1:3009 REPET_PLAYWRIGHT_MODULE=…/playwright/test REPET_CHROME_PATH='/Applications/Google Chrome.app/Contents/MacOS/Google Chrome' node --test apps/web/tests/tutor-planner.e2e.mjs` | exit 0; 17,759 с: два ученика, контакт/поиск, три занятия, пересечения/пауза, keyboard-окно, сохранение, reload, формы/edit/delete на 375px/touch, конфликт вкладок и refresh, приватность и запрет ученику; pageerror отсутствует |
| Те же browser-переменные, `node --test apps/web/tests/auth-navigation.e2e.mjs` | exit 0; 24,454 с, переход входа/регистрации мышью, touch и клавиатурой |
| Down/up 0011 на пустой preview-БД, down на заполненной test-БД | пустой down/up exit 0; заполненный down ожидаемо отклонён exit 3 без удаления данных |
| Ruby YAML parser + проверка локальных refs контракта `tutor-planner.yaml` | exit 0; 1 API path / 2 операции, 7 refs разрешены; это не полный OpenAPI-linter |

В браузере найден и исправлен выход скрытого заголовка таблицы за мобильный scroll-контейнер; итоговые проверки document.scrollWidth на 375px проходят. Скриншоты: `tmp/planner/week-desktop.png`, `income-mobile.png`, `students-mobile.png`. Исходники app/features/shared/lesson-content и конфигурация проверенного frontend-образа совпадают с рабочей копией, SHA-256 `5bfe3fbf14a1fe3bde6885b7675eab2af0b5e21cdff58acb99be648c2424a839`; node_modules/build-артефакты/тестовые скрипты этим digest не проверяются. Последняя правка E2E-селектора отдельно проверена ESLint с bind-mount tests.

Preview: web 3009, API 18089, отдельная синтетическая БД `repet_planner_preview`; существующие контейнеры task07 и их БД не перезапускались. Test-БД после проверок удалена. Push, PR, deployment и GitHub settings не выполнялись. `scripts/repet-progress.sh` — exit 0. Обычный `git diff --check` — exit 128: Git index по-прежнему недоступен (`unable to map index file: Operation timed out`); индекс не пересоздавался и чужие изменения не откатывались. Проверка `git diff --no-index --check /dev/null <file>` для всех 20 файлов этого среза прошла без whitespace-ошибок.

- `make test`, с `TEST_DATABASE_URL` на временной БД `repet_progress_audit_20261019`: exit 0. Пакеты httpapi и store выполнили PostgreSQL-тесты, а не пропустили их. auth/config — cached; это не новый независимый прогон этих unit-тестов.
- `make lint`: Go vet и gofmt прошли. Нативный ESLint не завершился примерно за 4 минуты; созданный этим запуском процесс остановлен SIGTERM, make вернул exit 2 (команда lint: 143). Это НЕ PASS полного нативного make lint; причина задержки локальных зависимостей не установлена.
- `TEST_DATABASE_URL=… make race`: exit 0 на той же отдельной БД, включая HTTP/store PostgreSQL-тесты; auth/config также выполнены без cached-отметки.
- Frontend в отдельном временном контейнере из текущего локального образа: `docker run --rm repet-web sh -c 'node node_modules/typescript/bin/tsc --noEmit && node node_modules/next/dist/bin/next build'` — exit 0. Это не новая сборка Docker и не deployment. Первая попытка через pnpm завершилась exit 127: pnpm отсутствует в runtime-образе; повтор через node успешен.
- `docker run --rm repet-web node node_modules/eslint/bin/eslint.js .` — exit 0. Это отдельная успешная ESLint-проверка, а не замалчивание ошибки нативного make lint.
- Для подтверждения совпадения кода вычислен SHA-256 отсортированных относительных путей и содержимого `app`, `features`, `shared`, `lesson-content`, а также `proxy.ts`, `next.config.ts`, `package.json`, `pnpm-lock.yaml`, `tsconfig.json`, `eslint.config.mjs`. Образ и рабочая копия дали одинаковый digest `6958b98b5f7e19a602009a78098117916a96d6a291556b7823a40d7307577d1f`. Host toolchain, node_modules и build-артефакты этим сравнением не проверяются.
- `docker compose ps`: postgres, api, web, mailpit — healthy. Порты этой среды: PostgreSQL 15433, API 18080, web 3000, SMTP 11025, Mailpit 18025; defaults в Compose отличаются.
- `docker compose config --quiet`: exit 0. `GET :18080/readyz`: 200, `status=ready`. `GET :3000/login`: 200. `GET :3000/api/v1/dashboard` без cookie: 401. `GET :3000/lesson-content/logic` без cookie: 401.
- `schema_migrations` временной БД: `0001_init`, `0002_repet_product`, `0003_integrity`, `0004_password_reset`. Down-миграции, backup/restore, SMTP-доставка и браузерные сценарии в этом аудите не проверены.
- Репозиторий уже имел незакоммиченные изменения frontend, core, AGENTS и библиотеки уроков. Они не изменяются этим аудитом. Работающая сборка содержит эти изменения; это не означает, что они сохранены в Git.
- Временная БД после test/race удалена; временные проверочные контейнеры запускались с `--rm`. Пользовательская БД и работающие контейнеры не заменялись и не перезапускались.
- `sh -n scripts/repet-progress.sh`, запуск сводки из корня и из `apps/web`: exit 0. Тестовые потоки с неизвестным статусом, отсутствующим ID 19 и повторным ID 01 вернули ожидаемый exit 1. `git diff --check`: exit 0.

## Найденные ограничения

1. Правило AGENTS запрещает клиентский `role`, а действующая регистрация принимает выбор tutor/student; admin запрещён. Это расхождение политики и API, не доказательство возможности стать admin. Прежний auth-критерий 05 требовал явного согласованного решения без поломки регистрации; после назначения номера 05 прохождениям это ограничение остаётся незакрытым.
2. `CORS` разрешает ответы определённым origins, но сам по себе не является полной CSRF-проверкой. Rate limiting auth-маршрутов в изученной цепочке middleware отсутствует. Production-безопасность не подтверждена.
3. Сообщения и уведомления обновляются запросами и локальным refresh, не серверной realtime-доставкой. Метрика `/metrics` отражает только liveness.
4. Календарь умеет создать/прочитать занятие; статусы в схеме БД не означают наличие API переноса и отмены.
5. Мастерская хранит автономные HTML в исходниках; 10 тем получены разделением входов «Круги Эйлера» и «Графы» одного файла. Это ещё не CMS для 90 уроков и не система учебного прогресса. `sandbox` разрешает scripts и same-origin: изоляцию необходимо пересмотреть перед загрузкой непроверенных HTML.
6. Старый `EXECUTION_PLAN.md` относит задания/чат/уведомления к будущему, хотя базовые реализации уже есть. Использовать актуальный трекер, не реализовывать их повторно.
7. Локальный lint не воспроизводится до конца за проверенное время; контейнерный lint того же frontend проходит. Для ежедневной разработки нужно отдельно диагностировать локальные node_modules/runtime. В этом аудите зависимости не переустанавливались.
8. Прежний критерий 07 «Главная и профиль» не закрыт сменой назначения номера: реальные dashboard-данные и пустые массивы были проверены, но полная браузерная приёмка сохранения профиля обеих ролей остаётся отдельной работой. Новый критерий 07 относится к интерактивным домашним заданиям.

## Что переиспользовать

| Область | Готовые точки расширения | Подтверждение / границы |
| --- | --- | --- |
| HTTP | `httpapi.NewServer`, `decodeJSON`, `pageOffset`, `WriteJSON`, `WriteError` | HTTP integration test прошёл; production hardening отдельно |
| Идентификация | `RegisterUser`, `LoginUser`, `UserBySessionToken`, `RevokeSession`, `IssuePasswordReset`, `ResetPassword` | Базовые store/HTTP сценарии; доставка SMTP не проверена |
| Связи | `CreateInvitation`, `AcceptInvitation`, `GetRelationForActor` | Email binding, повтор и конкуренция проверены |
| Занятия | `CreateLesson`, `ListLessonsPage`, `GetLessonForActor` | Создание и ACL проверены, не весь календарный lifecycle |
| Задания | `CreateAssignment`, `SubmitAssignment`, `GradeAssignment` | Базовый цикл и конкурентная отправка проверены |
| Сообщения | `ListConversations`, `ListMessages`, `SendMessage` | История, ACL и idempotency проверены |
| Уведомления | `ListNotificationsPage`, `MarkNotificationRead`, `MarkAllNotificationsRead` | Первые две операции частично покрыты; mark-all требует проверки |
| Данные | `store.New`, `store.Migrate`, `pgxpool`, существующие migrations | Прямые миграции проверены на свежей БД |
| Оболочка UI | `AppShell`, `useWorkspace`, `Brand`, `Icon`, CSS tokens | Код существует; не отдельная browser/a11y сертификация |
| Клиент API | `authApi`, `productApi`, типы в `shared/api/types.ts` | Одна точка запросов; не создавать дублирующий клиент |
| Даты | `dateInTimeZone`, `formatDateTime`, `formatTime`, `todayISO` | Существующие helpers; edge-case тесты timezone нужны |
| Уроки | `lessonCatalog`, `findLesson`, `teacherAccess`, `lessonTheme` | Есть каталог и серверный guard; интерактивность каждого HTML не проверена |
| Проверки | `TestHTTPAuthorizationPostgres`, `TestProductFlowPostgres`, Makefile, CI workflow | Расширять существующий сценарий, не заменять его заглушками |
| Учебный каталог | `CreateCourse`, `CreateTopic`, `CreateLearningLesson`, `SaveExercise`, `PublishLearningLesson`, `ListLearningLessons`, `GetLearningVersion` | API/PostgreSQL/race проверены в задаче 01; не дублировать каталог в новом сервисе без миграционного решения |
| UI учебного каталога | `catalogApi`, `allCatalogPages`, `loadLibraryStructure`, `TaskSectionNav`, `LessonArticle`, `ClassificationFields`, `EditorDialog` | Используют существующий `request`, сессию, Brand/AppShell и styles.css; browser E2E задачи 02 прошёл; ключи загружаются только авторским редактором |
| Движок упражнений | `ExerciseRenderer`, `exerciseRegistry`, `parseExercise`, `LessonExercises`, `exercisesApi`, `ValidateExerciseAnswer`, `CheckExercise` | Общий runtime-валидируемый UI и серверная проверка десяти типов; версии, ACL, idempotency и ручная проверка long_text; unit/API/browser проверки 03–04 прошли |
| Визуальные инструменты | `GridBoard`, `NumberLineInput`, `GridSelectInput`, `MatchingInput`, `OrderingInput`, `GeometryInput`, `parseVisualConfig` | Переиспользуемые React-компоненты и закрытые config-схемы; mouse/touch/keyboard, включая альтернативы drag-and-drop, проверены в 04 |
| Прохождения | `StartLearning`, `GetLearningState`, `CompleteLearning`, `RevealLearningHint`, `ListLearningAnswers`, `ListLearningAttempts`, `learningApi`, `StudentLearning`, `parseLearningState` | Используют существующие cookie, ACL каталога, request и серверную проверку упражнений; не создавать второй клиент или отдельную систему оценивания |

## Задача 01: backend каталога (2026-10-19)

По явному запросу пользователя номер 01 расширен от подтверждённого Go baseline до backend учебных материалов. Остальные статусы не пересматривались и следующие задачи не начинались. Критерии и ограничения API: [LEARNING_CATALOG_API.md](LEARNING_CATALOG_API.md). Основной OpenAPI подключает новый `learning-catalog.yaml`: 18 paths, 25 operations.

Результат:

- Созданы `courses`, `topics`, `learning_lessons`, `lesson_versions`, `exercises`, `exercise_answer_keys`; календарная `lessons` не изменена. Есть up/down миграции 0005 и FK, исключающие перенос темы/материала в чужой курс.
- Поддержаны school/olympiad/oge/ege, диапазоны классов 1–11, сложность 1–5, автор из сессии, статусы draft/published/archived и активные связи со студентами. Visibility по умолчанию linked_students; organization — явное расширение в пределах своей организации.
- Создание, получение, полное редактирование PUT, публикация/архивация, история, упражнения, закрытый авторский endpoint ключей. Поиск и стабильная пагинация с total, включая страницу за концом списка; фильтр темы включает подтемы.
- Публикация атомарна; копирует содержимое, упражнения и ключи в новый запечатанный снимок. Изменения черновика не переписывают публикацию. Триггеры запрещают UPDATE/DELETE и дописывание упражнений/ключей в старые публикации. Конкурентная публикация без новых правок создаёт только одну версию.
- Обычные DTO никогда не содержат answer/answer_key/explanation. Поиск не индексирует закрытые ключи. Ученику и другому преподавателю недоступны ключи, черновики и история; другое tenant-пространство не получает даже опубликованные organization-материалы.

Проверки в отдельной БД `repet_catalog_task01`, не в пользовательской базе:

| Команда / сценарий | Фактический результат |
| --- | --- |
| `TEST_DATABASE_URL=… make test` после последних изменений | exit 0; новый HTTP/PostgreSQL catalog test выполнен, не SKIP; неизменённые unit/store пакеты частично cached |
| `TEST_DATABASE_URL=… go test ./... -count=1` | exit 0; все тесты core выполнены заново после down/up миграции 0005 |
| `TEST_DATABASE_URL=… go test -race ./... -count=1` | exit 0; полный core, включая каталог и конкурентные публикации |
| После дополнительных негативных assertions: `go test -race ./internal/httpapi -run TestLearningCatalogPostgres -count=1` с TEST_DATABASE_URL | exit 0 |
| Итоговый `go test -race ./... -count=1` из services/core после проверки больших номеров/позиций и NUL в тексте/JSONB | exit 0; весь core, HTTP/PostgreSQL catalog test выполнен; итоговый Go vet/gofmt также exit 0 |
| 0005 down через `psql -v ON_ERROR_STOP=1`, удаление только тестовой записи 0005 из schema_migrations и повторная Migrate через тесты | exit 0; обратная и повторная прямая миграции успешны; legacy core остаётся работоспособен |
| `make lint` | Go vet и gofmt прошли; нативный ESLint не завершился примерно за 4 минуты, остановлен SIGTERM; make exit 2, lint-команда 143. Это не PASS полной команды |
| `docker run --rm repet-web node node_modules/eslint/bin/eslint.js .` | exit 0; frontend не менялся в задаче; совпадение с рабочими исходниками повторно проверено по digest из аудита выше |
| YAML parse и проверка локальных/external JSON Pointer refs OpenAPI | exit 0; 18 catalog paths / 25 operations; это не полноценная certification OpenAPI tooling |

`TestLearningCatalogPostgres` проверяет все 25 операций через httptest и реальную БД: авторский цикл, поддеревья/циклы/чужой курс, правильность defaults/диапазонов/направлений, отсутствие ключей в DTO и поиске, изменение черновика после публикации, сохранность старых ключей, конкуренцию, фильтры/offset/total, архивацию веток и завершение связи, чужого автора и реальные cookie-сессии другой организации. Непосредственные SQL-попытки испортить публикацию должны давать 23514. Календарная таблица не получает строк от операций учебного каталога.

Проверки проводились на HTTP-сервере теста, не на работающем Docker API. Пользовательские контейнеры не пересобирались/не перезапускались; новые endpoints в старой запущенной сборке не появляются автоматически. Браузерный UI, импорт HTML, отправка ответов ученика и автоматическая проверка упражнений не входят в задачу 01. Временная БД удалена после итоговых проверок. Push, PR и deployment не выполнялись.

## Задача 02: интерфейс библиотеки (2026-10-19)

По явному запросу пользователя этап 02 назначен frontend-библиотеке. Исторические результаты миграций из аудита выше сохранены; задача 03 и остальные этапы не выполнялись и их статусы не пересматривались.

- Маршруты: `/tasks/library`, `/tasks/library/[lessonId]`, `/tasks/library/new`, `/tasks/library/[lessonId]/edit`. Вход через поднавигацию «Задания», главное меню по-прежнему содержит четыре раздела.
- Каталог использует API 01, URL-фильтры класса/курса/темы/подготовки/сложности, авторские статусы и «Только мои», поиск только по названию и пагинацию по 12. Быстрые изменения фильтров объединяются через React optimistic state. Поскольку API q ищет также в содержимом, для поиска названия клиент проверяет все страницы кандидатов, а не только текущую страницу.
- Преподаватель создаёт курс и тему (включая подтему), урок и упражнения с закрытыми ключами. Сохранение частичного результата переживает повтор после ошибки без второго урока. Есть native validation, предупреждение об уходе с несохранёнными правками, preview без ключей и явное подтверждение публикации курса/цепочки тем/урока с указанием реальной области доступа.
- Страница чтения автора опубликованного урока показывает именно публикацию, отдельно предупреждает о черновых правках. Ученик не получает интерфейс редактора, ссылку редактирования или запрос answer-keys. Повторная публикация не меняет первый снимок.
- Обложки используют существующий Brand и реальное название курса; количество упражнений вычисляется из ответа API. Поля длительности в API нет, поэтому UI его не показывает. Предмет API сейчас только математика, поэтому предмет зафиксирован, без выдуманных вариантов. Текст выводится безопасно, без исполнения HTML. Импорт старых HTML-интерактивов, загрузка пользовательских обложек и решение/проверка упражнений не входят в 02.

Проверки текущей задачи:

| Команда / сценарий | Фактический результат |
| --- | --- |
| `TEST_DATABASE_URL=… make test` на отдельной `repet_library_task02_checks` | exit 0; httpapi и store PostgreSQL-тесты выполнены, не SKIP; auth/config cached. Проверочная БД удалена |
| `make lint` | Go vet/gofmt прошли; нативный ESLint не завершился за 4,5 минуты, собственный процесс остановлен SIGTERM; make exit 2, lint 143. Причина локального зависания не установлена, полный нативный lint не PASS |
| `docker build -t repet-library-task02-web apps/web` | exit 0; Next 16.2.10 production build и TypeScript, все четыре новых маршрута присутствуют |
| `docker run --rm repet-library-task02-web node node_modules/typescript/bin/tsc --noEmit` | exit 0 на итоговой сборке |
| ESLint итогового образа с bind mount текущей `apps/web/tests` | exit 0; проверены текущий frontend и актуальный browser-сценарий, не старые файлы теста из образа |
| `node --test apps/web/tests/library.e2e.mjs` с Playwright 1.62.1 и системным Chrome против :3001 | exit 0, 1 сквозной сценарий PASS, 0 SKIP, около 57 секунд; реальный API и PostgreSQL, не API-заглушка для основного цикла |
| Browser-проверки ошибок | Принудительные 503 только для проверки восстановления: сохранение упражнения и загрузка списка; повтор успешен. Loading проверен удержанием запроса, пустой результат проверен поиском текста, отсутствующего в названии |
| Browser desktop/mobile | Скриншоты 1440 и 375 px просмотрены; каталог, reader и редактор без горизонтального overflow на 375 px. Консольных pageerror нет; ответ и закрытое пояснение отсутствуют в preview/ученическом чтении |

SHA-256 отсортированных относительных путей и содержимого 63 файлов `app`, `features`, `shared`, `lesson-content`, `proxy.ts`, `next.config.ts`, `package.json`, `pnpm-lock.yaml`, `tsconfig.json`, `eslint.config.mjs` совпал у рабочей копии и проверяемого контейнера: `fa4efe6f5d0bc7fb5a2339155b2776f007570df69198a6aa045864a8e45a2653`. Сравнение не включает node_modules или артефакты сборки. Актуальный E2E запускается с host-файла; инструкция в `apps/web/tests/README.md`.

Локальный preview этого запуска: `http://127.0.0.1:3001/tasks/library`, отдельный API :18081 и отдельная БД `repet_library_task02` в существующем PostgreSQL. Владелец ресурсов: запуск задачи 02; имена контейнеров/сети `repet-library-task02-web`, `repet-library-task02-api`, `repet-library-task02`. Они оставлены для локального просмотра вместе с синтетическими E2E-данными. Preview не использует пользовательскую БД; старые web :3000/API :18080 не пересобирались и не перезапускались. Это локальная сборка, не deployment; push/PR/GitHub settings не выполнялись. Скриншоты находятся в игнорируемой `tmp/task02-library`.

## Задача 03: универсальный движок (2026-10-19)

Новый критерий 03 выполнен: один опубликованный урок показывает single_choice, multiple_choice, numeric, text и long_text через общий renderer/реестр. Объективные ответы проверяются сервером, развёрнутое решение сохраняется для автора; ручная проверка и уведомления используют существующий store. Закрытые ключи не входят в публичные JSON. Сервер владеет учеником, проверяющим, датами и результатом. Пагинация истории, объектные ACL, повтор запроса, конкуренция и неизменяемость опубликованного ключа покрыты интеграцией.

| Проверка | Результат |
| --- | --- |
| `TEST_DATABASE_URL=… make test` и `go test -race ./... -count=1` из services/core на отдельной проверочной БД | exit 0; PostgreSQL HTTP/store сценарии выполнены, не SKIP; повторены совместно с 04 |
| `node --experimental-strip-types --test tests/exercise-engine.test.mjs` | Четыре группы PASS: строгий публичный JSON/ответ сервера, дроби/знаки/пробелы, недопустимый ввод |
| `node --test apps/web/tests/exercise-engine.e2e.mjs` через Playwright/Chrome | PASS, 0 SKIP: преподаватель создаёт пять типов через UI, публикует; ученик решает, автор проверяет long_text; checking, network retry, reload, устаревшая версия и отсутствие утечек ключей проверены |
| Миграция 0006 down/up на отдельной пустой БД | exit 0; откат с несовместимыми новыми типами безопасно отклоняется |

Проверены POST `catalog/lessons/{id}/exercises/{exercise}/attempts`, GET `catalog/lessons/{id}/attempts`, PUT `catalog/lessons/{id}/attempts/{attempt}/review` (префикс `/api/v1/`). Числовая проверка использует стандартный `math/big.Rat`, новой формульной зависимости нет. Это не агрегированный учебный прогресс и не начало следующих этапов. Исторический frontend baseline остаётся в разделе аудита.

## Задача 04: визуальные математические инструменты (2026-10-19)

Критерий выполнен: number_line, grid_select, matching, ordering и geometry (точка на сетке, построение отрезков между заданными точками, выбор отрезка/многоугольника). Ученик изменяет объекты, а не смотрит статичную картинку. Мышь, touch и клавиатура работают; сортировка имеет кнопки вверх/вниз и клавиши как альтернативу перетаскиванию, matching/геометрия — select/radio. Анимации и взаимодействие локальные, итоговая проверка только по закрытому серверному ключу.

Миграция 0007 добавляет config к exercises, не меняя календарную lessons. Строгая Draft 7 схема, TypeScript runtime и Go проверяют формат; ссылки объектов, диапазоны и уникальность дополнительно проверяются по данным config. Опубликованный config неизменяем. Редактор преподавателя поддерживает настройки, preview и ввод закрытого ключа тем же визуальным инструментом. Number_line пока ограничена целой сеткой, geometry — минимальным набором объектов без символьной алгебры.

| Команда / сценарий | Фактический результат |
| --- | --- |
| `TEST_DATABASE_URL=… make test` на отдельной `repet_visual_task04_checks` | exit 0; HTTP/store с PostgreSQL выполнены, не SKIP; часть неизменённых unit-пакетов cached |
| `TEST_DATABASE_URL=… go test ./... -count=1` и `go test -race ./... -count=1` из services/core | exit 0; все core-тесты заново, включая 7 визуальных вариантов, неверные/недопустимые ответы, ACL и версии |
| `make lint` с доступными Go/pnpm | Go vet/gofmt прошли; нативный ESLint завис более двух минут, собственный процесс остановлен SIGTERM; make exit 2, lint 143. Причина локального зависания не установлена; это не PASS нативного lint |
| `docker build -t repet-visual-task04-web -f apps/web/Dockerfile apps/web` | exit 0; Next production build, TypeScript и генерация маршрутов прошли |
| ESLint + `tsc --noEmit` в итоговом образе с текущими tests и exercise-engine через read-only mount | exit 0; это отдельная успешная проверка контейнерного frontend, не подмена результата make lint |
| `node --experimental-strip-types --test tests/*.test.mjs` в том же контейнере | exit 0, 9 групп PASS, 0 SKIP; формальная схема с Ajv, runtime, границы сетки, пары, порядок, геометрия и предыдущие базовые типы |
| `node --test apps/web/tests/visual-exercises.e2e.mjs` с Playwright/Chrome против :3003 | exit 0, 1 сквозной сценарий PASS, 0 SKIP, около 14 секунд; реальный API/PostgreSQL, не mock |
| Регрессия `exercise-engine.e2e.mjs` на итоговом preview | PASS, 0 SKIP, около 18 секунд; все пять базовых типов и ручная проверка по-прежнему работают |
| Регрессия `library.e2e.mjs` на итоговом preview | После исправления гонки ожидания перехода в тесте exit 0, 1 сценарий PASS, 0 SKIP, около 13 секунд. Две первые попытки упали: ввод поиска попадал в ещё открытый редактор до завершения навигации; добавлено явное waitForURL, продуктовый код каталога не менялся |
| Mouse/keyboard/touch E2E | SVG-клик и range числовой прямой, клетки и граничные стрелки сетки, линии соответствий, настоящее mouse drag-and-drop, кнопки и клавиши сортировки, три режима geometry; верно/неверно и reload проверены |
| Browser desktop/mobile | Просмотрены снимки 1440 и 375 px; горизонтального overflow на 375 px нет, pageerror и запросов приватных ключей нет; взаимодействие до отправки ответа не вызывает проверочный API |
| 0007 down/up на отдельной `repet_visual_task04_migration` | exit 0; повторная Migrate успешна. Down на проверочной БД с 42 визуальными упражнениями ожидаемо exit 3; данные и config сохранены транзакцией |
| Проверка OpenAPI/JSON внешних и внутренних ссылок | exit 0; 3 файла, 233 ссылки разрешены. Контракт включает все десять типов и строгую схему config |

SHA-256 исходников `app`, `features`, `shared`, `lesson-content` (без Markdown) и конфигурационных файлов frontend совпал у рабочей копии и итогового preview-контейнера: `5cf873a4340f8519c91e8176d3518114032f6cca4a4032dda92694a9583de6c8`. Сравнение не включает node_modules, сборочные артефакты и тесты; последние проверены с актуального host-кода/mount. Детали форматов: `apps/web/features/exercise-engine/README.md`.

Проверочные БД `repet_visual_task04_checks` и `repet_visual_task04_migration` после завершения проверок удалены. `sh scripts/repet-progress.sh` и `git diff --check` завершились exit 0; 20 строк, 6 DONE, 10 IN_PROGRESS, 4 TODO, 0 BLOCKED. Это не процент готовности продукта.

Локальный preview 04: `http://127.0.0.1:3003/tasks/library`, API :18083, отдельная БД `repet_visual_task04` с синтетическими данными. Ресурсы этого запуска: контейнеры `repet-visual-task04-web-final`/`repet-visual-task04-api`, сеть `repet-visual-task04`. Они оставлены для просмотра; предыдущие preview 02/03 перестали работать после остановки Docker daemon между запусками. Docker Desktop был вновь запущен; основной web/API не пересобирались и не заменялись, пользовательская БД для тестов не использовалась. Push/PR/deployment не выполнялись. Задача 05 не начиналась.

## Дополнение мастерской: десять уроков (2026-10-20)

По отдельному запросу добавлены предоставленные пользователем HTML без изменения содержимого: «Комбинаторика — 1», «Сколько?», «Логика — 2», «Эффект плюс-минус один — 2», «Клетчатые фигуры», «Учти лишнее», «Комбинаторика — 2», «Последняя цифра», «Метод от противного», «Конструкции: много и мало». Новые карточки продолжают существующий каталог `/teaching/lessons`; каждый урок открывается отдельной страницей с прежней общей темой и teacher guard. Счётчики каталога и reader вычисляются по lessonCatalog. Новых верхних пунктов меню, API и таблиц БД нет.

| Проверка | Фактический результат |
| --- | --- |
| `TEST_DATABASE_URL=… make test` на отдельной `repet_workshop_additions_checks` | exit 0; HTTP PostgreSQL 16,1 с и store 8,4 с, не SKIP; неизменённые auth/config/exercise cached. БД после проверки удалена |
| `make lint` | Go vet/gofmt прошли; нативный ESLint не завершился за две минуты, собственный процесс остановлен SIGTERM; make exit 2, lint 143. Известное ограничение host-зависимостей, не PASS полного нативного lint |
| `docker build -t repet-workshop-additions-web -f apps/web/Dockerfile apps/web` | exit 0; Next production build и TypeScript прошли |
| ESLint, `tsc --noEmit`, `node --experimental-strip-types --test tests/*.test.mjs` в итоговом образе с текущими tests через read-only mount | exit 0; 10 групп unit PASS, 0 SKIP, включая уникальные маршруты, наличие всех HTML, синтаксис inline JS и JSON |
| `node --experimental-strip-types --test apps/web/tests/workshop-lessons.e2e.mjs` с Playwright/Chrome против :3004 | exit 0; 1 сквозной сценарий PASS, 0 SKIP, 18,6 с: все 10 новых уроков, 20 карточек, счётчики, общий theme, trainer/filter desktop и touch mobile 375px |
| Доступ и отображение | Для каждого нового HTML: аноним 401, ученик 403, преподаватель 200/private no-store; неизвестный slug 404. CSP запрещает сетевые запросы; pageerror и горизонтального overflow на desktop/mobile нет. Скриншоты каталога и reader просмотрены |

Первые две попытки browser-теста выявили различия исходных интерфейсов, а не ошибки уроков: мобильная навигация бывает отдельной ссылкой «Тренажёр», а option может иметь значение через текст без атрибута value. Сценарий исправлен и повторно прошёл целиком. Математическая корректность всех встроенных задач не проверялась. HTML остаются автономными teacher-only тренажёрами с localStorage, не материалами API каталога и не серверным учебным прогрессом. Статус этапа 12 остаётся IN_PROGRESS.

Исходники frontend итогового образа и рабочей копии (app/features/shared/lesson-content без Markdown плюс конфигурационные файлы) совпали по SHA-256: `e2eefec27af3d8a235c26a32dc3da5b1078f20e634376398ccd8f71cdff117c9`. Проверочный сценарий запускается с актуального host-файла; артефакты — игнорируемая `tmp/workshop-additions`.

Локальный просмотр: `http://127.0.0.1:3004/teaching/lessons`, контейнер этого запуска `repet-workshop-additions-web`. Он использует существующий изолированный preview API задачи 04 и её тестовую БД с синтетическими аккаунтами, не пользовательскую БД. Основные контейнеры не заменялись. Push/PR/deployment не выполнялись. Последующие roadmap-задачи не начинались.

## Задача 05: прохождения и результаты (2026-10-20)

Критерий выполнен для уроков API каталога. Миграция 0008 добавляет learning_attempts, журнал идемпотентных событий/подсказок и расширяет существующую exercise_attempts, не создавая вторую систему оценивания. Сохраняются все отправленные ответы, их серверный номер, результат/ручная проверка, версия, число выданных подсказок и серверные секунды. Восстановление не зависит от localStorage; новые публикации не меняют незавершённую закреплённую версию.

Добавлены start/resume, состояние, завершение, собственные результаты и история с пагинацией; отправка ответа переиспользует существующий endpoint с learning_attempt_id. Подсказки задаются автором в закрытом ключе и выдаются отдельной серверной операцией. UI показывает ответы, ошибки, использованные подсказки, результаты и историю прежних прохождений. Запоздавший сетевой ответ не может стереть более новое состояние. Доступ проверяется по текущему ресурсу и владельцу; отзыв доступа блокирует и повторные запросы.

Документация: [LEARNING_PROGRESS_API.md](LEARNING_PROGRESS_API.md). OpenAPI подключает learning-progress.yaml: 5 новых путей / 6 операций плюс существующая отправка ответа и review. Номера, identity, timestamps, hints_used и правильность не принимаются от ученика. Ручная проверка может завершиться после сдачи урока; completed не означает «все решения верны».

| Проверка | Фактический результат |
| --- | --- |
| Итоговый `TEST_DATABASE_URL=… make vet test` на отдельной repet_learning_task05_checks | exit 0; Go vet и HTTP/PostgreSQL выполнены, HTTP 6,9 с, не SKIP; неизменённые unit/store cached |
| `TEST_DATABASE_URL=… go test -race ./... -count=1` | exit 0; весь core выполнен заново, включая ACL, повторные запросы, конкурентный start, версии, подсказки, ошибки, завершение и manual review; финальный полный прогон HTTP 65 с/store 23 с при параллельных локальных сборках |
| После уточнения hint_count в ответе создания: `go test -race ./internal/httpapi -run TestExerciseEnginePostgres -count=1` | exit 0, 41,3 с; включает новую assertion и все сценарии 03–05 |
| `make lint` | Go vet/gofmt прошли; нативный ESLint не завершился более чем за три минуты, собственный процесс остановлен SIGTERM. make exit 2, lint 143; полный нативный lint не PASS, причина host-зависания не установлена |
| Docker-сборки repet-learning-task05-api-verified и repet-learning-task05-web-verified | exit 0; Go binary и Next production build/TypeScript прошли |
| ESLint, `tsc --noEmit`, `node --experimental-strip-types --test tests/*.test.mjs` итогового frontend-образа с текущими tests через read-only mount | exit 0; 12 групп unit PASS, 0 SKIP. Проверены runtime-состояние, метаданные, ответы/подсказки и защита от запоздавшего ответа сети |
| `node --test --test-concurrency=1` для learning-progress/exercise-engine/visual-exercises/library E2E с Playwright/Chrome против :3005 | exit 0, 4 сценария PASS, 0 SKIP, 60 с суммарно. Новый сценарий 05 около 7 с, базовый 03 около 16 с, библиотека около 14 с, визуальные типы около 20 с |
| Критерий продолжения | Три ответа отправлены, вкладка действительно закрыта и открыта заново; ответы, результат, подсказка, run ID и старая версия восстановлены после новой публикации. Затем решены ещё два упражнения, завершён урок, выполнена ручная проверка, создано новое прохождение и открыта история прежнего |
| Browser desktop/mobile | Проверены 1440/375px, отсутствие горизонтального overflow и pageerror, отсутствие запросов приватных ключей и закрытых пояснений в ученическом чтении. Просмотрены снимки завершённого урока и истории |
| Миграция 0008 на отдельной копии БД | Откат с прохождениями ожидаемо exit 3. На тестовой копии без новых progress-строк down/up exit 0; все 118 старых ответов сохранены и связаны с 19 импортированными прохождениями. Проверочная копия удалена |
| OpenAPI/JSON refs | exit 0; 4 файла и 273 внешних/внутренних ссылки разрешены |

Ранее существовавшие browser-сценарии 02/03 обновлены под новую политику закрепления: старое прохождение продолжает прежнюю публикацию. Первое регрессионное ожидание 03 о 409 в UI после публикации больше не соответствовало задаче 05; теперь UI успешно продолжает run, а отдельная legacy-отправка без run всё ещё проверяется на 409. Итоговые четыре сценария прошли полностью.

SHA-256 актуального frontend и проверяемого контейнера совпал: `fe2bec26497a6e945a875e037591dda0ec47d9cb67d312420437798f93e2791a` (app/features/shared/lesson-content без Markdown плюс конфигурационные файлы; не node_modules/build/tests). Актуальные тесты запускались с host/mount. Изменения других задач не откатывались.

Границы: сохраняются отправленные ответы, не неотправленный текст поля. Время — серверный wall-clock с начала/предыдущего ответа, включая перерывы, не измерение внимания. У импортированных/legacy-ответов неизвестное время равно 0; новые legacy-запросы без learning_attempt_id остаются в общей истории, но не меняют прохождение. Автономные teacher-only HTML не подключались к серверному прогрессу. Прежние auth/security-ограничения аудита не исправлялись и не объявляются DONE.

Проверочные БД repet_learning_task05_checks и repet_learning_task05_migration удалены. Для локального просмотра оставлены отдельные контейнеры repet-learning-task05-web-verified (:3005) и repet-learning-task05-api-verified (:18085), сеть repet-learning-task05 и синтетическая БД repet_learning_task05. Адрес: `http://127.0.0.1:3005/tasks/library`. Основной сайт и пользовательская БД не заменялись. Push/PR/deployment не выполнялись; следующая задача не начиналась.

## Задача 06: LearningPlayer (2026-10-20)

Реализованы последовательные объяснение → авторский анимированный пример (если задан) → упражнения → результаты. Переиспользуются ExerciseRenderer/реестр десяти типов, API 05, закрытые ключи и immutable versions; новый верхний раздел не добавлен. Прогресс, разрешённая навигация, сохранение шага, retry и Continue работают с серверным состоянием. Неотправленный ввод явно помечается и предупреждает об уходе. Результаты включают историю ошибок/подсказок и ожидание ручной проверки.

Тренировка допускает повторные ответы/подсказки и разбор после допустимой отправки; развёрнутая задача — после проверки. Контрольная допускает одну новую отправку на задачу с nonce-safe replay; сервер закрывает hints/solutions, скрывает правильность/feedback/reviewed_at/агрегаты до завершения, не позволяет legacy-отправке или запросу разбора старой тренировки обойти активную контрольную. Режим нельзя сменить посреди прохождения. Это не прокторинг; уже изученная история не стирается.

Добавлены migration 0009, POST `learning-attempts/{id}/navigate`, GET `learning-attempts/{id}/exercises/{exerciseID}/solution`, mode в start и metadata состояния. Пример редактируется преподавателем и копируется в публикацию. Его JSON закрыт и валидируется по полям/границам. Нет генерации выдуманного объяснения/примера для старых уроков; без примера этот шаг пропускается. Подробнее: `docs/LEARNING_PLAYER.md` и обновлённые OpenAPI-контракты.

### Проверенные результаты

| Проверка | Фактический результат |
|---|---|
| `TEST_DATABASE_URL=… make test` (изолированная repet_player_task06_checks) | exit 0; HTTP/PostgreSQL 27,3 с, не SKIP; неизменённые unit/store cached |
| `TEST_DATABASE_URL=… go test -race ./... -count=1` после финальных дополнений теста | exit 0; HTTP 32,6 с, store 12,8 с; все core проверки заново. Включены practice/exam, ACL, скрытие результатов, nonce/conflict, запрет locked step, replay без отката курсора, malformed example и неизменность публикации после draft edit |
| Frontend: ESLint, `tsc --noEmit`, `node --experimental-strip-types --test tests/*.test.mjs` в Docker с текущими features/tests, смонтированными read-only | exit 0 на финальных исходниках, включая нейтральное сообщение преподавателю для submitted; 15 групп PASS, 0 SKIP |
| `make lint PNPM="docker run --rm -v …/features:/app/features:ro -v …/tests:/app/tests:ro repet-player-task06-web-verified npm run"` | exit 0; native Go vet/gofmt и package.json lint через Docker; не выдаётся за native pnpm lint |
| Playwright/Chrome `node --test apps/web/tests/learning-player.e2e.mjs` против отдельного :3006 | Финальный образ: exit 0, 1 большой сценарий PASS, 0 SKIP, 21,4 с; автор создаёт пример/публикует, ученик проходит numeric/grid/long_text, восстанавливает шаг на mobile после закрытия вкладки, получает ручную проверку; контрольная сохраняет нейтральный ответ, запрещает ранний разбор и раскрывает итог после завершения |
| Desktop 1440px / mobile 375px, touch, скриншоты в tmp/task06-player | Скриншоты просмотрены; нет горизонтального overflow/JS page errors; точка реально движется, анимация не делает POST; после completion прогресс 100% |
| Миграция 0009: down/up на repet_player_task06_migration | exit 0 после завершения migrate-only; четыре поля прохождения восстановлены. Down на проверочной БД с новым контентом/прогрессом отклонён exit 3, транзакция не изменена |
| OpenAPI YAML и локальные `$ref` (существующий js-yaml из ESLint) | exit 0; 4 документа, 272 ссылки разрешены, включая JSON-схемы frontend |
| Docker production builds API/Next и сверка web source SHA-256 | exit 0; финальный web 103933c881f080e4e93f221db4f3861af8950382d17b1216d2c1ea888c50c9b6 совпадает с preview по app/features/shared/lesson-content и build-конфигурации |

Нативный `make lint` дошёл до ESLint и завис: остановлен созданный процесс, exit 2/143; это не PASS. Go vet/gofmt прошли. В runtime-слое Docker нет pnpm: пробный make с Docker-pnpm завершился module-not-found; успешный повтор использовал npm run lint, запускающий тот же package.json script без изменения зависимостей.

Предыдущие E2E 03–05 со всеми упражнениями на странице не мигрированы на последовательный UI и не запускались в этом проходе; их прежний PASS не переносится на новый плеер. Текущий student journey заменён новым E2E, старые unit/server regression проходят; все десять визуальных типов повторно в браузере внутри нового плеера не проверялись. Существующие ограничения auth/security из первоначального аудита не закрыты. Исторические доказательства приглашений (прежний этап 06) сохранены выше.

Проверочные БД repet_player_task06_checks и repet_player_task06_migration удалены. Изолированный preview задачи 06: `http://127.0.0.1:3006/tasks/library`, контейнеры repet-player-task06-web-accepted и repet-player-task06-api, API :18086, сеть repet-player-task06 и синтетическая БД repet_player_task06. Во время замены своего preview был name-conflict из-за асинхронного --rm и ранний browser ECONNREFUSED; создан контейнер с новым именем и весь финальный browser-сценарий повторён успешно. Основные web/API и пользовательская БД не заменялись. Push/PR/deployment не выполнялись; задача 07 не начиналась.

## Исправление перехода на вход (2026-10-20)

По отдельному запросу «Уже есть аккаунт?» объединён с «Войти» в одну ссылку-кнопку PublicShell; под формой регистрации весь текст также стал кнопкой `/login`. Изменение не затрагивает auth API и не закрывает общий критерий 15.

Проверено: `auth-navigation.e2e.mjs` через Playwright/Chrome — exit 0, desktop 1440px и touch 375px, клик именно по подписи, Enter и отсутствие overflow (11,4 с). Первый запуск упёрся в холодную dev-компиляцию и timeout загрузки; повтор с ожиданием DOM после компиляции прошёл. `make test` exit 0 (без TEST_DATABASE_URL, PostgreSQL integration SKIP; backend не менялся). `make lint` с Docker npm run вместо native pnpm — exit 0, Go vet/gofmt и ESLint; Docker `tsc --noEmit` exit 0. Существующее ограничение native lint не заявляется исправленным. Отдельный dev-preview: `http://127.0.0.1:3007/register`, контейнер repet-auth-navigation-preview с текущими features read-only. Основные контейнеры не заменялись; push/deploy не выполнялись.

## Задача 07: интерактивные домашние задания (2026-10-29)

В существующие assignments добавлен kind=material без удаления text. Преподаватель выбирает активного ученика, ищет опубликованный материал, назначает урок/подборку, задаёт срок и необязательное занятие календаря; сохраняет и повторно использует личные наборы. Ученик видит назначение на главной и в заданиях, выполняет его через LearningPlayer в `/tasks/[assignmentId]`. Преподаватель получает ответы, ошибки, число отправок по каждому упражнению, историю с пагинацией и ручную проверку; повторение создаёт отдельное назначение с прежним snapshot, не затирая результат.

Миграция 0010 связывает material_assignments/exercise_sets/learning_attempts с существующими заданиями. Snapshot и выбранные упражнения не меняются после публикации/правки/удаления набора; назначение — отдельный run, не свободная тренировка. Подборки ограничены одной версией одного урока и 200 упражнениями. Новые верхние пункты меню не добавлены. Календарная lessons не заменена; проверяется совпадение организации и пары tutor/student. Подробнее: `docs/LEARNING_ASSIGNMENTS.md`.

Доступ проверяется по ресурсу: только назначивший tutor и назначенный student при активной связи; другой преподаватель, ученик, организация и admin не получают material grant. Автор общего материала не видит чужие назначенные ответы и не может их проверять вместо назначившего tutor. Выданный snapshot остаётся доступным после архивирования каталога, но блокировка связи закрывает назначение/run/results; общая история свободной практики не выдаёт assigned runs. Исключённые упражнения нельзя отправить или раскрыть через hints. Контрольная не раскрывает правильность/feedback/ошибки до завершения; отправки через другое назначение/свободную практику не обходят активный exam. Server-owned поля не добавлены во входные DTO.

### Проверки 07

| Команда / сценарий | Фактический результат |
|---|---|
| `TEST_DATABASE_URL=… make test` на отдельной repet_assignment_task07 | exit 0; HTTP/PostgreSQL 40,0 с, не SKIP; неизменённые пакеты cached |
| `TEST_DATABASE_URL=… go test -race ./... -count=1` | exit 0; все core проверки заново, HTTP 50,6 с, store 19,0 с |
| После последних assertions другой организации: `go test -race ./internal/httpapi -run TestExerciseEnginePostgres -count=1` | exit 0, 21,2 с; включает ACL tutor/student/author/org, scope подборки, nonce/conflict, snapshot, личные наборы, ручную проверку, отдельное повторение, exam и запрет обхода |
| `make lint PNPM="docker run --rm -v …/tests:/app/tests:ro repet-assignment-task07-web-accepted npm run"` | exit 0: native Go vet/gofmt и package.json lint через Docker. Native pnpm lint не объявляется исправленным |
| Docker ESLint, `tsc --noEmit`, `node --experimental-strip-types --test tests/*.test.mjs` | exit 0; 16 групп PASS, 0 SKIP; есть runtime-проверка assignment_id |
| Сборка API через исходный Dockerfile | exit 0, образ repet-assignment-task07-api-final; исходники сервера соответствуют проверенному runtime |
| Next production build / TypeScript | exit 0. Чистая попытка Dockerfile остановлена после длительных сетевых ошибок pnpm download; повторная сборка использует имеющийся проверенный dependency image задачи 06 с неизменными package.json/lock, копирует текущие исходники и заново выполняет next build. Новые зависимости не добавлены |
| Playwright/Chrome, последовательно auth-navigation + learning-player + material-assignments против :3008 | exit 0; 3 сценария PASS, 0 SKIP, 99,1 с: вход 19,2 с, плеер 28,2 с, назначения 48,8 с. В 07 проверены saved selection/retry, главная/задания ученика, mobile resume/решение, ошибка и исправление, ручная проверка, повторение, целый урок, text coexistence и mobile composer |
| Миграция 0010 down/up на пустой repet_assignment_task07_migration | exit 0 после исправления порядка drop индексов; assignment_id восстановлен. На заполненной проверочной БД down отклоняется exit 3, транзакция сохраняет данные |
| OpenAPI YAML и `$ref` | exit 0; 5 документов, 319 локальных ссылок разрешены |

Скриншоты teacher desktop 1440px и student/teacher mobile 375px просмотрены; горизонтального overflow и JS page errors нет, запросов answer-keys от ученика нет. Артефакты: `tmp/task07-assignments`. Начальные browser-прогоны уточнили доступные имена select, ожидание навигации и generic-ошибку 503; dev-прогон не засчитан из-за запрета Next dev cross-origin, финальная приёмка выполняется на production-сборке. После осмотра мобильного skeleton список материалов получил компактное состояние загрузки: финальный production build repet-assignment-task07-web-final exit 0, browser 07 повторён полностью — exit 0, 36,1 с, mobile screenshot просмотрен. Runtime source SHA-256 `100992387da95f12b1e52654799ece569527d87ce81525b41308d01e4fdbbc32` совпал с рабочими app/features/shared/lesson-content и build-конфигурацией.

Обычный `git diff --check` недоступен: exit 128, `.git/index: unable to map index file: Operation timed out`. Попытка безопасно прочитать копию индекса тоже получила файловый timeout; индекс не изменялся, пустая временная копия удалена. Для 34 затронутых файлов выполнен `git diff --no-index --check /dev/null <file>` с проверкой stdout/stderr и допустимого exit 1 для разных файлов — нарушений whitespace нет. Это не выдаётся за успешный обычный git diff.

Проверочные БД repet_assignment_task07 и repet_assignment_task07_migration удалены. Для просмотра оставлена только синтетическая repet_assignment_task07_preview на отдельном PostgreSQL-контейнере repet-assignment-task07-postgres (:15437), сеть repet-assignment-task07, API repet-assignment-task07-api-final (:18088) и web repet-assignment-task07-web-final (:3008): `http://127.0.0.1:3008/tasks`. Пользовательские БД/контейнеры не заменялись; push/PR/deployment не выполнялись. Следующий этап не начинался. Ограничения первоначального auth/security-аудита и старых browser-тестов 03–05 не объявляются закрытыми этой задачей.

## Журнал

- 2026-10-29, задача 07: интерактивные домашние задания DONE по новому запросу; назначение, решение, результаты/проверка и повторение подтверждены PostgreSQL/race и браузером. Текстовые задания сохранены; ограничения старого dashboard/profile-критерия сохранены отдельно. Следующая задача не начиналась.

- 2026-10-19: завершён аудит 00; подтверждены базовые этапы 01, 02, 03, 06. Всего 5 DONE, 11 IN_PROGRESS, 4 TODO, 0 BLOCKED. Это не процент готовности продукта. Новые продуктовые функции, push, PR и deployment не выполнялись.
- 2026-10-19, задача 01: по новому запросу реализован и проверен backend учебного каталога. Новый критерий 01 выполнен; статус DONE. Другие этапы не выполнялись. Итоги проверок и ограничение нативного lint записаны отдельным разделом выше.
- 2026-10-19, задача 02: выполнен интерфейс библиотеки поверх API 01; критерий создания/публикации преподавателем и поиска/чтения учеником подтверждён браузером. Статус DONE. Задача 03 не начиналась; её прежний статус сохранён.
- 2026-10-19, задача 03: реализован и проверен универсальный движок пяти базовых типов с серверными попытками и авторской ручной проверкой; DONE. Исторические доказательства frontend сохранены.
- 2026-10-19, задача 04: добавлены и проверены пять визуальных типов, включая три режима geometry; DONE. Последующие задачи не запускались. Ограничение нативного lint и границы геометрии/числовой прямой указаны выше.
- 2026-10-20: по отдельному запросу добавлены 10 HTML в мастерскую, теперь 20 тем. Новые маршруты, ACL и trainer/mobile smoke проверены; общий этап 12 остаётся IN_PROGRESS, его полный критерий не заявляется выполненным.
- 2026-10-20, задача 05: выполнены сохраняемые прохождения и результаты, критерий трёх ответов после закрытия вкладки проверен браузером; DONE по новому запросу. Прежний auth/security-критерий сохранён как незакрытые ограничения аудита, не смешивается с новой готовностью прохождений.
- 2026-10-20, задача 06: LearningPlayer DONE по новому запросу; практика/контрольная, авторский пример и mobile-продолжение проверены; исторические проверки приглашений сохранены. Следующий этап не начинался.

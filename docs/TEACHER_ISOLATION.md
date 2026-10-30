# Независимые Преподаватели

Задача16 **IN_PROGRESS**, не завершена настройками. Первый проверенный срез2026-10-30: приватные предпочтения и устранение утечки в standalone попытках открытых учебных шаблонов. Sharing/copy/licensing/private visibility и полная сквозная матрица остаются следующими шагами.

## Личные Настройки

`GET/PUT /api/v1/teaching-preferences`: tutor-only, владелец/организация только из сессии. GET без сохранения даёт revision0, math, классы1–11 и60мин. PUT — CAS, при устаревшей revision409. Один–десять предметов из allowlist, один–одиннадцать уникальных классов1–11, предпочтительная длительность1–240мин. Unknown/server-owned поля400. Другой преподаватель, даже той же организации, получает свой документ/defaults; query-подстановка tutor_id не меняет scope. Ученик403.

Миграция0023 создаёт отдельную teaching_preferences, не изменяет auth roles и организации. Настройки доступны в «Личный кабинет → Преподавание», checkbox предметов/классов, numeric duration, loading/error/retry, conflict без перезаписи другой вкладки, reload/persistence. Это предпочтения; они пока не переписывают существующие уроки и настройки планировщика, не ограничивают доступ к опубликованной библиотеке и не обещают автоматического расписания.

## Автор Не Равен Преподавателю

Открытый материал не даёт его автору права читать личные решения всех посетителей. Standalone GET lesson/attempts теперь фильтрует преподавателя по собственной действующей связи с конкретным учеником. Попытки чужих assignments/study/review как раньше исключены из этого списка и доступны только через свои контексты. Автор шаблона не может проверять решение ребёнка после прекращения связи.

Objective ответы по общедоступному материалу по-прежнему проверяются сервером без необходимости связывать ученика с автором. Long_text требует активного преподавателя соответствующего контекста: assignment/review tutor либо автор standalone материала. Перед сохранением/уведомлением эта relation блокируется FOR SHARE. При её отсутствии404: ответ не сохраняется и чужому автору не отправляются имя/уведомление о ребёнке. Ранее сам факт публикации открывал author-list/review и маршрутизировал ему manual notification; этот обход закрыт. Нужен понятный UI-сценарий для самостоятельного посетителя чужого public long_text при дальнейшей реализации sharing/copy, не ослабление backend ACL.

## Следующие Шаги

1. Published-only private sharing конкретного native материала с другим преподавателем, отзыв доступа и пагинация, без расширения доступа ко всему курсу или исходным ученикам.
2. Авторское разрешение копирования/лицензия/атрибуция; copy snapshot в собственный draft с новыми IDs и server-only копией ключей. PDF/media требуют отдельного ACL пути, не утечки asset IDs источника. Не отдавать закрытые ответы recipient/public API.
3. Private/linked_students/organization availability в согласованных catalog/study/review/chat/HTML/media правилах, UI управления доступом. Новое private значение нельзя добавить только в enum, забыв остальные SQL-предикаты.
4. Полная матрица двух преподавателей/детей/организаций на календарь, задания, чаты/файлы, native/HTML/PDF, analytics/reviews/rewards и shared/copy. Существующие частные тесты не доказывают все критерии16.

## Доказательства Первого Среза

`TestTeachingIsolationPostgres` проверяет реальные cookie-сессии двух tutors/двух students и отдельной организации, CAS/неизвестные поля/валидаторы/teacher own settings, public readonly без edit, отсутствие чужих objective попыток и manual notification, собственный manual/revoke404. Focused2,854сPASS; exercise regression6,875сPASS. Focused race TeachingIsolation+ExerciseEngine35,915сPASS в отдельной synthetic DB. Полный make test в новой repet_tenants16_final_checks exit0 HTTP28,861/store2,414с; остальные cached явно. Первоначальный full run выявил test-assumption len=1 на глобальных public карточках из других fixtures: ChatAttachments теперь ищет уникальное название; production доступ не сужался ради теста.

Make lint native vet/gofmt и контейнерный текущий frontend+tests exit0,0errors/5прежних imgwarnings. Production Next79,2с/TS19,0сPASS; digest host/image2aaac635097872d5a4fac6221d660440eb97940f74019a72762c51748165eded. Browser2PASS44,880с: основной integration36,387с и preferences7,248с (persistence, другой tutor/defaults, CAS/reload, student403,320–1440/0overflow/0JSerrors). Скриншоты tmp/tenants16 просмотрены.0001–0023up/23emptydown-upPASS; populated23downexpected3/exportguard.15YAML475refs разрешены, не полный OpenAPI validator. Финальные unit/whitespace записываются в трекер отдельно.

Текущий preview3020 сохраняет прежнюю собственную repet_integration_preview и named volumes, но web/API image обновлены до repet-tenants16-web/api; имена контейнеров repet-messages15-* для сохранения routing. Пользовательский repet_postgres-data не изменялся. Test/race/migrations DB repet_tenants16_* отдельные synthetic. Sharing/copy ещё не реализованы,17не начинается. Никакого публичного deployment/push/PR/production secrets.

Окончательный visual-fix: checkbox padding/min-height больше не наследует46px text-input styling, grade10/11 проверены на nowrap/label bounds. FinalNext48,4с/TS15,9сPASS, image repet-tenants16-web-final (alias repet-tenants16-web), source digest de9ac0efac443053c14f5391088431190f81d4dc10693bc8cc8dfb7f66553b61 совпал. Finalpreferences browser12,501сPASS,36unitPASS0SKIP1,457с, final make lint exit0/5прежних warnings и make test repeat exit0cached.21files no-indexwhitespacePASS; Git index timeout128не изменялся. Новые personal settings не закрывают критерии sharing/copy или всей16.

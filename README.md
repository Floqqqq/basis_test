# Task Manager

Web-приложение и REST API для управления задачами в командах.

Проект состоит из Go backend, React frontend и асинхронного notification worker. Он поддерживает регистрацию пользователей, JWT-аутентификацию, командную работу, ролевую модель `owner/admin/member`, создание и обновление задач, комментарии, историю изменений задач, email-уведомления, Redis-кеширование, rate limiting, Prometheus-метрики, OpenTelemetry tracing, SQL-отчёты на PostgreSQL, Docker Compose и тесты.

## Описание проекта

Сервис предназначен для управления задачами внутри команд.

Пользователь может:

* зарегистрироваться;
* авторизоваться и получить JWT;
* создать команду;
* пригласить другого пользователя в команду;
* создать задачу внутри команды;
* назначить задачу на участника команды;
* получить список задач с фильтрацией и пагинацией;
* обновить задачу;
* посмотреть историю изменений задачи;
* получить SQL-отчёты по командам и задачам.

В проекте реализованы обязательные части ТЗ:

* Go backend;
* PostgreSQL;
* Redis;
* Docker и Docker Compose;
* регистрация и логин;
* JWT-аутентификация;
* команды;
* роли в командах;
* задачи;
* история изменений задач;
* таблица комментариев к задачам;
* Redis-кеширование списка задач команды;
* сложные SQL-запросы;
* индексы PostgreSQL;
* connection pooling;
* пагинация на уровне БД;
* unit-тесты;
* интеграционные тесты с PostgreSQL через testcontainers;
* domain events и transactional outbox;
* асинхронный worker и SMTP email-уведомления;
* rate limiting;
* graceful shutdown;
* Prometheus-метрики;
* конфигурация через YAML и ENV.

## Стек технологий

| Компонент             | Технология                          |
| --------------------- | ----------------------------------- |
| Язык                  | Go 1.22                             |
| Frontend              | React, TypeScript, Vite             |
| Работа с API          | TanStack Query, React Router        |
| HTTP router           | `github.com/go-chi/chi/v5`          |
| База данных           | PostgreSQL 16                       |
| PostgreSQL driver     | `github.com/jackc/pgx/v5`           |
| Кеш                   | Redis                               |
| Redis client          | `github.com/redis/go-redis/v9`      |
| Авторизация           | JWT, `github.com/golang-jwt/jwt/v5` |
| Хеширование паролей   | bcrypt                              |
| Метрики               | Prometheus                          |
| Email                 | SMTP, Mailpit для локальной разработки |
| Тесты repository-слоя | `sqlmock`, `testcontainers-go`      |
| Redis-тесты           | `redismock`                         |
| Контейнеризация       | Docker, Docker Compose              |
| Конфигурация          | YAML + ENV                          |

## Архитектура проекта

```text
.
├── backend/
│   ├── cmd/
│   │   ├── api/
│   │   └── worker/
│   ├── internal/
│   │   ├── handlers/
│   │   ├── repository/
│   │   └── service/
│   ├── migrations/
│   ├── Dockerfile
│   ├── go.mod
│   └── go.sum
├── frontend/
│   ├── src/
│   ├── Dockerfile
│   └── package.json
├── monitoring/
│   ├── prometheus/
│   └── grafana/
├── docker-compose.yml
├── Makefile
└── README.md
```

Назначение основных директорий:

| Путь                  | Назначение                                                                                             |
| --------------------- | ------------------------------------------------------------------------------------------------------ |
| `backend/cmd/api`             | Точка входа API, настройка router, запуск HTTP-сервера и graceful shutdown                         |
| `backend/cmd/worker`          | Асинхронная обработка outbox, worker metrics и graceful shutdown                                  |
| `backend/internal/config`     | Загрузка конфигурации из YAML и ENV                                                               |
| `backend/internal/db`         | Подключение к PostgreSQL, retry, connection pooling                                                |
| `backend/internal/events`     | Типы domain events, metadata и payload                                                             |
| `backend/internal/redis`      | Подключение к Redis с retry                                                                        |
| `backend/internal/cache`      | Redis-кеширование списка задач                                                                     |
| `backend/internal/handlers`   | HTTP handlers, валидация request body/query params и JSON-ответы                                   |
| `backend/internal/middleware` | JWT middleware, rate limiting и Prometheus middleware                                              |
| `backend/internal/models`     | Основные модели данных                                                                             |
| `backend/internal/notifications` | Recipient policy, email templates, SMTP sender, retry, worker и metrics                         |
| `backend/internal/repository` | Работа с PostgreSQL, транзакции и notification outbox                                              |
| `backend/internal/service`    | Бизнес-логика, права доступа и формирование domain events                                           |
| `backend/internal/telemetry`  | OpenTelemetry SDK, OTLP exporter и HTTP instrumentation                                            |
| `backend/migrations`          | SQL-схема, связи и индексы                                                                         |
| `frontend`                    | Самостоятельный npm-модуль React: страницы, API client, маршрутизация и стили                       |
| `monitoring`                  | Конфигурация Prometheus и автоматически provisioned Grafana dashboard                              |

## Быстрый запуск

Запуск проекта:

```bash
docker compose up --build
```

После запуска интерфейс доступен по адресу:

```text
http://localhost:3000
```

API доступно по адресу:

```text
http://localhost:18080
```

Внутри контейнера приложение слушает порт `8080`.

На хост по умолчанию публикуется порт `18080`, чтобы не конфликтовать с локальными приложениями на `8080`.

### Запуск frontend для разработки

```bash
cd frontend
npm install
npm run dev
```

Vite откроет интерфейс на `http://localhost:5173` и перенаправит запросы `/api` на backend по адресу `http://localhost:18080`.

### Запуск backend для разработки

Backend является отдельным Go-модулем:

```bash
cd backend
go run ./cmd/api
```

Зависимости frontend и backend устанавливаются независимо: через `npm` в `frontend/` и через Go Modules в `backend/`. В корне проекта нет общего application-модуля, корень используется только для Docker Compose и общих команд.

## Порты по умолчанию

| Сервис | Внутренний порт | Host-порт |
| ------ | --------------: | --------: |
| Frontend |          `80` |    `3000` |
| API    |          `8080` |   `18080` |
| PostgreSQL |       `5432` |    `5433` |
| Redis  |          `6379` |    `6380` |
| OTLP Collector | `4317` | `4317` |
| Jaeger UI |       `16686` | `16686` |
| Prometheus UI |     `9090` | `9090` |
| Grafana UI |        `3000` | `3001` |
| Worker metrics |   `9091` | `19091` |
| Mailpit SMTP |      `1025` | `1025` |
| Mailpit UI |        `8025` | `8025` |

Если нужно поменять host-порт API:

```bash
APP_HOST_PORT=18081 docker compose up --build
```

Если нужно запустить API именно на `localhost:8080`:

```bash
APP_HOST_PORT=8080 docker compose up --build
```

Если нужно поменять все host-порты:

```bash
FRONTEND_HOST_PORT=3000 APP_HOST_PORT=18080 POSTGRES_HOST_PORT=5433 REDIS_HOST_PORT=6380 docker compose up --build
```

## Docker Compose

В `docker-compose.yml` поднимаются:

* приложение Go;
* notification worker;
* React frontend с Nginx;
* PostgreSQL 16;
* Redis 7;
* OpenTelemetry Collector 0.157.0;
* Jaeger 1.76.0.
* Mailpit 1.30.7.
* Prometheus 2.53.1;
* Grafana 11.2.0 с автоматически подключённым dashboard.

PostgreSQL и Redis имеют healthcheck. Приложение стартует после того, как PostgreSQL и Redis становятся healthy.

SQL-миграции из директории `backend/migrations/` автоматически применяются при первом создании контейнера PostgreSQL:

```text
backend/migrations/001_init.sql
backend/migrations/002_indexes.sql
backend/migrations/003_task_comments_ordering_index.sql
backend/migrations/004_notification_outbox.sql
backend/migrations/005_outbox_worker_index.sql
backend/migrations/006_team_leave_requests.sql
backend/migrations/007_activity_events.sql
```

Если база уже была создана раньше, PostgreSQL не применит init scripts повторно. Чтобы пересоздать БД с нуля:

```bash
docker compose down -v
docker compose up --build
```

## Конфигурация

Проект поддерживает конфигурацию через YAML и ENV.

Файл backend-модуля по умолчанию:

```text
backend/config.yaml
```

Пример `config.yaml`:

```yaml
app_port: "8080"
postgres_dsn: "postgres://postgres:postgres@localhost:5433/task_manager?sslmode=disable"
redis_addr: "localhost:6379"
jwt_secret: "secret"
otel_enabled: false
otel_service_name: "task-manager-api"
otel_exporter_otlp_endpoint: "localhost:4317"
otel_exporter_otlp_insecure: true
smtp_host: "localhost"
smtp_port: 1025
smtp_username: ""
smtp_password: ""
smtp_from: "no-reply@task-manager.local"
worker_metrics_port: "9091"
```

Путь к YAML-файлу можно переопределить:

```bash
cd backend
CONFIG_PATH=./config.yaml go run ./cmd/api
```

ENV-переменные имеют приоритет над YAML.

| Переменная        | Описание                                     | Значение по умолчанию                                       |
| ----------------- | -------------------------------------------- | ----------------------------------------------------------- |
| `CONFIG_PATH`     | Путь к YAML-конфигу                          | `backend/config.yaml` при запуске из корня                   |
| `APP_PORT`        | Порт HTTP-сервера внутри контейнера/процесса | `8080`                                                      |
| `APP_HOST_PORT`   | Host-порт API в Docker Compose               | `18080`                                                     |
| `FRONTEND_HOST_PORT` | Host-порт frontend в Docker Compose       | `3000`                                                      |
| `POSTGRES_DSN`       | DSN подключения к PostgreSQL                 | `postgres://postgres:postgres@localhost:5432/task_manager?sslmode=disable` |
| `POSTGRES_HOST_PORT` | Host-порт PostgreSQL в Docker Compose        | `5433`                                                      |
| `REDIS_ADDR`      | Адрес Redis для приложения                   | `localhost:6379`                                            |
| `REDIS_HOST_PORT` | Host-порт Redis в Docker Compose             | `6380`                                                      |
| `JWT_SECRET`      | Секрет для подписи JWT                       | `secret`                                                    |
| `OTEL_ENABLED` | Включить отправку traces | `false` |
| `OTEL_SERVICE_NAME` | Имя сервиса в tracing backend | `task-manager-api` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | Адрес OTLP gRPC Collector | `localhost:4317` |
| `OTEL_EXPORTER_OTLP_INSECURE` | Отключить TLS для локального OTLP | `true` |
| `OTEL_COLLECTOR_GRPC_PORT` | Host-порт локального Collector | `4317` |
| `JAEGER_UI_PORT` | Host-порт Jaeger UI | `16686` |
| `PROMETHEUS_HOST_PORT` | Host-порт Prometheus UI | `9090` |
| `GRAFANA_HOST_PORT` | Host-порт Grafana UI | `3001` |
| `GRAFANA_ADMIN_USER` | Локальный администратор Grafana | `admin` |
| `GRAFANA_ADMIN_PASSWORD` | Локальный пароль Grafana | `admin` |
| `SMTP_HOST` | SMTP server host | `localhost` |
| `SMTP_PORT` | SMTP server port | `1025` |
| `SMTP_USERNAME` | SMTP username | пусто |
| `SMTP_PASSWORD` | SMTP password, не выводится в логах | пусто |
| `SMTP_FROM` | Адрес отправителя | `no-reply@task-manager.local` |
| `WORKER_METRICS_PORT` | Порт metrics внутри worker | `9091` |
| `WORKER_METRICS_HOST_PORT` | Host-порт worker metrics | `19091` |
| `MAILPIT_SMTP_HOST_PORT` | Host-порт локального SMTP | `1025` |
| `MAILPIT_UI_HOST_PORT` | Host-порт Mailpit UI | `8025` |

В Docker Compose для приложения используются значения:

```yaml
APP_PORT: "8080"
JWT_SECRET: "secret"
POSTGRES_DSN: "postgres://postgres:postgres@postgres:5432/task_manager?sslmode=disable"
REDIS_ADDR: "redis:6379"
OTEL_ENABLED: "true"
OTEL_SERVICE_NAME: "task-manager-api"
OTEL_EXPORTER_OTLP_ENDPOINT: "otel-collector:4317"
OTEL_EXPORTER_OTLP_INSECURE: "true"
```

## База данных

Схема создаётся миграцией:

```text
backend/migrations/001_init.sql
```

Индексы создаются миграциями:

```text
backend/migrations/002_indexes.sql
backend/migrations/003_task_comments_ordering_index.sql
backend/migrations/004_notification_outbox.sql
backend/migrations/005_outbox_worker_index.sql
backend/migrations/006_team_leave_requests.sql
backend/migrations/007_activity_events.sql
```

Основные таблицы:

| Таблица         | Назначение                                                                      |
| --------------- | ------------------------------------------------------------------------------- |
| `users`         | Пользователи системы                                                            |
| `teams`         | Команды                                                                         |
| `team_members`  | Связь пользователей и команд many-to-many, содержит роль пользователя в команде |
| `tasks`         | Задачи команды                                                                  |
| `task_history`  | История изменений задач                                                         |
| `task_comments` | Комментарии к задачам                                                           |
| `notification_outbox` | События для асинхронной обработки уведомлений                              |
| `team_leave_requests` | Заявки участников на выход из команды                                      |
| `activity_events` | Постоянная пользовательская лента событий команды                               |

Связи:

| Связь                                 | Описание                      |
| ------------------------------------- | ----------------------------- |
| `teams.created_by -> users.id`        | Создатель команды             |
| `team_members.user_id -> users.id`    | Пользователь в команде        |
| `team_members.team_id -> teams.id`    | Команда участника             |
| `tasks.assignee_id -> users.id`       | Исполнитель задачи            |
| `tasks.team_id -> teams.id`           | Команда задачи                |
| `tasks.created_by -> users.id`        | Автор задачи                  |
| `task_history.task_id -> tasks.id`    | История конкретной задачи     |
| `task_history.changed_by -> users.id` | Автор изменения               |
| `task_comments.task_id -> tasks.id`   | Комментарии конкретной задачи |
| `task_comments.user_id -> users.id`   | Автор комментария             |

## Индексы PostgreSQL

Индексы вынесены в миграцию:

```text
backend/migrations/002_indexes.sql
```

Они добавлены для ускорения:

* фильтрации задач по команде;
* фильтрации задач по статусу;
* фильтрации задач по исполнителю;
* пагинации списка задач;
* выборки истории задачи;
* SQL-отчётов;
* проверки участников команды;
* работы с foreign key полями.

## Ролевая модель

В проекте есть три роли:

```text
owner
admin
member
```

### owner

`owner` создаётся автоматически при создании команды.

Права:

* может приглашать пользователей;
* может назначать роли `admin` и `member`;
* может создавать задачи;
* может обновлять любую задачу команды;
* может смотреть историю любой задачи команды.

### admin

Права:

* может приглашать пользователей;
* может назначать роли `admin` и `member`;
* может создавать задачи;
* может обновлять любую задачу команды;
* может смотреть историю любой задачи команды.

### member

Права:

* может создавать задачи в команде;
* может обновлять задачу, если он является автором или исполнителем;
* может смотреть историю задачи, если он является автором или исполнителем;
* не может приглашать пользователей.

## Правила доступа

Основные правила:

* пользователь может видеть только команды, где он состоит;
* пользователь может создавать задачи только в команде, где он состоит;
* `assignee_id` можно не передавать;
* если `assignee_id` передан, этот пользователь должен состоять в команде задачи;
* нельзя создать или переназначить задачу на пользователя вне команды;
* `owner` и `admin` могут обновлять любую задачу команды;
* `member` может обновлять только задачи, где он creator или assignee;
* `owner` не выдаётся через invite, owner появляется только при создании команды.

## API

Все ответы возвращаются в JSON.

Ошибки имеют формат:

```json
{
  "error": "message"
}
```

Защищённые endpoint требуют заголовок:

```text
Authorization: Bearer <token>
```

## Регистрация

```text
POST /api/v1/register
```

JWT не нужен.

Request body:

```json
{
  "email": "user1@example.com",
  "password": "pass123"
}
```

Response body:

```json
{
  "user_id": 1,
  "token": "jwt-token"
}
```

Пример:

```bash
curl -s -X POST http://localhost:18080/api/v1/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"user1@example.com","password":"pass123"}'
```

## Логин

```text
POST /api/v1/login
```

JWT не нужен.

Request body:

```json
{
  "email": "user1@example.com",
  "password": "pass123"
}
```

Response body:

```json
{
  "user_id": 1,
  "token": "jwt-token"
}
```

Пример:

```bash
curl -s -X POST http://localhost:18080/api/v1/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"user1@example.com","password":"pass123"}'
```

## Текущий пользователь

```text
GET /api/v1/me
```

JWT нужен. Ответ содержит `id`, `email` и `created_at`. Хеш пароля в API не возвращается.

## Создание команды

```text
POST /api/v1/teams
```

JWT нужен.

Request body:

```json
{
  "name": "Backend Team"
}
```

Response body:

```json
{
  "team_id": 1
}
```

Пример:

```bash
curl -s -X POST http://localhost:18080/api/v1/teams \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"name":"Backend Team"}'
```

## Получение списка команд

```text
GET /api/v1/teams
```

JWT нужен.

Response body:

```json
[
  {
    "id": 1,
    "name": "Backend Team",
    "created_by": 1,
    "created_at": "2026-06-22T15:00:00Z",
    "role": "owner"
  }
]
```

Пример:

```bash
curl -s http://localhost:18080/api/v1/teams \
  -H "Authorization: Bearer $TOKEN"
```

## Участники команды

```text
GET /api/v1/teams/{id}/members
```

JWT нужен. Endpoint доступен только участнику команды и возвращает `id`, `email`, `role` и `joined_at` каждого участника.

## Приглашение пользователя в команду

```text
POST /api/v1/teams/{id}/invite
```

JWT нужен.

Доступно только ролям:

* `owner`;
* `admin`.

Request body:

```json
{
  "user_id": 2,
  "role": "member"
}
```

Допустимые роли для invite:

```text
admin
member
```

Response body:

```json
{
  "status": "invited"
}
```

Пример:

```bash
curl -s -X POST http://localhost:18080/api/v1/teams/1/invite \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"user_id":2,"role":"member"}'
```

Если пользователя не существует, API возвращает `404`.

Если пользователь уже состоит в команде, API возвращает `409`.

## Управление составом команды

Владелец может удалить администратора или обычного участника. Администратор может удалить только обычного участника. Владельца удалить нельзя.

```text
DELETE /api/v1/teams/{id}/members/{user_id}
```

Обычный участник или администратор отправляет заявку на выход:

```text
POST /api/v1/teams/{id}/leave-requests
```

Владелец и администраторы получают общий список активных заявок:

```text
GET /api/v1/teams/{id}/leave-requests
```

Решение по заявке:

```text
POST /api/v1/teams/{id}/leave-requests/{request_id}/decision
```

Request body:

```json
{
  "decision": "approve"
}
```

Допустимые решения: `approve` и `reject`. Заявка блокируется на время обработки через PostgreSQL `FOR UPDATE`; после первого решения она исчезает из активного списка, а повторная попытка возвращает `409`.

## Активность команды

```text
GET /api/v1/teams/{id}/activity?limit=20&offset=0
```

JWT нужен. Лента доступна только участникам команды и отсортирована по `created_at DESC, id DESC`. Она хранится в отдельной таблице `activity_events` и не использует transport-очередь `notification_outbox` как пользовательское хранилище.

Каждая запись содержит тип события, автора, сущность, исходный payload, человекочитаемое сообщение и время. В интерфейсе лента находится во вкладке «Активность».

## Real-time обновления

```text
GET /api/v1/teams/{id}/ws
```

WebSocket используется только как сигнал об изменении. JWT передаётся вторым значением WebSocket subprotocol после `access_token`; перед upgrade backend проверяет участие пользователя в команде. События другой команды в соединение не отправляются.

Frontend автоматически переподключается после разрыва. При получении события он инвалидирует TanStack Query cache, после чего актуальные данные загружаются через REST API.

## Создание задачи

```text
POST /api/v1/tasks
```

JWT нужен.

Request body:

```json
{
  "team_id": 1,
  "title": "Implement API",
  "description": "Finish task endpoints",
  "status": "todo",
  "assignee_id": 2
}
```

Поля:

| Поле          | Обязательное | Описание                           |
| ------------- | ------------ | ---------------------------------- |
| `team_id`     | Да           | ID команды                         |
| `title`       | Да           | Название задачи                    |
| `description` | Нет          | Описание задачи                    |
| `status`      | Нет          | Статус задачи, по умолчанию `todo` |
| `assignee_id` | Нет          | ID исполнителя                     |

Допустимые статусы:

```text
todo
in_progress
done
```

Response body:

```json
{
  "task_id": 1
}
```

Пример:

```bash
curl -s -X POST http://localhost:18080/api/v1/tasks \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"team_id":1,"title":"Implement API","description":"Finish endpoints","status":"todo","assignee_id":2}'
```

## Получение списка задач

```text
GET /api/v1/tasks?team_id=1&status=todo&assignee_id=2&limit=20&offset=0
```

JWT нужен.

Query parameters:

| Параметр      | Обязательный | Описание                                           |
| ------------- | ------------ | -------------------------------------------------- |
| `team_id`     | Да           | ID команды                                         |
| `status`      | Нет          | Фильтр по статусу                                  |
| `assignee_id` | Нет          | Фильтр по исполнителю                              |
| `limit`       | Нет          | Размер страницы, по умолчанию `20`, максимум `100` |
| `offset`      | Нет          | Смещение, по умолчанию `0`                         |

Response body:

```json
[
  {
    "id": 1,
    "title": "Implement API",
    "description": "Finish task endpoints",
    "status": "todo",
    "assignee_id": 2,
    "completed_at": null,
    "team_id": 1,
    "created_by": 1,
    "created_at": "2026-06-22T15:00:00Z",
    "updated_at": "2026-06-22T15:00:00Z"
  }
]
```

Пример:

```bash
curl -s 'http://localhost:18080/api/v1/tasks?team_id=1&status=todo&assignee_id=2&limit=20&offset=0' \
  -H "Authorization: Bearer $TOKEN"
```

## Получение задачи

```text
GET /api/v1/tasks/{id}
```

JWT нужен. Задачу может получить любой участник её команды.

## Обновление задачи

```text
PUT /api/v1/tasks/{id}
```

JWT нужен.

Обновлять задачу могут:

* `owner`;
* `admin`;
* creator задачи;
* assignee задачи.

Request body:

```json
{
  "title": "Updated title",
  "description": "Updated description",
  "status": "done",
  "assignee_id": 2
}
```

Все поля опциональны, но нужно передать хотя бы одно поле.

Если передан `assignee_id`, пользователь должен состоять в команде задачи.

Значение `null` снимает исполнителя, а отсутствие поля оставляет текущего исполнителя без изменений.

При изменении задачи записывается история изменений в таблицу `task_history`.

Если статус меняется на `done`, поле `completed_at` заполняется текущим временем.

Если статус меняется с `done` на другой статус, поле `completed_at` сбрасывается.

Response body:

```json
{
  "status": "updated"
}
```

Пример:

```bash
curl -s -X PUT http://localhost:18080/api/v1/tasks/1 \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"status":"done"}'
```

Пример переназначения задачи:

```bash
curl -s -X PUT http://localhost:18080/api/v1/tasks/1 \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"assignee_id":2}'
```

Снять исполнителя:

```bash
curl -s -X PUT http://localhost:18080/api/v1/tasks/1 \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"assignee_id":null}'
```

## Комментарии задачи

```text
POST /api/v1/tasks/{id}/comments
GET /api/v1/tasks/{id}/comments
```

JWT нужен. Читать и создавать комментарии могут участники команды задачи. Текст обязателен, максимальная длина - 4000 символов.

```json
{
  "comment": "Нужно проверить этот кейс"
}
```

Ответ содержит `id`, `task_id`, `user_id`, `email`, `text` и `created_at`.

## История задачи

```text
GET /api/v1/tasks/{id}/history
```

JWT нужен.

Историю задачи могут смотреть:

* `owner`;
* `admin`;
* creator задачи;
* assignee задачи.

Response body:

```json
[
  {
    "id": 1,
    "task_id": 1,
    "changed_by": 1,
    "field_name": "status",
    "old_value": "todo",
    "new_value": "done",
    "created_at": "2026-06-22T15:10:00Z"
  }
]
```

Пример:

```bash
curl -s http://localhost:18080/api/v1/tasks/1/history \
  -H "Authorization: Bearer $TOKEN"
```

## SQL-отчёты

Все SQL-отчёты требуют JWT.

Отчёты возвращают данные только по тем командам, где текущий пользователь состоит.

### Статистика команд

```text
GET /api/v1/reports/team-stats
```

Возвращает для каждой доступной команды:

* ID команды;
* название команды;
* количество участников;
* количество задач в статусе `done` за последние 7 дней.

Пример:

```bash
curl -s http://localhost:18080/api/v1/reports/team-stats \
  -H "Authorization: Bearer $TOKEN"
```

### Топ пользователей

```text
GET /api/v1/reports/top-users
```

Возвращает топ-3 пользователей по количеству созданных задач в каждой доступной команде за текущий календарный месяц.

В запросе используется оконная функция PostgreSQL:

```sql
ROW_NUMBER() OVER (
  PARTITION BY team_id
  ORDER BY COUNT(*) DESC
)
```

Пример:

```bash
curl -s http://localhost:18080/api/v1/reports/top-users \
  -H "Authorization: Bearer $TOKEN"
```

### Некорректные исполнители

```text
GET /api/v1/reports/invalid-assignees
```

Возвращает задачи, где `assignee_id` указан, но этот пользователь не состоит в команде задачи.

В нормальной работе такой список должен быть пустым, потому что API запрещает создавать и обновлять задачи с некорректным исполнителем.

Пример:

```bash
curl -s http://localhost:18080/api/v1/reports/invalid-assignees \
  -H "Authorization: Bearer $TOKEN"
```

## Prometheus и Grafana

API и worker публикуют метрики в формате Prometheus. Prometheus автоматически собирает их внутри Docker-сети:

| Target | Endpoint внутри Docker | Endpoint с хоста |
| ------ | ---------------------- | ---------------- |
| API | `app:8080/metrics` | `http://localhost:18080/metrics` |
| Worker | `worker:9091/metrics` | `http://localhost:19091/metrics` |

Prometheus UI и состояние targets:

```text
http://localhost:9090
http://localhost:9090/targets
```

Готовый Grafana dashboard открывается автоматически:

```text
http://localhost:3001/d/task-manager-overview
```

Для локального запуска разрешён анонимный просмотр. Для редактирования можно войти с `admin` / `admin` или переопределить `GRAFANA_ADMIN_USER` и `GRAFANA_ADMIN_PASSWORD`.

Dashboard показывает:

* доступность API и worker;
* HTTP throughput и ошибки;
* p95 времени ответа по endpoint;
* размер очереди outbox;
* обработку notification events;
* отправленные и неотправленные email;
* количество goroutine API и worker.

JWT для `/metrics` не нужен.

Пример:

```bash
curl -s http://localhost:18080/metrics
curl -s http://localhost:19091/metrics
```

Реализованные метрики:

| Метрика                         | Описание                              |
| ------------------------------- | ------------------------------------- |
| `http_requests_total`           | Количество HTTP-запросов              |
| `http_request_duration_seconds` | Histogram времени ответа              |
| `http_errors_total`             | Количество HTTP-ошибок 4xx/5xx        |
| `notification_processed_total`  | Успешно обработанные уведомления       |
| `notification_failed_total`     | Ошибки обработки уведомлений           |
| `email_sent_total`               | Отправленные email                     |
| `email_failed_total`             | Ошибки отправки email                  |
| `outbox_pending`                 | События, ожидающие обработки           |
| Go/process metrics              | Стандартные метрики Prometheus client |

## OpenTelemetry tracing

Приложение отправляет traces по OTLP gRPC через следующую цепочку:

```text
task-manager-api -> OpenTelemetry Collector -> Jaeger
```

Prometheus-метрики продолжают работать независимо от tracing через `GET /metrics`.

При запуске через Docker Compose tracing включён автоматически. Jaeger UI доступен по адресу:

```text
http://localhost:16686
```

Для локального запуска приложения без Collector tracing по умолчанию выключен. Его можно включить так:

```bash
cd backend
OTEL_ENABLED=true \
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317 \
OTEL_EXPORTER_OTLP_INSECURE=true \
go run ./cmd/api
```

HTTP spans используют шаблоны маршрутов, например `GET /api/v1/tasks/{id}`. Внутри них создаются spans значимых операций `TaskService` и `TeamService`.

Для ручной проверки после `docker compose up --build` нужно выполнить login и несколько защищённых запросов, затем выбрать сервис `task-manager-api` в Jaeger UI.

## Redis-кеширование

Redis используется для кеширования списка задач команды.

Кешируется endpoint:

```text
GET /api/v1/tasks
```

TTL кеша:

```text
5 минут
```

Ключ кеша учитывает:

* `team_id`;
* `status`;
* `assignee_id`;
* `limit`;
* `offset`.

Формат ключа:

```text
team_tasks:{team_id}:status={status}:assignee={assignee}:limit={limit}:offset={offset}
```

Пример:

```text
team_tasks:1:status=todo:assignee=2:limit=20:offset=0
```

Инвалидация кеша выполняется:

* при создании задачи;
* при обновлении задачи.

Если Redis недоступен при чтении списка задач, сервис не должен ломать основной сценарий и может получить данные из PostgreSQL.

## Rate limiting

Rate limiting реализован через Redis.

Для публичных endpoint используется лимит по IP:

```text
POST /api/v1/register
POST /api/v1/login
```

Для защищённых endpoint используется лимит по `user_id`.

Правило:

```text
100 запросов в минуту
```

Примеры ключей:

```text
rate_limit:ip:127.0.0.1
rate_limit:1
```

При превышении лимита возвращается:

```http
429 Too Many Requests
```

Response body:

```json
{
  "error": "too many requests"
}
```

Если Redis временно недоступен, rate limiter работает в fail-open режиме: API не блокирует запрос только из-за недоступности Redis.

## Domain events и transactional outbox

Бизнес-операции формируют события `task.created`, `task.updated`, `task.assigned`, `task.status_changed` и `team.member_added`.

Для создания и обновления задач, а также добавления участника бизнес-изменение и событие записываются в одной PostgreSQL-транзакции. События хранятся в таблице `notification_outbox` и не теряются после завершения HTTP-запроса.

`OutboxRepository` поддерживает конкурентную выборку событий через `FOR UPDATE SKIP LOCKED`, lease для зависших `processing` событий, статусы обработки и metadata для повторной попытки.

Отдельный процесс `backend/cmd/worker` получает события batch-ами и передаёт их в `NotificationService`. Получатели, исключение actor и удаление дубликатов определяются централизованно. SMTP скрыт за интерфейсом `EmailSender`; API request не ждёт отправку письма.

Временные ошибки используют exponential backoff: 5 секунд, 30 секунд, 2 минуты, 10 минут и 30 минут. После пяти неудачных попыток событие получает терминальный статус `failed`.

`event_id` передаётся как `Message-ID` и `X-Event-ID`. Worker не забирает уже обработанные события намеренно, но SMTP не поддерживает exactly-once: при разрыве соединения после приёма письма возможна повторная доставка. Circuit breaker не добавлен, поскольку batch polling и backoff уже ограничивают обращения к недоступному provider; retry отвечает за повтор конкретного события.

Метрики worker доступны на `http://localhost:19091/metrics`, письма локального SMTP видны в Mailpit UI на `http://localhost:8025`.

## Graceful shutdown

Приложение обрабатывает сигналы:

```text
SIGINT
SIGTERM
```

При завершении:

* HTTP server останавливается через `server.Shutdown`;
* используется timeout `10s`;
* закрывается подключение к PostgreSQL;
* закрывается подключение к Redis;
* в лог пишется старт и остановка сервера.

## Примеры curl

Ниже приведён полный сценарий ручной проверки API.

### 1. Регистрация первого пользователя

```bash
curl -s -X POST http://localhost:18080/api/v1/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"user1@example.com","password":"pass123"}'
```

### 2. Регистрация второго пользователя

```bash
curl -s -X POST http://localhost:18080/api/v1/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"user2@example.com","password":"pass123"}'
```

### 3. Логин первого пользователя

```bash
curl -s -X POST http://localhost:18080/api/v1/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"user1@example.com","password":"pass123"}'
```

Скопируйте токен из ответа и сохраните в переменную:

```bash
TOKEN="..."
```

### 4. Создание команды

```bash
curl -s -X POST http://localhost:18080/api/v1/teams \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"name":"Backend Team"}'
```

### 5. Приглашение второго пользователя

Если у второго пользователя `user_id = 2`:

```bash
curl -s -X POST http://localhost:18080/api/v1/teams/1/invite \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"user_id":2,"role":"member"}'
```

### 6. Создание задачи

```bash
curl -s -X POST http://localhost:18080/api/v1/tasks \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"team_id":1,"title":"Implement API","description":"Finish endpoints","status":"todo","assignee_id":2}'
```

### 7. Получение списка задач

```bash
curl -s 'http://localhost:18080/api/v1/tasks?team_id=1&status=todo&limit=20&offset=0' \
  -H "Authorization: Bearer $TOKEN"
```

### 8. Обновление задачи

```bash
curl -s -X PUT http://localhost:18080/api/v1/tasks/1 \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"status":"done"}'
```

### 9. Переназначение задачи

```bash
curl -s -X PUT http://localhost:18080/api/v1/tasks/1 \
  -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"assignee_id":2}'
```

### 10. Получение истории задачи

```bash
curl -s http://localhost:18080/api/v1/tasks/1/history \
  -H "Authorization: Bearer $TOKEN"
```

### 11. Получение отчётов

```bash
curl -s http://localhost:18080/api/v1/reports/team-stats \
  -H "Authorization: Bearer $TOKEN"
```

```bash
curl -s http://localhost:18080/api/v1/reports/top-users \
  -H "Authorization: Bearer $TOKEN"
```

```bash
curl -s http://localhost:18080/api/v1/reports/invalid-assignees \
  -H "Authorization: Bearer $TOKEN"
```

### 12. Проверка метрик

```bash
curl -s http://localhost:18080/metrics
```

## Тестирование

В проекте реализованы:

* unit-тесты сервисов;
* unit-тесты middleware;
* unit-тесты handlers;
* unit-тесты config;
* Redis-тесты через `redismock`;
* repository-тесты через `sqlmock`;
* integration-тесты repository-слоя с PostgreSQL через `testcontainers`.

Запуск всех тестов:

```bash
(cd backend && go test ./...)
```

Запуск тестов с отчётом покрытия:

```bash
make test-cover
```

Команда `make test-cover` запускает тесты по основным пакетам приложения:

```text
./backend/internal/cache
./backend/internal/config
./backend/internal/handlers
./backend/internal/middleware
./backend/internal/repository
./backend/internal/service
```

После запуска выводится подробный отчёт:

```bash
(cd backend && go tool cover -func=coverage.out)
```

На последней локальной проверке тесты успешно проходят, а общее покрытие по основным пакетам составляет около:

```text
50.1%
```

Важно: в ТЗ указано требование минимум 85% покрытия по критическим методам. Текущий проект содержит тесты и рабочий отчёт покрытия, но фактическое покрытие пока ниже 85%. Для полного формального соответствия этому пункту нужно дополнительно покрыть тестами:

* `backend/internal/handlers/reports.go`;
* `backend/internal/repository/reports.go`;
* `backend/internal/handlers/tasks.go`;
* `backend/internal/handlers/teams.go`;
* `backend/internal/middleware/metrics.go`;
* часть методов `backend/internal/repository`.

Integration-тесты используют PostgreSQL 16 через `testcontainers`, поэтому для их запуска должен быть включён Docker Desktop.

Отдельный запуск integration-тестов:

```bash
make test-integration
```

Полный локальный цикл проверки:

```bash
(cd backend && go mod tidy)
(cd backend && go test ./...)
make test-cover
docker compose up --build
```

## Проверка качества кода

Форматирование:

```bash
gofmt -w backend
```

Проверка зависимостей:

```bash
(cd backend && go mod tidy)
```

Статический анализ:

```bash
(cd backend && go vet ./...)
```

Проверка Docker Compose:

```bash
docker compose config
```

Полный цикл:

```bash
gofmt -w backend
(cd backend && go mod tidy)
(cd backend && go test ./...)
make test-cover
(cd backend && go vet ./...)
docker compose config
docker compose up --build
```

## Makefile

Доступные команды:

```bash
make test
```

Запускает:

```bash
(cd backend && go test ./...)
```

```bash
make test-cover
```

Запускает тесты по основным пакетам и выводит coverage report.

```bash
make test-integration
```

Запускает integration-тесты repository-слоя.

```bash
make run
```

Запускает проект через Docker Compose.

```bash
make down
```

Останавливает Docker Compose.

## Возможные проблемы и решения

### Docker не запущен

Если Docker не запущен, могут появиться ошибки:

```text
Cannot connect to the Docker daemon
```

или:

```text
testcontainers cannot start PostgreSQL container
```

Решение:

```bash
open -a Docker
```

Потом проверить:

```bash
docker ps
```

### Порт 18080 занят

Выберите другой host-порт:

```bash
APP_HOST_PORT=18081 docker compose up --build
```

### Нужно запустить API на 8080

```bash
APP_HOST_PORT=8080 docker compose up --build
```

### Порт PostgreSQL 5432 занят

Обычному запуску это не мешает, потому что PostgreSQL из Docker Compose публикуется на host-порт `5433`.

Если нужен другой debug-порт PostgreSQL:

```bash
POSTGRES_HOST_PORT=15432 docker compose up --build
```

### Порт Redis 6379 занят

Обычному запуску это не мешает, потому что Redis из Docker Compose публикуется на host-порт `6380`.

Если нужен другой debug-порт Redis:

```bash
REDIS_HOST_PORT=16380 docker compose up --build
```

### Нужно пересоздать базу

Если нужно заново применить миграции:

```bash
docker compose down -v
docker compose up --build
```

### Ошибка авторизации

Для защищённых endpoint нужен JWT:

```text
Authorization: Bearer <token>
```

Пример:

```bash
curl -s http://localhost:18080/api/v1/teams \
  -H "Authorization: Bearer $TOKEN"
```

Если заголовок отсутствует или токен некорректный, API вернёт:

```http
401 Unauthorized
```

## Соответствие ТЗ

| Требование                 | Статус                 | Комментарий                                                                     |
| -------------------------- | ---------------------- | ------------------------------------------------------------------------------- |
| Go                         | Выполнено              | Backend написан на Go                                                           |
| PostgreSQL                 | Выполнено              | Используется PostgreSQL 16                                                      |
| Redis                      | Выполнено              | Используется для кеша и rate limiting                                           |
| Docker                     | Выполнено              | Backend и frontend имеют отдельные Dockerfile                                  |
| Docker Compose             | Выполнено              | Поднимает приложение, хранилища, tracing, Prometheus, Grafana и Mailpit         |
| Git                        | Выполнено              | Проект готов для хранения в Git                                                 |
| Регистрация                | Выполнено              | `POST /api/v1/register`                                                         |
| Аутентификация             | Выполнено              | `POST /api/v1/login`, JWT                                                       |
| Команды                    | Выполнено              | Создание, список, invite                                                        |
| Роли                       | Выполнено              | `owner/admin/member`                                                            |
| Задачи                     | Выполнено              | Создание, список, обновление                                                    |
| История изменений          | Выполнено              | `GET /api/v1/tasks/{id}/history`                                                |
| Комментарии                | Выполнено              | `POST/GET /api/v1/tasks/{id}/comments`                                          |
| JOIN 3+ таблиц + агрегация | Выполнено              | `GET /api/v1/reports/team-stats`                                                |
| Оконная функция            | Выполнено              | `GET /api/v1/reports/top-users`                                                 |
| Проверка связанных таблиц  | Выполнено              | `GET /api/v1/reports/invalid-assignees`                                         |
| Redis cache TTL 5 минут    | Выполнено              | Кеш списка задач                                                                |
| Индексы PostgreSQL         | Выполнено              | Миграции `002_indexes.sql`-`005_outbox_worker_index.sql`                        |
| Connection pooling         | Выполнено              | Настроено в `backend/internal/db/postgres.go`                                   |
| Пагинация на уровне БД     | Выполнено              | `LIMIT/OFFSET`                                                                  |
| Unit-тесты                 | Выполнено              | Есть тесты сервисов, handlers, middleware, repository                           |
| Integration-тесты с PostgreSQL | Выполнено          | Используется `testcontainers`                                                   |
| 85% покрытия               | Не выполнено полностью | Текущее покрытие около `50.1%`; требуется добавить тесты                        |
| Transactional outbox       | Выполнено              | Бизнес-изменения и события сохраняются атомарно                                 |
| Email worker               | Выполнено              | Асинхронная SMTP-доставка, retry, max attempts и graceful shutdown              |
| Rate limiting              | Выполнено              | 100 запросов/мин; IP для register/login, user_id для защищённых endpoint        |
| Graceful shutdown          | Выполнено              | `SIGINT/SIGTERM`, timeout 10 секунд                                             |
| Prometheus metrics         | Выполнено              | Prometheus собирает API и worker metrics                                        |
| Grafana dashboard          | Выполнено              | Dashboard provisioned автоматически и доступен на порту `3001`                  |
| Config YAML/ENV            | Выполнено              | `backend/config.yaml` + ENV override                                            |

## Финальный результат

Проект запускается одной командой:

```bash
docker compose up --build
```

После успешного запуска API доступно на:

```text
http://localhost:18080
```

База данных PostgreSQL и Redis поднимаются автоматически через Docker Compose.

Миграции из директории `backend/migrations/` применяются при первом старте PostgreSQL-контейнера.

Основные сценарии проверки:

* регистрация пользователя;
* логин;
* создание команды;
* приглашение участника;
* создание задачи;
* обновление задачи;
* переназначение задачи;
* получение истории задачи;
* получение списка задач с фильтрацией и пагинацией;
* получение SQL-отчётов;
* проверка Prometheus-метрик.

Проект функционально закрывает основные требования ТЗ. Единственный заметный пункт, который требует доработки для полного формального соответствия, — покрытие тестами до 85%.

# mkk

**Описание**
Сервис управления командами и задачами с REST API, Swagger UI, метриками Prometheus и мониторингом через VictoriaMetrics + Grafana.

**Быстрый старт**
1. Создать `.env` (на базе примера):

```bash
cp .env.example .env
```

2. Запустить инфраструктуру:

```bash
make up
```

**Сервисы и доступ**
- Приложение: `http://localhost:8080`
- Swagger UI: `http://localhost:8081`
- Grafana: `http://localhost:3000`
- VictoriaMetrics: `http://localhost:8428`
- MySQL: `localhost:3306`
- Redis: `localhost:6379`

**Маршруты API**
- `POST /api/v1/register` — регистрация
- `POST /api/v1/login` — логин
- `GET /api/v1/teams` — список команд
- `POST /api/v1/teams` — создать команду
- `POST /api/v1/teams/{id}/invite` — пригласить пользователя
- `POST /api/v1/teams/invites/accept` — принять приглашение
- `GET /api/v1/tasks` — список задач
- `POST /api/v1/tasks` — создать задачу
- `PUT /api/v1/tasks/{id}` — обновить задачу
- `GET /api/v1/tasks/{id}/history` — история изменений
- `GET /api/v1/tasks/{id}/comments` — список комментариев
- `POST /api/v1/tasks/{id}/comments` — создать комментарий
- `PUT /api/v1/tasks/{id}/comments/{comment_id}` — обновить комментарий
- `DELETE /api/v1/tasks/{id}/comments/{comment_id}` — удалить комментарий
- `GET /api/v1/reports/team-summary` — сводка по командам
- `GET /api/v1/reports/top-creators` — топ создателей задач
- `GET /api/v1/reports/invalid-assignees` — задачи с неверными исполнителями

**Swagger и OpenAPI**
- Swagger UI: `http://localhost:8081`
- OpenAPI JSON: `http://localhost:8080/openapi.json`

**Метрики**
- Endpoint: `GET /metrics`
- Метрики:
- `mkk_http_requests_total` — количество HTTP запросов по меткам `method`, `path`, `status`
- `mkk_http_request_duration_seconds` — время обработки HTTP запросов по меткам `method`, `path`, `status`

**Grafana**
- Логин: `admin`
- Пароль: `admin`
- Дашборды сохраняются в volume `grafana_data`.

**Дефолтный пользователь**
- Email: `admin@example.com`
- Пароль: `admin123`

**Модели (объекты)**
- В обработчиках и сервисах используются типы, сгенерированные из OpenAPI спецификации (`internal/api`).

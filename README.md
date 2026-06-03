# EventHub

Backend API для мероприятий. Проект использует Redis для сессий и кэшей, MongoDB для пользователей и мероприятий, Cassandra для реакций и отзывов, Neo4j для графа рекомендаций.

## Запуск

1. Проверьте `.env.local`.
2. Запустите сервисы:

```sh
make run
```

API будет доступно на `http://localhost:8080`, если `APP_PORT=8080`.

Полезные команды:

```sh
make logs     # логи всех контейнеров
make stop     # остановить контейнеры
make clean    # остановить и удалить volumes
make test     # go test ./...
```

## API

Swagger/OpenAPI лежит в [api/openapi.yaml](api/openapi.yaml).

Основные ручки:

- `POST /users` - регистрация и выдача cookie `X-Session-Id`
- `POST /auth/login`, `POST /auth/logout` - вход и выход
- `GET /users`, `GET /users/{id}`, `GET /users/{id}/events` - пользователи
- `POST /events`, `GET /events`, `GET /events/{id}`, `PATCH /events/{id}` - мероприятия
- `POST /events/{id}/like`, `POST /events/{id}/dislike` - реакции
- `POST /events/{id}/reviews`, `GET /events/{id}/reviews`, `PATCH /events/{id}/reviews/{review_id}` - отзывы
- `GET /recommendations` - рекомендации для текущего авторизованного пользователя

`include=reactions,reviews` можно передавать в запросы списка/получения мероприятий.

## Хранилища

- MongoDB: документы пользователей и мероприятий.
- Redis: сессии, кэш реакций, отзывов и рекомендаций.
- Cassandra: таблицы `event_reactions` и `event_reviews`, создаются через `scripts/cassandra-init.sh`.
- Neo4j: узлы `User`, `Event` и связь `LIKED`, constraints создаются через `scripts/neo4j-init.sh`.

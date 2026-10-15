# Семейные расходы

Сервис для учёта семейных расходов. Траты можно добавлять через Telegram обычным текстом. Веб-панель показывает общий список и позволяет руководителю семьи добавлять, изменять и удалять записи.

Бэкенд написан на Go. Данные хранятся в PostgreSQL. Проект запускается одной командой через Docker Compose.

## Запуск

Нужны Docker и Docker Compose.

```bash
cp .env.example .env
docker compose up --build
```

После запуска веб-панель доступна по адресу http://localhost:8080.

Для остановки:

```bash
docker compose down
```

## Telegram и LLM

В `.env` нужно указать токен бота:

```env
TELEGRAM_BOT_TOKEN=токен_бота
```

Для подключения LLM также указываются:

```env
LLM_BASE_URL=https://api.openai.com/v1
LLM_API_KEY=ключ
LLM_MODEL=gpt-4o-mini
```

Используется OpenAI-совместимый API, поэтому провайдера и модель можно заменить через переменные окружения. Если ключ LLM не задан, бот понимает основные команды встроенным разбором текста.

Примеры сообщений:

```text
кофе 350
такси 1200 вчера
сколько потратил на еду в этом месяце
покажи траты за неделю
последнее было не 350, а 450
удали такси
```

Telegram ID используется как идентификатор пользователя. В командах бота все операции ограничены этим ID, поэтому пользователь не видит и не изменяет чужие расходы.

## REST API

Основные методы:

```text
GET    /api/expenses
POST   /api/expenses
GET    /api/expenses/{id}
PUT    /api/expenses/{id}
DELETE /api/expenses/{id}
GET    /health
```

Для списка доступны фильтры `user_id`, `category`, `from`, `to` и `limit`.

Пример добавления:

```bash
curl -X POST http://localhost:8080/api/expenses \
  -H 'Content-Type: application/json' \
  -d '{"user_id":123,"amount":"350.00","category":"food","spent_at":"2026-10-15","comment":"кофе"}'
```

Категории: `food`, `transport`, `housing`, `entertainment`, `health`, `shopping`, `other`.

## Проверка

```bash
go test ./...
go test -race ./...
go vet ./...
```

В проекте оставлен один небольшой набор тестов на разбор основных команд из задания. Остальная проверка выполняется через REST API после запуска Docker Compose.

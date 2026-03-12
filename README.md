# Berkut Solutions - Sonarium

<p align="center">
  <img src="gui/static/logo.png" alt="Sonarium logo" width="220">
</p>

[English version](README.en.md)

Sonarium — self-hosted музыкальная платформа для локальной библиотеки: стриминг, умный плеер, коллекции, совместная работа и современный web-интерфейс без внешних SaaS-зависимостей.

Актуальная версия: `1.0.0`

## Что это за продукт

Sonarium — единая среда для личной или командной музыкальной библиотеки, где локальные треки, альбомы, исполнители, плейлисты и доступы управляются в одном интерфейсе.

Проект рассчитан на self-hosted сценарий: вы храните музыку, базу данных и все пользовательские данные в своём контуре, а приложение даёт полноценный UI для прослушивания, каталогизации и совместного доступа.

## Для кого подходит

- Тем, кто хочет держать музыкальную библиотеку локально и не зависеть от внешних стриминговых сервисов.
- Командам, которым нужен общий доступ к плейлистам и коллекциям с разграничением прав.
- Пользователям с большой библиотекой, где важны жанры, поиск дублей, редактирование метаданных и удобная навигация.
- Тем, кому нужен современный web UI с deep-link страницами, а не набор модальных окон.

## Что даёт Sonarium

- Централизует локальную музыкальную библиотеку в одном интерфейсе.
- Даёт удобный плеер, очереди, избранное, жанры и навигацию по каталогу.
- Позволяет делиться сущностями и плейлистами между пользователями.
- Снижает зависимость от внешних музыкальных платформ и облачных сервисов.

## Основные возможности

- Каталог и навигация:
  - отдельные страницы для альбомов, исполнителей, треков, плейлистов и жанров
  - поиск по библиотеке
  - фильтрация и поиск дублей по названию треков
- Встроенный плеер:
  - очередь воспроизведения
  - drag-and-drop перестановка треков в очереди
  - shuffle / repeat
  - визуальный progress UI и waveform
- Работа с библиотекой:
  - сканирование музыкальной директории
  - загрузка отдельных файлов и целых папок
  - чтение metadata, cover art и жанров из тегов
  - редактирование альбомов, треков, исполнителей и плейлистов
  - объединение дублей исполнителей и альбомов
- Совместная работа:
  - публичные ссылки на сущности
  - доступ к плейлистам для listener / editor
  - просмотр профилей пользователей
  - shared with me / совместные коллекции
- Администрирование:
  - первый пользователь становится admin
  - управление пользователями
  - включение и отключение регистрации
  - проверка обновлений с GitHub в Settings
- Совместимость:
  - REST API
  - Subsonic adapter (`/rest`)

## Безопасность и доступ

- Zero-trust авторизация с сессиями.
- Блокировка доступа к приложению без входа.
- Разделение прав пользователей и admin-функций.
- Доступ к профилю и shared-сущностям управляется на серверной стороне.

## Технический профиль

- Backend: Go
- Database: PostgreSQL
- Deployment: Docker / Docker Compose
- Media stack: FFmpeg
- Migrations: Goose

## Скриншоты

![Screenshot 1](gui/static/screen1.png)

![Screenshot 2](gui/static/screen2.png)

![Screenshot 3](gui/static/screen3.png)

## Быстрый запуск

1. Скопируйте пример env:

```bash
cp .env.example .env
```

2. Запустите стек:

```bash
docker compose up -d --build
```

3. Откройте приложение:

```text
http://localhost:8080
```

4. Создайте первого пользователя. Он автоматически станет администратором.

## Docker volumes

Проект использует отдельные Docker named volumes:

- `postgres_data` — PostgreSQL
- `soundhub_data` — сервисные данные, кэш, thumbnails
- `soundhub_music` — музыкальная библиотека

Проверка:

```bash
docker volume ls
```

Полное удаление стека вместе с данными:

```bash
docker compose down -v
```

## Документация

- Общий индекс: [docs/README.md](docs/README.md)
- Русская документация: [docs/ru/README.md](docs/ru/README.md)
- English docs: [docs/eng/README.md](docs/eng/README.md)
- Архитектура: [docs/architecture.md](docs/architecture.md)
- API: [docs/api.md](docs/api.md)
- Docker strategy: [docs/docker_strategy.md](docs/docker_strategy.md)
- Структура репозитория: [docs/repository_structure.md](docs/repository_structure.md)
- Публикация на GitHub: [docs/ru/github_publish.md](docs/ru/github_publish.md)

## Публикация образа в GHCR

```bash
docker build -t sonarium:1.0.0 -f Dockerfile .

docker tag sonarium:1.0.0 ghcr.io/<your-github-username>/sonarium:1.0.0
docker tag sonarium:1.0.0 ghcr.io/<your-github-username>/sonarium:latest
docker push ghcr.io/<your-github-username>/sonarium:1.0.0
docker push ghcr.io/<your-github-username>/sonarium:latest
```

## Публикация на GitHub

Если репозиторий уже создан на GitHub:

```bash
git add .
git commit -m "Initial Sonarium release"
git branch -M main
git remote add origin https://github.com/<your-username>/sonarium.git
git push -u origin main
```

## Лицензия

[LICENSE](LICENSE)

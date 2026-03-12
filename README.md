# Berkut Solutions - Sonarium

<p align="center">
  <img src="gui/static/logo.png" alt="Sonarium logo" width="220">
</p>

[English version](README.en.md)

Sonarium — self-hosted музыкальная платформа для локальной библиотеки: стриминг, умный плеер, коллекции, совместная работа и веб-интерфейс без внешних SaaS-зависимостей.

## Что это за проект

Sonarium разворачивается в Docker Compose, хранит библиотеку и данные в отдельных Docker volumes, индексирует локальную музыку, строит обложки и даёт полноценный web UI с deep-link страницами для альбомов, исполнителей, треков и плейлистов.

Проект ориентирован на:
- локальное self-hosted использование
- zero-trust авторизацию с сессиями
- совместную работу с плейлистами и шарингом
- большую музыкальную библиотеку

## Основные возможности

- Веб-приложение с отдельными страницами для:
  - альбомов
  - исполнителей
  - треков
  - плейлистов
  - жанров
  - профилей пользователей
- Встроенный плеер:
  - очередь
  - drag-and-drop перестановка треков
  - shuffle / repeat
  - waveform / progress UI
- Локальная библиотека:
  - сканирование директории
  - загрузка отдельных файлов и папок
  - извлечение metadata и cover art
  - жанры, избранное, поиск дублей
- Совместная работа:
  - share по ссылке
  - доступ к плейлистам для listener / editor
  - просмотр профилей пользователей
- Администрирование:
  - первый пользователь становится admin
  - управление пользователями
  - включение/отключение регистрации
- Совместимость:
  - REST API
  - Subsonic adapter (`/rest`)

## Скриншоты

После добавления своих изображений положи их в `docs/assets/screenshots/` с такими именами:

- `docs/assets/screenshots/home.png`
- `docs/assets/screenshots/library.png`
- `docs/assets/screenshots/player.png`
- `docs/assets/screenshots/login.png`

README уже готов их показывать:

![Home](docs/assets/screenshots/home.png)
![Library](docs/assets/screenshots/library.png)
![Player](docs/assets/screenshots/player.png)
![Login](docs/assets/screenshots/login.png)

## Быстрый запуск

1. Скопируй env:

```bash
cp .env.example .env
```

2. Подними стек:

```bash
docker compose up -d --build
```

3. Открой приложение:

```text
http://localhost:8080
```

4. Создай первого пользователя. Он станет администратором.

## Docker volumes

Проект хранит данные в named volumes:

- `postgres_data` — PostgreSQL
- `soundhub_data` — app data, thumbnails, service data
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

- Индекс документации: [docs/README.md](docs/README.md)
- Русская документация: [docs/ru/README.md](docs/ru/README.md)
- English docs: [docs/eng/README.md](docs/eng/README.md)

Ключевые разделы:
- Архитектура: [docs/architecture.md](docs/architecture.md)
- API: [docs/api.md](docs/api.md)
- Docker strategy: [docs/docker_strategy.md](docs/docker_strategy.md)
- Структура репозитория: [docs/repository_structure.md](docs/repository_structure.md)
- Публикация на GitHub: [docs/ru/github_publish.md](docs/ru/github_publish.md)

## Публикация на GitHub

Если у тебя установлен GitHub CLI:

```bash
git init
git add .
git commit -m "Initial Sonarium release"
git branch -M main
gh auth login
gh repo create sonarium --public --source . --remote origin --push
```

Если репозиторий создашь вручную на GitHub:

```bash
git init
git add .
git commit -m "Initial Sonarium release"
git branch -M main
git remote add origin https://github.com/<your-username>/sonarium.git
git push -u origin main
```

## Технологии

- Go
- PostgreSQL
- Docker / Docker Compose
- Vanilla JS UI
- FFmpeg
- Goose migrations

## Лицензия

[LICENSE](LICENSE)

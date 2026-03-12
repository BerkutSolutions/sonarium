# Публикация Sonarium на GitHub

## Вариант 1: через GitHub CLI

```bash
git init
git add .
git commit -m "Initial Sonarium release"
git branch -M main
gh auth login
gh repo create sonarium --public --source . --remote origin --push
```

## Вариант 2: если репозиторий создаётся вручную

1. Создай пустой репозиторий на GitHub.
2. Выполни:

```bash
git init
git add .
git commit -m "Initial Sonarium release"
git branch -M main
git remote add origin https://github.com/<your-username>/sonarium.git
git push -u origin main
```

## Рекомендуемый pre-publish чеклист

- обновить `README.md`
- добавить скриншоты в `docs/assets/screenshots`
- проверить `.env.example`
- убедиться, что `docker compose up -d --build` проходит
- не коммитить реальные секреты и рабочий `.env`

## Полезные команды после публикации

```bash
git status
git add .
git commit -m "docs: update project documentation"
git push
```

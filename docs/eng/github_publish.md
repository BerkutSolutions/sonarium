# Publishing Sonarium to GitHub

## Option 1: GitHub CLI

```bash
git init
git add .
git commit -m "Initial Sonarium release"
git branch -M main
gh auth login
gh repo create sonarium --public --source . --remote origin --push
```

## Option 2: create the repository manually

1. Create an empty repository on GitHub.
2. Run:

```bash
git init
git add .
git commit -m "Initial Sonarium release"
git branch -M main
git remote add origin https://github.com/<your-username>/sonarium.git
git push -u origin main
```

## Recommended pre-publish checklist

- update `README.en.md`
- place screenshots into `docs/assets/screenshots`
- verify `.env.example`
- confirm `docker compose up -d --build` succeeds
- make sure no real secrets or working `.env` are committed

## Useful follow-up commands

```bash
git status
git add .
git commit -m "docs: update project documentation"
git push
```

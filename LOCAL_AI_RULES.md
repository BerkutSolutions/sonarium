# LOCAL_AI_RULES.md

Purpose: strict local engineering rules for AI assistants contributing to this repository.

Scope:
- This file is local repository governance.
- It must be read before any modification.
- It must not be packaged into runtime container images.

## 1. Project Discipline

- Read `docs/architecture.md`, `docs/modules.md`, and this file before changing code.
- Do not add ad-hoc patterns that bypass defined module boundaries.
- Keep changes focused and minimal to the requested scope.
- Update relevant docs in the same change when architecture or behavior changes.

## 2. Architecture Rules (Mandatory)

- Architecture style is modular monolith, not microservices.
- Every domain change must belong to an existing bounded context or explicitly introduce a new documented module.
- Transport handlers must not contain business rules.
- Domain services must not depend on HTTP or transport details.
- Repository interfaces live in module domain; implementations live in platform/infrastructure.
- No cyclic dependencies between modules.
- Cross-module access must go through interfaces/services, never direct storage access.

## 3. Module Contract Rules

Each module must preserve:
- `models`
- `service`
- `repository` interface(s)
- transport adapters when exposed externally

Do not merge unrelated module responsibilities into shared utility packages.

## 4. Subsonic Compatibility Rules

- Subsonic layer is adapter-only.
- Do not let Subsonic DTOs define internal domain models.
- Map Subsonic requests to internal services.
- Keep compatibility logic isolated in `subsonic` module.

## 5. Docker and Deployment Rules

- The application must remain fully runnable in Docker.
- Any feature requiring filesystem, DB, or cache must work with mounted volumes.
- Keep `docker-compose.yml`, `deploy/*`, and `.env.example` aligned with runtime requirements.
- Do not introduce runtime dependencies on external CDNs.
- Container startup must fail fast on invalid configuration.

## 6. Database and Migration Rules

- SQLite is default; PostgreSQL optional.
- Persistence access must go through repository interfaces.
- Schema changes require versioned migrations.
- Migration scripts must be deterministic and reviewed.
- Avoid breaking migrations; prefer forward-compatible evolution.

## 7. Release Discipline

- Backward compatibility for public APIs should be preserved when possible.
- Breaking changes require documentation updates and migration notes.
- Feature flags should be used for incomplete major features.
- Health endpoints must remain stable for orchestrators.

## 8. Code Modularity and Size

- Prefer files around 300-400 lines when practical.
- Split large files by responsibility (handler/service/repository/query).
- Avoid god-objects and cross-cutting utility dumping.
- Keep functions cohesive and testable.

## 9. Compatibility Policy

- Maintain compatibility with Docker Compose and Portainer stack deployment.
- Preserve Subsonic compatibility behavior once implemented; if changed, document explicitly.
- Keep API contracts versioned/documented when external behavior changes.

## 10. Localization Policy

- Source code, identifiers, docs, and logs should use English by default.
- User-facing localization support can be added later through structured i18n mechanisms.
- Do not hardcode mixed-language strings in domain logic.

## 11. Encoding and Text Rules

- All text files must use UTF-8 encoding.
- Avoid BOM unless tooling explicitly requires it.
- Normalize line endings consistently per repository settings.

## 12. Testing and Verification Expectations

- New behavior should include tests in the appropriate module layer.
- Avoid brittle tests coupled to implementation internals.
- Validate container startup path for significant runtime/config changes.

## 13. Documentation Update Requirements

Update docs when any of the following changes:
- module responsibilities
- repository structure
- environment variables
- Docker/Compose behavior
- API compatibility commitments

Minimum docs to check/update:
- `docs/architecture.md`
- `docs/modules.md`
- `docs/repository_structure.md`
- `docs/docker_strategy.md`

## 14. Forbidden Practices

- Introducing business logic directly in HTTP handlers
- Bypassing repositories with inline DB access from services
- Creating cyclic module dependencies
- Shipping local governance files in container runtime image
- Adding external runtime CDN dependency for required UI functionality

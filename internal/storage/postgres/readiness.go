package postgres

import (
	"context"
	"database/sql"
)

type ReadinessDependency struct {
	DB *sql.DB
}

func (d ReadinessDependency) Name() string {
	return "postgres"
}

func (d ReadinessDependency) Check(ctx context.Context) error {
	return d.DB.PingContext(ctx)
}

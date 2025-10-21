package postgres

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" //nolint: revive,nolintlint
)

type Logger interface {
	Info(msg string, args ...any)
}

type Provider struct {
	db        *sql.DB
	cs        string
	idlConns  int
	openConns int
	lifetime  time.Duration
	logger    Logger
}

func New(user, pass string, sqlDataBase SQLDataBase, logger Logger) *Provider {
	info := fmt.Sprintf(
		"user=%s password=%s host=%s port=%s dbname=%s sslmode=disable",
		user,
		pass,
		sqlDataBase.Server,
		sqlDataBase.Port,
		sqlDataBase.Database,
	)

	logger.Info("postgres connection", "user", user, "host", sqlDataBase.Server, "port", sqlDataBase.Port, "db", sqlDataBase.Database)

	return &Provider{
		cs:        info,
		idlConns:  sqlDataBase.MaxIdleCons,
		openConns: sqlDataBase.MaxOpenCons,
		lifetime:  time.Duration(sqlDataBase.ConnMaxLifetime) * time.Minute,
		logger:    logger,
	}
}

func (p *Provider) Open() error {
	var err error

	p.db, err = sql.Open("pgx", p.cs)
	if err != nil {
		return fmt.Errorf("can't open db conn: %w", err)
	}

	p.db.SetMaxIdleConns(p.idlConns)

	p.db.SetMaxOpenConns(p.openConns)

	p.db.SetConnMaxLifetime(p.lifetime)

	err = p.db.Ping()
	if err != nil {
		return fmt.Errorf("can't ping db: %w", err)
	}

	p.logger.Info("pg connection open")

	return nil
}

func (p *Provider) GetConn() *sql.DB {
	return p.db
}

func (p *Provider) Close() error {
	if p.db != nil {
		return p.db.Close()
	}
	return nil
}

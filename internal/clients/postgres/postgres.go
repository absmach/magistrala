package postgres

import (
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib" // required for SQL access
	"github.com/jmoiron/sqlx"
	"github.com/mainflux/mainflux/internal/env"
	"github.com/mainflux/mainflux/pkg/errors"
	migrate "github.com/rubenv/sql-migrate"
)

var (
	errConfig    = errors.New("failed to load postgresql configuration")
	errConnect   = errors.New("failed to connect to postgresql server")
	errMigration = errors.New("failed to apply migrations")
)

type Config struct {
	Host        string `env:"DB_HOST,notEmpty"           envDefault:"localhost"`
	Port        string `env:"DB_PORT,notEmpty"           envDefault:"5432"`
	User        string `env:"DB_USER,notEmpty"           envDefault:"mainflux"`
	Pass        string `env:"DB_PASS,notEmpty"           envDefault:"mainflux"`
	Name        string `env:"DB"                         envDefault:""`
	SSLMode     string `env:"DB_SSL_MODE,notEmpty"       envDefault:"disable"`
	SSLCert     string `env:"DB_SSL_CERT"                envDefault:""`
	SSLKey      string `env:"DB_SSL_KEY"                 envDefault:""`
	SSLRootCert string `env:"DB_SSL_ROOT_CERT"           envDefault:""`
}

// Setup creates a connection to the PostgreSQL instance and applies any
// unapplied database migrations. A non-nil error is returned to indicate failure.
func Setup(prefix string, migrations migrate.MemoryMigrationSource) (*sqlx.DB, error) {
	return SetupWithConfig(prefix, migrations, Config{})
}

// SetupWithConfig creates a connection to the PostgreSQL instance and applies any
// unapplied database migrations. A non-nil error is returned to indicate failure.
func SetupWithConfig(prefix string, migrations migrate.MemoryMigrationSource, defConfig Config) (*sqlx.DB, error) {
	cfg := defConfig
	if err := env.Parse(&cfg, env.Options{Prefix: prefix}); err != nil {
		return nil, errors.Wrap(errConfig, err)
	}
	return SetupDB(cfg, migrations)
}

// SetupDB creates a connection to the PostgreSQL instance and applies any
// unapplied database migrations. A non-nil error is returned to indicate failure.
func SetupDB(cfg Config, migrations migrate.MemoryMigrationSource) (*sqlx.DB, error) {
	db, err := Connect(cfg)
	if err != nil {
		return nil, err
	}
	if err := MigrateDB(db, migrations); err != nil {
		return nil, err
	}
	return db, nil
}

// Connect creates a connection to the PostgreSQL instance
func Connect(cfg Config) (*sqlx.DB, error) {
	url := fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=%s sslcert=%s sslkey=%s sslrootcert=%s", cfg.Host, cfg.Port, cfg.User, cfg.Name, cfg.Pass, cfg.SSLMode, cfg.SSLCert, cfg.SSLKey, cfg.SSLRootCert)

	db, err := sqlx.Open("pgx", url)
	if err != nil {
		return nil, errors.Wrap(errConnect, err)
	}

	return db, nil
}

// MigrateDB applies any unapplied database migrations
func MigrateDB(db *sqlx.DB, migrations migrate.MemoryMigrationSource) error {
	_, err := migrate.Exec(db.DB, "postgres", migrations, migrate.Up)
	if err != nil {
		return errors.Wrap(errMigration, err)
	}
	return nil
}

func (c *Config) LoadEnv(prefix string) error {
	if err := env.Parse(c, env.Options{Prefix: prefix}); err != nil {
		return errors.Wrap(errConfig, err)
	}
	return nil
}

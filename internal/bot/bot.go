// Package bot provides the Delta Chat bot setup and lifecycle management.
package bot

import (
	"database/sql"
	"io/fs"

	"github.com/chatmail/rpc-client-go/deltachat"
	"github.com/deltachat-bot/deltabot-cli-go/botcli"
	"github.com/spf13/cobra"

	"github.com/polpetta/patrizio/internal/adapter/sqlite"
	"github.com/polpetta/patrizio/internal/adapter/storage"
	"github.com/polpetta/patrizio/internal/config"
	"github.com/polpetta/patrizio/internal/database"
	"github.com/polpetta/patrizio/internal/domain"
	"github.com/spf13/afero"
)

// Setup creates and configures the BotCli instance with lifecycle hooks.
// The deps contains all injected dependencies (repository, storage, config).
// The migrationsFS and migrationsDir are used to run database migrations on start.
func Setup(deps *domain.Dependencies, migrationsFS fs.FS, migrationsDir string) *botcli.BotCli {
	cli := botcli.New("patrizio")

	cli.OnBotInit(func(cli *botcli.BotCli, bot *deltachat.Bot, _ *cobra.Command, _ []string) {
		bot.OnNewMsg(newMsgHandler(cli, bot, deps))
	})

	cli.OnBotStart(func(cli *botcli.BotCli, _ *deltachat.Bot, _ *cobra.Command, _ []string) {
		db, err := InitDatabase(deps.Config, migrationsFS, migrationsDir)
		if err != nil {
			cli.Logger.Panicf("Failed to initialize database: %v", err)
		}

		// Database is successfully initialized and available via deps.FilterRepository
		_ = db // Keep connection alive for the lifetime of the bot
		cli.Logger.Info("Database initialized successfully")
	})

	return cli
}

// InitDatabase opens the SQLite database and runs pending migrations.
// Exported for use in main.go to initialize DB before building dependencies.
func InitDatabase(cfg domain.Config, migrationsFS fs.FS, migrationsDir string) (*sql.DB, error) {
	dbPath := cfg.DBPath()

	db, err := database.Open(dbPath)
	if err != nil {
		return nil, err
	}

	if err := database.Migrate(db, migrationsFS, migrationsDir); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}

// BuildDependencies constructs the Dependencies with real adapter implementations.
func BuildDependencies(cfg *config.Config, db *sql.DB) *domain.Dependencies {
	return &domain.Dependencies{
		FilterRepository: sqlite.New(db),
		MediaStorage:     storage.New(afero.NewOsFs(), cfg.MediaPath()),
		Config:           cfg,
	}
}

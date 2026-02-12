// Package bot provides the Delta Chat bot setup and lifecycle management.
package bot

import (
	"database/sql"
	"io/fs"

	"github.com/chatmail/rpc-client-go/deltachat"
	"github.com/deltachat-bot/deltabot-cli-go/botcli"
	"github.com/spf13/cobra"

	"github.com/polpetta/patrizio/internal/config"
	"github.com/polpetta/patrizio/internal/database"
)

// Setup creates and configures the BotCli instance with lifecycle hooks.
// The migrationsFS and migrationsDir are used to run database migrations on start.
func Setup(migrationsFS fs.FS, migrationsDir string) *botcli.BotCli {
	cli := botcli.New("patrizio")

	cli.OnBotInit(func(cli *botcli.BotCli, bot *deltachat.Bot, _ *cobra.Command, _ []string) {
		bot.OnNewMsg(newMsgHandler(cli))
	})

	cli.OnBotStart(func(cli *botcli.BotCli, _ *deltachat.Bot, _ *cobra.Command, _ []string) {
		db, err := initDatabase(migrationsFS, migrationsDir)
		if err != nil {
			cli.Logger.Panicf("Failed to initialize database: %v", err)
		}

		// Store the database connection for later use.
		// For now we just ensure it's open and migrated; future changes
		// will wire it into the handler for query access.
		_ = db
	})

	return cli
}

// initDatabase opens the SQLite database and runs pending migrations.
func initDatabase(migrationsFS fs.FS, migrationsDir string) (*sql.DB, error) {
	dbPath := config.DBPath()

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

// Package bot provides the Delta Chat bot setup and lifecycle management.
package bot

import (
	"database/sql"
	"io/fs"

	"github.com/chatmail/rpc-client-go/deltachat"
	"github.com/deltachat-bot/deltabot-cli-go/botcli"
	"github.com/spf13/cobra"

	dcadapter "github.com/polpetta/patrizio/internal/adapter/deltachat"
	oai "github.com/polpetta/patrizio/internal/adapter/openai"
	"github.com/polpetta/patrizio/internal/adapter/sqlite"
	"github.com/polpetta/patrizio/internal/adapter/storage"
	"github.com/polpetta/patrizio/internal/config"
	"github.com/polpetta/patrizio/internal/database"
	"github.com/polpetta/patrizio/internal/domain"
	"github.com/spf13/afero"
)

// Setup creates and configures the BotCli instance with lifecycle hooks.
// The deps contains all injected dependencies (repository, storage, config).
func Setup(deps *domain.Dependencies) *botcli.BotCli {
	cli := botcli.New("patrizio")

	cli.OnBotInit(func(cli *botcli.BotCli, bot *deltachat.Bot, _ *cobra.Command, _ []string) {
		deps.Messenger = dcadapter.New(bot.Rpc)
		bot.OnNewMsg(func(_ *deltachat.Bot, accID deltachat.AccountId, msgID deltachat.MsgId) {
			go processMessage(cli.GetLogger(accID), uint64(accID), uint64(msgID), deps)
		})
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
		if closingErr := db.Close(); closingErr != nil {
			return nil, closingErr
		}
		return nil, err
	}

	return db, nil
}

// BuildDependencies constructs the Dependencies with real adapter implementations.
func BuildDependencies(cfg *config.Config, db *sql.DB) *domain.Dependencies {
	deps := &domain.Dependencies{
		FilterRepository:       sqlite.New(db),
		MediaStorage:           storage.New(afero.NewOsFs(), cfg.MediaPath()),
		Config:                 cfg,
		ConversationRepository: sqlite.NewConversationRepository(db),
	}

	// Only create the OpenAI client if an API key is configured.
	if cfg.OpenAIAPIKey() != "" {
		deps.AIClient = oai.New(cfg.OpenAIAPIKey(), cfg.OpenAIBaseURL(), cfg.OpenAIModel())
	}

	return deps
}

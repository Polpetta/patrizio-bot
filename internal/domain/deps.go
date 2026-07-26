package domain

// Dependencies holds all port implementations needed by the domain layer
type Dependencies struct {
	FilterRepository       FilterRepository
	MediaStorage           MediaStorage
	Config                 Config
	AIClient               AIClient
	ConversationRepository ConversationRepository
	Messenger              Messenger
	MemoryRepository       MemoryRepository
	ChatSettingsRepository ChatSettingsRepository
	ChatExecutor           ChatExecutor
}

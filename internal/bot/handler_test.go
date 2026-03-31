package bot

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/chatmail/rpc-client-go/deltachat"

	"github.com/polpetta/patrizio/internal/domain"
)

// --- Mock rpcClient ---

type mockRPC struct {
	// GetMessage
	getMessageFn func(deltachat.AccountId, deltachat.MsgId) (*deltachat.MsgSnapshot, error)
	// GetBasicChatInfo
	getBasicChatInfoFn func(deltachat.AccountId, deltachat.ChatId) (*deltachat.BasicChatSnapshot, error)
	// MiscSendTextMessage
	sentMessages []sentMessage
	sendErr      error
	// SendMsg
	sentMsgData []sentMsgDataEntry
	sendMsgErr  error
	// SendReaction
	sentReactions []sentReaction
	reactionErr   error
	// DownloadFullMessage
	downloadCalled bool
	downloadErr    error
}

type sentMessage struct {
	accID  deltachat.AccountId
	chatID deltachat.ChatId
	text   string
}

type sentReaction struct {
	accID    deltachat.AccountId
	msgID    deltachat.MsgId
	reaction []string
}

type sentMsgDataEntry struct {
	accID  deltachat.AccountId
	chatID deltachat.ChatId
	data   deltachat.MsgData
}

func (m *mockRPC) GetMessage(accID deltachat.AccountId, msgID deltachat.MsgId) (*deltachat.MsgSnapshot, error) {
	if m.getMessageFn != nil {
		return m.getMessageFn(accID, msgID)
	}
	return nil, fmt.Errorf("GetMessage not configured")
}

func (m *mockRPC) GetBasicChatInfo(accID deltachat.AccountId, chatID deltachat.ChatId) (*deltachat.BasicChatSnapshot, error) {
	if m.getBasicChatInfoFn != nil {
		return m.getBasicChatInfoFn(accID, chatID)
	}
	return nil, fmt.Errorf("GetBasicChatInfo not configured")
}

func (m *mockRPC) MiscSendTextMessage(accID deltachat.AccountId, chatID deltachat.ChatId, text string) (deltachat.MsgId, error) {
	m.sentMessages = append(m.sentMessages, sentMessage{accID: accID, chatID: chatID, text: text})
	return deltachat.MsgId(1), m.sendErr
}

func (m *mockRPC) SendMsg(accID deltachat.AccountId, chatID deltachat.ChatId, data deltachat.MsgData) (deltachat.MsgId, error) {
	m.sentMsgData = append(m.sentMsgData, sentMsgDataEntry{accID: accID, chatID: chatID, data: data})
	return deltachat.MsgId(1), m.sendMsgErr
}

func (m *mockRPC) SendReaction(accID deltachat.AccountId, msgID deltachat.MsgId, reaction ...string) (deltachat.MsgId, error) {
	m.sentReactions = append(m.sentReactions, sentReaction{accID: accID, msgID: msgID, reaction: reaction})
	return deltachat.MsgId(1), m.reactionErr
}

func (m *mockRPC) DownloadFullMessage(_ deltachat.AccountId, _ deltachat.MsgId) error {
	m.downloadCalled = true
	return m.downloadErr
}

// --- Mock FilterRepository ---

type mockFilterRepository struct {
	createTextFilterCalled     bool
	createMediaFilterCalled    bool
	createReactionFilterCalled bool

	lastChatID    int64
	lastTriggers  []string
	lastResponse  string
	lastMediaHash string
	lastMediaType string
	lastReaction  string

	createErr error

	// For FindMatchingFilters
	matchingFilters []domain.FilterResponse
	matchErr        error

	// For ListFilters
	listedFilters []domain.Filter
	listErr       error

	// For RemoveTrigger
	removedMediaHash *string
	removeErr        error

	// For RemoveAllFilters
	removedAllHashes []string
	removeAllErr     error
}

func (m *mockFilterRepository) CreateTextFilter(_ context.Context, chatID int64, triggers []string, responseText string) error {
	m.createTextFilterCalled = true
	m.lastChatID = chatID
	m.lastTriggers = triggers
	m.lastResponse = responseText
	return m.createErr
}

func (m *mockFilterRepository) CreateMediaFilter(_ context.Context, chatID int64, triggers []string, mediaHash string, mediaType string) error {
	m.createMediaFilterCalled = true
	m.lastChatID = chatID
	m.lastTriggers = triggers
	m.lastMediaHash = mediaHash
	m.lastMediaType = mediaType
	return m.createErr
}

func (m *mockFilterRepository) CreateReactionFilter(_ context.Context, chatID int64, triggers []string, reaction string) error {
	m.createReactionFilterCalled = true
	m.lastChatID = chatID
	m.lastTriggers = triggers
	m.lastReaction = reaction
	return m.createErr
}

func (m *mockFilterRepository) RemoveTrigger(_ context.Context, _ int64, _ string) (*string, error) {
	return m.removedMediaHash, m.removeErr
}

func (m *mockFilterRepository) RemoveAllFilters(_ context.Context, _ int64) ([]string, error) {
	return m.removedAllHashes, m.removeAllErr
}

func (m *mockFilterRepository) ListFilters(_ context.Context, _ int64) ([]domain.Filter, error) {
	return m.listedFilters, m.listErr
}

func (m *mockFilterRepository) FindMatchingFilters(_ context.Context, _ int64, _ string) ([]domain.FilterResponse, error) {
	return m.matchingFilters, m.matchErr
}

// --- Mock MediaStorage ---

type mockMediaStorage struct {
	saved    map[string][]byte
	saveErr  error
	basePath string // returned by Path; defaults to "/mock/media"
}

func newMockMediaStorage() *mockMediaStorage {
	return &mockMediaStorage{saved: make(map[string][]byte), basePath: "/mock/media"}
}

func (m *mockMediaStorage) Save(hash string, data []byte) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.saved[hash] = data
	return nil
}

func (m *mockMediaStorage) Delete(hash string) error {
	delete(m.saved, hash)
	return nil
}

func (m *mockMediaStorage) Read(hash string) ([]byte, error) {
	data, ok := m.saved[hash]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return data, nil
}

func (m *mockMediaStorage) Path(hash string) string {
	return filepath.Join(m.basePath, hash)
}

func (m *mockMediaStorage) Exists(hash string) (bool, error) {
	_, ok := m.saved[hash]
	return ok, nil
}

// --- Mock Logger ---

type mockLogger struct {
	infos  []string
	errors []string
	warns  []string
}

func (l *mockLogger) Infof(format string, args ...interface{}) {
	l.infos = append(l.infos, fmt.Sprintf(format, args...))
}

func (l *mockLogger) Errorf(format string, args ...interface{}) {
	l.errors = append(l.errors, fmt.Sprintf(format, args...))
}

func (l *mockLogger) Warnf(format string, args ...interface{}) {
	l.warns = append(l.warns, fmt.Sprintf(format, args...))
}

// --- Mock AIClient ---

type mockAIClient struct {
	response string
	err      error
	// Track calls
	lastMessages []domain.ChatMessage
	called       bool
}

func (m *mockAIClient) ChatCompletion(_ context.Context, messages []domain.ChatMessage) (string, error) {
	m.called = true
	m.lastMessages = messages
	return m.response, m.err
}

// --- Mock ConversationRepository ---

type mockConversationRepo struct {
	// SaveMessage tracking
	savedMessages []savedConversationMessage
	saveErr       error

	// IsConversationMessage
	isConvMsgExists     bool
	isConvMsgThreadRoot *int64
	isConvMsgErr        error

	// GetThreadChain
	threadChain    []domain.ChatMessage
	threadChainErr error
}

type savedConversationMessage struct {
	threadRootID int64
	msgID        int64
	parentMsgID  *int64
	role         string
	content      string
}

func (m *mockConversationRepo) SaveMessage(_ context.Context, threadRootID int64, msgID int64, parentMsgID *int64, role string, content string) error {
	m.savedMessages = append(m.savedMessages, savedConversationMessage{
		threadRootID: threadRootID,
		msgID:        msgID,
		parentMsgID:  parentMsgID,
		role:         role,
		content:      content,
	})
	return m.saveErr
}

func (m *mockConversationRepo) GetThreadChain(_ context.Context, _ int64, _ int) ([]domain.ChatMessage, error) {
	return m.threadChain, m.threadChainErr
}

func (m *mockConversationRepo) IsConversationMessage(_ context.Context, _ int64) (bool, *int64, error) {
	return m.isConvMsgExists, m.isConvMsgThreadRoot, m.isConvMsgErr
}

// --- Mock Config ---

type mockConfig struct {
	dbPath           string
	logLevel         string
	mediaPath        string
	openAIBaseURL    string
	openAIAPIKey     string
	openAIModel      string
	openAIMaxHistory int
	allowedChatIDs   []int64
	systemPrompt     string
}

func (m *mockConfig) DBPath() string                { return m.dbPath }
func (m *mockConfig) LogLevel() string              { return m.logLevel }
func (m *mockConfig) MediaPath() string             { return m.mediaPath }
func (m *mockConfig) OpenAIBaseURL() string         { return m.openAIBaseURL }
func (m *mockConfig) OpenAIAPIKey() string          { return m.openAIAPIKey }
func (m *mockConfig) OpenAIModel() string           { return m.openAIModel }
func (m *mockConfig) OpenAIMaxHistory() int         { return m.openAIMaxHistory }
func (m *mockConfig) OpenAIAllowedChatIDs() []int64 { return m.allowedChatIDs }
func (m *mockConfig) OpenAISystemPrompt() string    { return m.systemPrompt }

// --- Helper: write a temp file and return its path ---

func writeTempFile(t *testing.T, content []byte) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test-media.jpg")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	return path
}

// computeSHA512 returns the hex-encoded SHA-512 hash of data.
func computeSHA512(data []byte) string {
	h := sha512.Sum512(data)
	return hex.EncodeToString(h[:])
}

// --- Tests ---

func TestHandleFilterCommand_TextFilter(t *testing.T) {
	rpc := &mockRPC{}
	repo := &mockFilterRepository{}
	storage := newMockMediaStorage()
	logger := &mockLogger{}

	msg := &deltachat.MsgSnapshot{
		ChatId:   100,
		FromId:   42,
		Text:     "/filter hello Hi there!",
		ViewType: deltachat.MsgText,
	}

	deps := &domain.Dependencies{
		FilterRepository: repo,
		MediaStorage:     storage,
	}

	handleFilterCommand(rpc, logger, deltachat.AccountId(1), deltachat.MsgId(7), msg, deps)

	if !repo.createTextFilterCalled {
		t.Fatal("expected CreateTextFilter to be called")
	}
	if repo.lastResponse != "Hi there!" {
		t.Errorf("expected response 'Hi there!', got %q", repo.lastResponse)
	}
	if len(repo.lastTriggers) != 1 || repo.lastTriggers[0] != domain.NormalizeTrigger("hello") {
		t.Errorf("unexpected triggers: %v", repo.lastTriggers)
	}

	// Should have sent a confirmation as a quote-reply via SendMsg
	if len(rpc.sentMsgData) != 1 {
		t.Fatalf("expected 1 SendMsg call (confirmation), got %d", len(rpc.sentMsgData))
	}
	if rpc.sentMsgData[0].chatID != 100 {
		t.Errorf("expected confirmation sent to chat 100, got %d", rpc.sentMsgData[0].chatID)
	}
	if rpc.sentMsgData[0].data.QuotedMessageId != 7 {
		t.Errorf("expected QuotedMessageId 7, got %d", rpc.sentMsgData[0].data.QuotedMessageId)
	}
}

func TestHandleFilterCommand_ReactionFilter(t *testing.T) {
	rpc := &mockRPC{}
	repo := &mockFilterRepository{}
	storage := newMockMediaStorage()
	logger := &mockLogger{}

	msg := &deltachat.MsgSnapshot{
		ChatId:   100,
		FromId:   42,
		Text:     "/filter hello react:😂",
		ViewType: deltachat.MsgText,
	}

	deps := &domain.Dependencies{
		FilterRepository: repo,
		MediaStorage:     storage,
	}

	handleFilterCommand(rpc, logger, deltachat.AccountId(1), deltachat.MsgId(7), msg, deps)

	if !repo.createReactionFilterCalled {
		t.Fatal("expected CreateReactionFilter to be called")
	}
	if repo.lastReaction != "😂" {
		t.Errorf("expected reaction '😂', got %q", repo.lastReaction)
	}
}

// TestHandleFilterCommand_MediaFromAttachment is a regression test for Bug 2:
// /filter cat with an attached image (no text response) should create a media filter
// using the attached image, not error out.
func TestHandleFilterCommand_MediaFromAttachment(t *testing.T) {
	// Write a real temp file so processMediaFile can os.ReadFile it
	mediaContent := []byte("fake-image-data-for-test")
	mediaPath := writeTempFile(t, mediaContent)
	expectedHash := computeSHA512(mediaContent) + filepath.Ext(mediaPath)

	rpc := &mockRPC{}
	repo := &mockFilterRepository{}
	storage := newMockMediaStorage()
	logger := &mockLogger{}

	// Message has /filter cat with no text response, but carries an image attachment.
	msg := &deltachat.MsgSnapshot{
		ChatId:   100,
		FromId:   42,
		Text:     "/filter cat",
		ViewType: deltachat.MsgImage,
		File:     mediaPath,
	}

	deps := &domain.Dependencies{
		FilterRepository: repo,
		MediaStorage:     storage,
	}

	handleFilterCommand(rpc, logger, deltachat.AccountId(1), deltachat.MsgId(7), msg, deps)

	if len(logger.errors) > 0 {
		t.Fatalf("unexpected errors logged: %v", logger.errors)
	}

	if !repo.createMediaFilterCalled {
		t.Fatal("expected CreateMediaFilter to be called")
	}
	if repo.lastMediaHash != expectedHash {
		t.Errorf("expected media hash %q, got %q", expectedHash, repo.lastMediaHash)
	}
	if repo.lastMediaType != domain.MediaTypeImage {
		t.Errorf("expected media type %q, got %q", domain.MediaTypeImage, repo.lastMediaType)
	}
	if len(repo.lastTriggers) != 1 || repo.lastTriggers[0] != domain.NormalizeTrigger("cat") {
		t.Errorf("unexpected triggers: %v", repo.lastTriggers)
	}

	// Verify the media was saved to storage
	if _, exists := storage.saved[expectedHash]; !exists {
		t.Error("expected media to be saved to storage")
	}

	// Should have sent a confirmation as a quote-reply via SendMsg
	if len(rpc.sentMsgData) != 1 {
		t.Fatalf("expected 1 SendMsg call (confirmation), got %d", len(rpc.sentMsgData))
	}
	if rpc.sentMsgData[0].data.QuotedMessageId != 7 {
		t.Errorf("expected QuotedMessageId 7, got %d", rpc.sentMsgData[0].data.QuotedMessageId)
	}
}

// TestHandleFilterCommand_MediaFromAttachment_MultipleTriggers tests media filter with
// multiple triggers and an attached image.
func TestHandleFilterCommand_MediaFromAttachment_MultipleTriggers(t *testing.T) {
	mediaContent := []byte("another-fake-image")
	mediaPath := writeTempFile(t, mediaContent)
	expectedHash := computeSHA512(mediaContent) + filepath.Ext(mediaPath)

	rpc := &mockRPC{}
	repo := &mockFilterRepository{}
	storage := newMockMediaStorage()
	logger := &mockLogger{}

	msg := &deltachat.MsgSnapshot{
		ChatId:   200,
		FromId:   42,
		Text:     `/filter (cat, dog, "cute animals")`,
		ViewType: deltachat.MsgImage,
		File:     mediaPath,
	}

	deps := &domain.Dependencies{
		FilterRepository: repo,
		MediaStorage:     storage,
	}

	handleFilterCommand(rpc, logger, deltachat.AccountId(1), deltachat.MsgId(7), msg, deps)

	if len(logger.errors) > 0 {
		t.Fatalf("unexpected errors logged: %v", logger.errors)
	}
	if !repo.createMediaFilterCalled {
		t.Fatal("expected CreateMediaFilter to be called")
	}
	if len(repo.lastTriggers) != 3 {
		t.Fatalf("expected 3 triggers, got %d: %v", len(repo.lastTriggers), repo.lastTriggers)
	}
	if repo.lastMediaHash != expectedHash {
		t.Errorf("expected media hash %q, got %q", expectedHash, repo.lastMediaHash)
	}
}

func TestHandleFilterCommand_MediaFromQuotedMessage(t *testing.T) {
	mediaContent := []byte("quoted-media-content")
	mediaPath := writeTempFile(t, mediaContent)
	expectedHash := computeSHA512(mediaContent) + filepath.Ext(mediaPath)

	rpc := &mockRPC{}
	repo := &mockFilterRepository{}
	storage := newMockMediaStorage()
	logger := &mockLogger{}

	// The quoted message returns an image with DownloadDone state
	rpc.getMessageFn = func(_ deltachat.AccountId, msgID deltachat.MsgId) (*deltachat.MsgSnapshot, error) {
		if msgID == 999 {
			return &deltachat.MsgSnapshot{
				Id:            999,
				ViewType:      deltachat.MsgImage,
				File:          mediaPath,
				DownloadState: deltachat.DownloadDone,
			}, nil
		}
		return nil, fmt.Errorf("unexpected msgID %d", msgID)
	}

	// /filter cat with no text, replying to a media message
	msg := &deltachat.MsgSnapshot{
		ChatId:   100,
		FromId:   42,
		Text:     "/filter cat",
		ViewType: deltachat.MsgText,
		Quote: &deltachat.MsgQuote{
			MessageId: 999,
		},
	}

	deps := &domain.Dependencies{
		FilterRepository: repo,
		MediaStorage:     storage,
	}

	handleFilterCommand(rpc, logger, deltachat.AccountId(1), deltachat.MsgId(7), msg, deps)

	if len(logger.errors) > 0 {
		t.Fatalf("unexpected errors logged: %v", logger.errors)
	}
	if !repo.createMediaFilterCalled {
		t.Fatal("expected CreateMediaFilter to be called")
	}
	if repo.lastMediaHash != expectedHash {
		t.Errorf("expected media hash %q, got %q", expectedHash, repo.lastMediaHash)
	}
	if repo.lastMediaType != domain.MediaTypeImage {
		t.Errorf("expected media type %q, got %q", domain.MediaTypeImage, repo.lastMediaType)
	}
}

// TestHandleFilterCommand_MediaFromAttachment_PreferredOverQuote verifies that when both
// an attachment and a quoted message are present, the attachment takes priority.
func TestHandleFilterCommand_MediaFromAttachment_PreferredOverQuote(t *testing.T) {
	mediaContent := []byte("attached-media-wins")
	mediaPath := writeTempFile(t, mediaContent)
	expectedHash := computeSHA512(mediaContent) + filepath.Ext(mediaPath)

	rpc := &mockRPC{}
	repo := &mockFilterRepository{}
	storage := newMockMediaStorage()
	logger := &mockLogger{}

	// Quoted message would return a different image, but it should not be used
	rpc.getMessageFn = func(_ deltachat.AccountId, _ deltachat.MsgId) (*deltachat.MsgSnapshot, error) {
		t.Error("GetMessage should not be called when attachment is present")
		return nil, fmt.Errorf("should not be called")
	}

	msg := &deltachat.MsgSnapshot{
		ChatId:   100,
		FromId:   42,
		Text:     "/filter cat",
		ViewType: deltachat.MsgSticker,
		File:     mediaPath,
		Quote: &deltachat.MsgQuote{
			MessageId: 999,
		},
	}

	deps := &domain.Dependencies{
		FilterRepository: repo,
		MediaStorage:     storage,
	}

	handleFilterCommand(rpc, logger, deltachat.AccountId(1), deltachat.MsgId(7), msg, deps)

	if len(logger.errors) > 0 {
		t.Fatalf("unexpected errors logged: %v", logger.errors)
	}
	if !repo.createMediaFilterCalled {
		t.Fatal("expected CreateMediaFilter to be called")
	}
	if repo.lastMediaHash != expectedHash {
		t.Errorf("attachment hash should be used, got %q", repo.lastMediaHash)
	}
	if repo.lastMediaType != domain.MediaTypeSticker {
		t.Errorf("expected media type %q, got %q", domain.MediaTypeSticker, repo.lastMediaType)
	}
}

func TestHandleFilterCommand_MediaNoAttachmentNoQuote(t *testing.T) {
	rpc := &mockRPC{}
	repo := &mockFilterRepository{}
	storage := newMockMediaStorage()
	logger := &mockLogger{}

	// /filter cat with no text and no media anywhere — should produce an error
	msg := &deltachat.MsgSnapshot{
		ChatId:   100,
		FromId:   42,
		Text:     "/filter cat",
		ViewType: deltachat.MsgText,
	}

	deps := &domain.Dependencies{
		FilterRepository: repo,
		MediaStorage:     storage,
	}

	handleFilterCommand(rpc, logger, deltachat.AccountId(1), deltachat.MsgId(7), msg, deps)

	if repo.createMediaFilterCalled || repo.createTextFilterCalled {
		t.Fatal("no filter should have been created")
	}

	// Should have sent an error message as a quote-reply via SendMsg
	if len(rpc.sentMsgData) != 1 {
		t.Fatalf("expected 1 SendMsg call (error message), got %d", len(rpc.sentMsgData))
	}
	if rpc.sentMsgData[0].data.QuotedMessageId != 7 {
		t.Errorf("expected QuotedMessageId 7, got %d", rpc.sentMsgData[0].data.QuotedMessageId)
	}
}

func TestHandleDMMessage(t *testing.T) {
	rpc := &mockRPC{}
	logger := &mockLogger{}

	msg := &deltachat.MsgSnapshot{
		ChatId: 50,
		FromId: 42,
		Text:   "hey there",
	}

	deps := &domain.Dependencies{
		FilterRepository: &mockFilterRepository{},
		MediaStorage:     newMockMediaStorage(),
	}

	handleDMMessage(rpc, logger, deltachat.AccountId(1), deltachat.MsgId(1), msg, deps)

	if len(rpc.sentMessages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(rpc.sentMessages))
	}
	if rpc.sentMessages[0].text != helpText {
		t.Errorf("expected help text, got %q", rpc.sentMessages[0].text)
	}
}

func TestHandleGroupMessage_TextFilterMatch(t *testing.T) {
	rpc := &mockRPC{}
	repo := &mockFilterRepository{
		matchingFilters: []domain.FilterResponse{
			{
				ResponseType: domain.ResponseTypeText,
				ResponseText: "I love puppies!",
			},
		},
	}
	logger := &mockLogger{}

	msg := &deltachat.MsgSnapshot{
		ChatId:   100,
		FromId:   42,
		Text:     "look at these puppies",
		ViewType: deltachat.MsgText,
	}

	deps := &domain.Dependencies{
		FilterRepository: repo,
		MediaStorage:     newMockMediaStorage(),
	}

	handleGroupMessage(rpc, logger, deltachat.AccountId(1), deltachat.MsgId(1), msg, deps)

	// Text responses now use SendMsg (with QuotedMessageId) instead of MiscSendTextMessage
	if len(rpc.sentMsgData) != 1 {
		t.Fatalf("expected 1 SendMsg call, got %d", len(rpc.sentMsgData))
	}
	sent := rpc.sentMsgData[0]
	if sent.data.Text != "I love puppies!" {
		t.Errorf("expected text 'I love puppies!', got %q", sent.data.Text)
	}
	if sent.data.QuotedMessageId != 1 {
		t.Errorf("expected QuotedMessageId 1, got %d", sent.data.QuotedMessageId)
	}
	if sent.chatID != 100 {
		t.Errorf("expected chatID 100, got %d", sent.chatID)
	}
}

func TestHandleGroupMessage_ReactionFilterMatch(t *testing.T) {
	rpc := &mockRPC{}
	repo := &mockFilterRepository{
		matchingFilters: []domain.FilterResponse{
			{
				ResponseType: domain.ResponseTypeReaction,
				Reaction:     "😂",
			},
		},
	}
	logger := &mockLogger{}

	msg := &deltachat.MsgSnapshot{
		ChatId:   100,
		FromId:   42,
		Text:     "something funny",
		ViewType: deltachat.MsgText,
	}

	deps := &domain.Dependencies{
		FilterRepository: repo,
		MediaStorage:     newMockMediaStorage(),
	}

	handleGroupMessage(rpc, logger, deltachat.AccountId(1), deltachat.MsgId(5), msg, deps)

	if len(rpc.sentReactions) != 1 {
		t.Fatalf("expected 1 reaction, got %d", len(rpc.sentReactions))
	}
	if len(rpc.sentReactions[0].reaction) != 1 || rpc.sentReactions[0].reaction[0] != "😂" {
		t.Errorf("unexpected reaction: %v", rpc.sentReactions[0].reaction)
	}
	if rpc.sentReactions[0].msgID != 5 {
		t.Errorf("expected reaction on msgID 5, got %d", rpc.sentReactions[0].msgID)
	}
}

func TestHandleGroupMessage_MediaFilterMatch(t *testing.T) {
	mediaHash := "abc123hash"
	storage := newMockMediaStorage()
	// Pre-populate storage with the media file so Exists returns true
	storage.saved[mediaHash] = []byte("fake-image-bytes")

	rpc := &mockRPC{}
	repo := &mockFilterRepository{
		matchingFilters: []domain.FilterResponse{
			{
				FilterID:     1,
				ResponseType: domain.ResponseTypeMedia,
				MediaHash:    mediaHash,
				MediaType:    domain.MediaTypeImage,
			},
		},
	}
	logger := &mockLogger{}

	msg := &deltachat.MsgSnapshot{
		ChatId:   100,
		FromId:   42,
		Text:     "show me a cat",
		ViewType: deltachat.MsgText,
	}

	deps := &domain.Dependencies{
		FilterRepository: repo,
		MediaStorage:     storage,
	}

	handleGroupMessage(rpc, logger, deltachat.AccountId(1), deltachat.MsgId(10), msg, deps)

	if len(logger.errors) > 0 {
		t.Fatalf("unexpected errors: %v", logger.errors)
	}
	if len(rpc.sentMsgData) != 1 {
		t.Fatalf("expected 1 SendMsg call, got %d", len(rpc.sentMsgData))
	}
	sent := rpc.sentMsgData[0]
	if sent.chatID != 100 {
		t.Errorf("expected chatID 100, got %d", sent.chatID)
	}
	expectedPath := filepath.Join("/mock/media", mediaHash)
	if sent.data.File != expectedPath {
		t.Errorf("expected file path %q, got %q", expectedPath, sent.data.File)
	}
	if sent.data.ViewType != deltachat.MsgImage {
		t.Errorf("expected ViewType %q, got %q", deltachat.MsgImage, sent.data.ViewType)
	}
	if sent.data.QuotedMessageId != 10 {
		t.Errorf("expected QuotedMessageId 10, got %d", sent.data.QuotedMessageId)
	}
}

func TestHandleGroupMessage_MediaFilterMatch_MissingFile(t *testing.T) {
	mediaHash := "nonexistent-hash"
	storage := newMockMediaStorage()
	// Do NOT populate storage — file does not exist

	rpc := &mockRPC{}
	repo := &mockFilterRepository{
		matchingFilters: []domain.FilterResponse{
			{
				FilterID:     1,
				ResponseType: domain.ResponseTypeMedia,
				MediaHash:    mediaHash,
				MediaType:    domain.MediaTypeImage,
			},
		},
	}
	logger := &mockLogger{}

	msg := &deltachat.MsgSnapshot{
		ChatId:   100,
		FromId:   42,
		Text:     "show me a cat",
		ViewType: deltachat.MsgText,
	}

	deps := &domain.Dependencies{
		FilterRepository: repo,
		MediaStorage:     storage,
	}

	handleGroupMessage(rpc, logger, deltachat.AccountId(1), deltachat.MsgId(10), msg, deps)

	// Should have logged an error, not sent anything
	if len(rpc.sentMsgData) != 0 {
		t.Errorf("expected no SendMsg calls, got %d", len(rpc.sentMsgData))
	}
	if len(logger.errors) == 0 {
		t.Error("expected an error to be logged for missing media file")
	}
}

func TestHandleGroupMessage_CommandRouting(t *testing.T) {
	rpc := &mockRPC{}
	repo := &mockFilterRepository{}
	logger := &mockLogger{}

	msg := &deltachat.MsgSnapshot{
		ChatId:   100,
		FromId:   42,
		Text:     "/filter hello world response text",
		ViewType: deltachat.MsgText,
	}

	deps := &domain.Dependencies{
		FilterRepository: repo,
		MediaStorage:     newMockMediaStorage(),
	}

	handleGroupMessage(rpc, logger, deltachat.AccountId(1), deltachat.MsgId(1), msg, deps)

	// The /filter command should have been processed, creating a text filter
	if !repo.createTextFilterCalled {
		t.Fatal("expected /filter command to route to handleFilterCommand and create a text filter")
	}
}

func TestConvertChatID(t *testing.T) {
	tests := []struct {
		name    string
		chatID  deltachat.ChatId
		want    int64
		wantErr bool
	}{
		{name: "zero", chatID: 0, want: 0},
		{name: "normal", chatID: 100, want: 100},
		{name: "max int64", chatID: deltachat.ChatId(1<<63 - 1), want: 1<<63 - 1},
		{name: "overflow", chatID: deltachat.ChatId(1 << 63), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := convertChatID(tt.chatID)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("expected %d, got %d", tt.want, got)
			}
		})
	}
}

// --- Prompt Command Tests ---

func TestHandlePromptCommand_NewThread(t *testing.T) {
	aiClient := &mockAIClient{response: "Paris is the capital of France."}
	convRepo := &mockConversationRepo{}
	cfg := &mockConfig{
		systemPrompt:     "You are a helpful assistant.",
		openAIMaxHistory: 50,
	}
	rpc := &mockRPC{}
	logger := &mockLogger{}

	msg := &deltachat.MsgSnapshot{
		ChatId: 100,
		FromId: 42,
		Text:   "/prompt What is the capital of France?",
	}

	deps := &domain.Dependencies{
		FilterRepository:       &mockFilterRepository{},
		MediaStorage:           newMockMediaStorage(),
		AIClient:               aiClient,
		ConversationRepository: convRepo,
		Config:                 cfg,
	}

	handlePromptCommand(rpc, logger, deltachat.AccountId(1), deltachat.MsgId(10), msg, deps)

	// Verify AI was called with system prompt + user message
	if !aiClient.called {
		t.Fatal("expected AIClient.ChatCompletion to be called")
	}
	if len(aiClient.lastMessages) != 2 {
		t.Fatalf("expected 2 messages (system+user), got %d", len(aiClient.lastMessages))
	}
	if aiClient.lastMessages[0].Role != "system" {
		t.Errorf("expected first message role 'system', got %q", aiClient.lastMessages[0].Role)
	}
	if aiClient.lastMessages[1].Content != "What is the capital of France?" {
		t.Errorf("expected user message content, got %q", aiClient.lastMessages[1].Content)
	}

	// Verify response was sent as quote-reply
	if len(rpc.sentMsgData) != 1 {
		t.Fatalf("expected 1 SendMsg call, got %d", len(rpc.sentMsgData))
	}
	if rpc.sentMsgData[0].data.Text != "Paris is the capital of France." {
		t.Errorf("expected AI response text, got %q", rpc.sentMsgData[0].data.Text)
	}
	if rpc.sentMsgData[0].data.QuotedMessageId != 10 {
		t.Errorf("expected QuotedMessageId 10, got %d", rpc.sentMsgData[0].data.QuotedMessageId)
	}

	// Verify both messages were saved
	if len(convRepo.savedMessages) != 2 {
		t.Fatalf("expected 2 saved messages, got %d", len(convRepo.savedMessages))
	}
	// User message: thread_root=10, msg_id=10, parent=nil, role=user
	userMsg := convRepo.savedMessages[0]
	if userMsg.threadRootID != 10 || userMsg.msgID != 10 || userMsg.parentMsgID != nil || userMsg.role != "user" {
		t.Errorf("unexpected user message: %+v", userMsg)
	}
	// Assistant message: thread_root=10, msg_id=1 (from mockRPC), parent=10, role=assistant
	assistantMsg := convRepo.savedMessages[1]
	if assistantMsg.threadRootID != 10 || assistantMsg.role != "assistant" {
		t.Errorf("unexpected assistant message: %+v", assistantMsg)
	}
	if assistantMsg.parentMsgID == nil || *assistantMsg.parentMsgID != 10 {
		t.Errorf("expected assistant parent msg ID 10, got %v", assistantMsg.parentMsgID)
	}
}

func TestHandlePromptCommand_NoSystemPrompt(t *testing.T) {
	aiClient := &mockAIClient{response: "Hi!"}
	convRepo := &mockConversationRepo{}
	cfg := &mockConfig{
		systemPrompt:     "",
		openAIMaxHistory: 50,
	}
	rpc := &mockRPC{}
	logger := &mockLogger{}

	msg := &deltachat.MsgSnapshot{
		ChatId: 100,
		FromId: 42,
		Text:   "/prompt Hello",
	}

	deps := &domain.Dependencies{
		FilterRepository:       &mockFilterRepository{},
		MediaStorage:           newMockMediaStorage(),
		AIClient:               aiClient,
		ConversationRepository: convRepo,
		Config:                 cfg,
	}

	handlePromptCommand(rpc, logger, deltachat.AccountId(1), deltachat.MsgId(10), msg, deps)

	// Without system prompt, only 1 message should be sent
	if len(aiClient.lastMessages) != 1 {
		t.Fatalf("expected 1 message (user only), got %d", len(aiClient.lastMessages))
	}
	if aiClient.lastMessages[0].Role != "user" {
		t.Errorf("expected role 'user', got %q", aiClient.lastMessages[0].Role)
	}
}

func TestHandlePromptCommand_Unconfigured(t *testing.T) {
	rpc := &mockRPC{}
	logger := &mockLogger{}

	msg := &deltachat.MsgSnapshot{
		ChatId: 100,
		FromId: 42,
		Text:   "/prompt Hello",
	}

	deps := &domain.Dependencies{
		FilterRepository: &mockFilterRepository{},
		MediaStorage:     newMockMediaStorage(),
		AIClient:         nil, // Not configured
		Config:           &mockConfig{},
	}

	handlePromptCommand(rpc, logger, deltachat.AccountId(1), deltachat.MsgId(10), msg, deps)

	// Should send error about unconfigured AI
	if len(rpc.sentMsgData) != 1 {
		t.Fatalf("expected 1 error message, got %d", len(rpc.sentMsgData))
	}
	if rpc.sentMsgData[0].data.Text == "" {
		t.Error("expected non-empty error message")
	}
}

func TestHandlePromptCommand_APIError(t *testing.T) {
	aiClient := &mockAIClient{err: fmt.Errorf("API error")}
	convRepo := &mockConversationRepo{}
	cfg := &mockConfig{
		systemPrompt:     "You are helpful.",
		openAIMaxHistory: 50,
	}
	rpc := &mockRPC{}
	logger := &mockLogger{}

	msg := &deltachat.MsgSnapshot{
		ChatId: 100,
		FromId: 42,
		Text:   "/prompt Hello",
	}

	deps := &domain.Dependencies{
		FilterRepository:       &mockFilterRepository{},
		MediaStorage:           newMockMediaStorage(),
		AIClient:               aiClient,
		ConversationRepository: convRepo,
		Config:                 cfg,
	}

	handlePromptCommand(rpc, logger, deltachat.AccountId(1), deltachat.MsgId(10), msg, deps)

	// Should send an error message to user
	if len(rpc.sentMsgData) != 1 {
		t.Fatalf("expected 1 error message, got %d", len(rpc.sentMsgData))
	}
	// No messages should be saved on error
	if len(convRepo.savedMessages) != 0 {
		t.Errorf("expected 0 saved messages on error, got %d", len(convRepo.savedMessages))
	}
}

func TestHandlePromptCommand_AllowlistDenied(t *testing.T) {
	aiClient := &mockAIClient{response: "Should not reach here"}
	cfg := &mockConfig{
		allowedChatIDs: []int64{200, 300}, // Chat 100 is NOT in the list
	}
	rpc := &mockRPC{}
	logger := &mockLogger{}

	msg := &deltachat.MsgSnapshot{
		ChatId: 100,
		FromId: 42,
		Text:   "/prompt Hello",
	}

	deps := &domain.Dependencies{
		FilterRepository:       &mockFilterRepository{},
		MediaStorage:           newMockMediaStorage(),
		AIClient:               aiClient,
		ConversationRepository: &mockConversationRepo{},
		Config:                 cfg,
	}

	handlePromptCommand(rpc, logger, deltachat.AccountId(1), deltachat.MsgId(10), msg, deps)

	// AI should NOT be called
	if aiClient.called {
		t.Fatal("AI client should not be called for denied chat")
	}
	// Should send "not authorized" error
	if len(rpc.sentMsgData) != 1 {
		t.Fatalf("expected 1 error message, got %d", len(rpc.sentMsgData))
	}
}

func TestHandlePromptCommand_AllowlistAllowed(t *testing.T) {
	aiClient := &mockAIClient{response: "Allowed!"}
	cfg := &mockConfig{
		allowedChatIDs:   []int64{100, 200},
		openAIMaxHistory: 50,
	}
	rpc := &mockRPC{}
	logger := &mockLogger{}

	msg := &deltachat.MsgSnapshot{
		ChatId: 100,
		FromId: 42,
		Text:   "/prompt Hello",
	}

	deps := &domain.Dependencies{
		FilterRepository:       &mockFilterRepository{},
		MediaStorage:           newMockMediaStorage(),
		AIClient:               aiClient,
		ConversationRepository: &mockConversationRepo{},
		Config:                 cfg,
	}

	handlePromptCommand(rpc, logger, deltachat.AccountId(1), deltachat.MsgId(10), msg, deps)

	if !aiClient.called {
		t.Fatal("AI client should be called for allowed chat")
	}
}

func TestHandlePromptCommand_EmptyAllowlist(t *testing.T) {
	aiClient := &mockAIClient{response: "Open access!"}
	cfg := &mockConfig{
		allowedChatIDs:   nil, // Empty = allow all
		openAIMaxHistory: 50,
	}
	rpc := &mockRPC{}
	logger := &mockLogger{}

	msg := &deltachat.MsgSnapshot{
		ChatId: 100,
		FromId: 42,
		Text:   "/prompt Hello",
	}

	deps := &domain.Dependencies{
		FilterRepository:       &mockFilterRepository{},
		MediaStorage:           newMockMediaStorage(),
		AIClient:               aiClient,
		ConversationRepository: &mockConversationRepo{},
		Config:                 cfg,
	}

	handlePromptCommand(rpc, logger, deltachat.AccountId(1), deltachat.MsgId(10), msg, deps)

	if !aiClient.called {
		t.Fatal("AI client should be called when allowlist is empty")
	}
}

// --- Thread Continuation Tests ---

func TestHandleThreadContinuation_ValidContinuation(t *testing.T) {
	threadRoot := int64(5)
	aiClient := &mockAIClient{response: "Here's more detail."}
	convRepo := &mockConversationRepo{
		isConvMsgExists:     true,
		isConvMsgThreadRoot: &threadRoot,
		threadChain: []domain.ChatMessage{
			{Role: "user", Content: "What is Go?"},
			{Role: "assistant", Content: "Go is a programming language."},
		},
	}
	cfg := &mockConfig{
		systemPrompt:     "You are helpful.",
		openAIMaxHistory: 50,
	}
	rpc := &mockRPC{}
	logger := &mockLogger{}

	msg := &deltachat.MsgSnapshot{
		ChatId: 100,
		FromId: 42,
		Text:   "Tell me more about Go",
		Quote: &deltachat.MsgQuote{
			MessageId: 6, // Quoting a Patrizio message
		},
	}

	deps := &domain.Dependencies{
		FilterRepository:       &mockFilterRepository{},
		MediaStorage:           newMockMediaStorage(),
		AIClient:               aiClient,
		ConversationRepository: convRepo,
		Config:                 cfg,
	}

	handleThreadContinuation(rpc, logger, deltachat.AccountId(1), deltachat.MsgId(20), msg, deps, threadRoot)

	// Verify AI was called with system + chain + new user message
	if !aiClient.called {
		t.Fatal("expected AI client to be called")
	}
	// system + 2 chain messages + 1 new user = 4
	if len(aiClient.lastMessages) != 4 {
		t.Fatalf("expected 4 messages, got %d", len(aiClient.lastMessages))
	}
	if aiClient.lastMessages[0].Role != "system" {
		t.Errorf("expected first message role 'system', got %q", aiClient.lastMessages[0].Role)
	}
	if aiClient.lastMessages[3].Content != "Tell me more about Go" {
		t.Errorf("expected last message to be user's new text, got %q", aiClient.lastMessages[3].Content)
	}

	// Verify response sent
	if len(rpc.sentMsgData) != 1 {
		t.Fatalf("expected 1 SendMsg, got %d", len(rpc.sentMsgData))
	}
	if rpc.sentMsgData[0].data.Text != "Here's more detail." {
		t.Errorf("expected AI response, got %q", rpc.sentMsgData[0].data.Text)
	}

	// Verify both messages saved
	if len(convRepo.savedMessages) != 2 {
		t.Fatalf("expected 2 saved messages, got %d", len(convRepo.savedMessages))
	}
	// User message: parent is the quoted msg (6)
	userMsg := convRepo.savedMessages[0]
	if userMsg.threadRootID != threadRoot || userMsg.role != "user" {
		t.Errorf("unexpected user message: %+v", userMsg)
	}
	if userMsg.parentMsgID == nil || *userMsg.parentMsgID != 6 {
		t.Errorf("expected parent msg ID 6, got %v", userMsg.parentMsgID)
	}
}

func TestIsThreadContinuation_NoQuote(t *testing.T) {
	msg := &deltachat.MsgSnapshot{
		ChatId: 100,
		Text:   "Just a normal message",
	}

	deps := &domain.Dependencies{
		ConversationRepository: &mockConversationRepo{},
	}

	isCont, _ := isThreadContinuation(context.Background(), msg, deps)
	if isCont {
		t.Error("expected no continuation for message without quote")
	}
}

func TestIsThreadContinuation_NonConversationQuote(t *testing.T) {
	convRepo := &mockConversationRepo{
		isConvMsgExists: false, // Quoted message is NOT a conversation message
	}

	msg := &deltachat.MsgSnapshot{
		ChatId: 100,
		Text:   "Replying to something else",
		Quote: &deltachat.MsgQuote{
			MessageId: 999,
		},
	}

	deps := &domain.Dependencies{
		ConversationRepository: convRepo,
	}

	isCont, _ := isThreadContinuation(context.Background(), msg, deps)
	if isCont {
		t.Error("expected no continuation for non-conversation quote")
	}
}

func TestIsThreadContinuation_NilRepo(t *testing.T) {
	msg := &deltachat.MsgSnapshot{
		ChatId: 100,
		Text:   "Some message",
		Quote: &deltachat.MsgQuote{
			MessageId: 5,
		},
	}

	deps := &domain.Dependencies{
		ConversationRepository: nil,
	}

	isCont, _ := isThreadContinuation(context.Background(), msg, deps)
	if isCont {
		t.Error("expected no continuation when ConversationRepository is nil")
	}
}

func TestHandleThreadContinuation_Unconfigured(t *testing.T) {
	rpc := &mockRPC{}
	logger := &mockLogger{}

	msg := &deltachat.MsgSnapshot{
		ChatId: 100,
		FromId: 42,
		Text:   "Continue thread",
		Quote: &deltachat.MsgQuote{
			MessageId: 5,
		},
	}

	deps := &domain.Dependencies{
		FilterRepository:       &mockFilterRepository{},
		MediaStorage:           newMockMediaStorage(),
		AIClient:               nil, // Not configured
		ConversationRepository: &mockConversationRepo{},
		Config:                 &mockConfig{},
	}

	handleThreadContinuation(rpc, logger, deltachat.AccountId(1), deltachat.MsgId(20), msg, deps, 5)

	// Should send error about unconfigured AI
	if len(rpc.sentMsgData) != 1 {
		t.Fatalf("expected 1 error message, got %d", len(rpc.sentMsgData))
	}
}

func TestHandleThreadContinuation_AllowlistDenied(t *testing.T) {
	aiClient := &mockAIClient{response: "Should not reach here"}
	cfg := &mockConfig{
		allowedChatIDs: []int64{200}, // Chat 100 NOT allowed
	}
	rpc := &mockRPC{}
	logger := &mockLogger{}

	msg := &deltachat.MsgSnapshot{
		ChatId: 100,
		FromId: 42,
		Text:   "Continue",
		Quote: &deltachat.MsgQuote{
			MessageId: 5,
		},
	}

	deps := &domain.Dependencies{
		FilterRepository:       &mockFilterRepository{},
		MediaStorage:           newMockMediaStorage(),
		AIClient:               aiClient,
		ConversationRepository: &mockConversationRepo{},
		Config:                 cfg,
	}

	handleThreadContinuation(rpc, logger, deltachat.AccountId(1), deltachat.MsgId(20), msg, deps, 5)

	if aiClient.called {
		t.Fatal("AI client should not be called for denied chat")
	}
}

// --- DM Handler with Prompt ---

func TestHandleDMMessage_PromptCommand(t *testing.T) {
	aiClient := &mockAIClient{response: "AI response in DM"}
	convRepo := &mockConversationRepo{}
	cfg := &mockConfig{
		systemPrompt:     "Be helpful",
		openAIMaxHistory: 50,
	}
	rpc := &mockRPC{}
	logger := &mockLogger{}

	msg := &deltachat.MsgSnapshot{
		ChatId: 50,
		FromId: 42,
		Text:   "/prompt Hello from DM",
	}

	deps := &domain.Dependencies{
		FilterRepository:       &mockFilterRepository{},
		MediaStorage:           newMockMediaStorage(),
		AIClient:               aiClient,
		ConversationRepository: convRepo,
		Config:                 cfg,
	}

	handleDMMessage(rpc, logger, deltachat.AccountId(1), deltachat.MsgId(1), msg, deps)

	if !aiClient.called {
		t.Fatal("expected AI client to be called for /prompt in DM")
	}
	// Should NOT have sent help text
	if len(rpc.sentMessages) != 0 {
		t.Errorf("expected no MiscSendTextMessage calls, got %d", len(rpc.sentMessages))
	}
}

func TestHandleDMMessage_ThreadContinuation(t *testing.T) {
	threadRoot := int64(5)
	aiClient := &mockAIClient{response: "Continued in DM"}
	convRepo := &mockConversationRepo{
		isConvMsgExists:     true,
		isConvMsgThreadRoot: &threadRoot,
		threadChain: []domain.ChatMessage{
			{Role: "user", Content: "Original prompt"},
			{Role: "assistant", Content: "Original response"},
		},
	}
	cfg := &mockConfig{
		systemPrompt:     "Be helpful",
		openAIMaxHistory: 50,
	}
	rpc := &mockRPC{}
	logger := &mockLogger{}

	msg := &deltachat.MsgSnapshot{
		ChatId: 50,
		FromId: 42,
		Text:   "Follow up in DM",
		Quote: &deltachat.MsgQuote{
			MessageId: 6,
		},
	}

	deps := &domain.Dependencies{
		FilterRepository:       &mockFilterRepository{},
		MediaStorage:           newMockMediaStorage(),
		AIClient:               aiClient,
		ConversationRepository: convRepo,
		Config:                 cfg,
	}

	handleDMMessage(rpc, logger, deltachat.AccountId(1), deltachat.MsgId(20), msg, deps)

	if !aiClient.called {
		t.Fatal("expected AI client to be called for thread continuation in DM")
	}
	// Should NOT have sent help text
	if len(rpc.sentMessages) != 0 {
		t.Errorf("expected no MiscSendTextMessage calls, got %d", len(rpc.sentMessages))
	}
}

// --- Group Handler with Prompt ---

func TestHandleGroupMessage_PromptCommand(t *testing.T) {
	aiClient := &mockAIClient{response: "Group AI response"}
	convRepo := &mockConversationRepo{}
	cfg := &mockConfig{
		systemPrompt:     "You are helpful.",
		openAIMaxHistory: 50,
	}
	rpc := &mockRPC{}
	logger := &mockLogger{}

	msg := &deltachat.MsgSnapshot{
		ChatId: 100,
		FromId: 42,
		Text:   "/prompt Tell me a joke",
	}

	deps := &domain.Dependencies{
		FilterRepository:       &mockFilterRepository{},
		MediaStorage:           newMockMediaStorage(),
		AIClient:               aiClient,
		ConversationRepository: convRepo,
		Config:                 cfg,
	}

	handleGroupMessage(rpc, logger, deltachat.AccountId(1), deltachat.MsgId(10), msg, deps)

	if !aiClient.called {
		t.Fatal("expected AI client to be called for /prompt in group")
	}
}

func TestHandleGroupMessage_ThreadContinuation(t *testing.T) {
	threadRoot := int64(5)
	aiClient := &mockAIClient{response: "Continued in group"}
	convRepo := &mockConversationRepo{
		isConvMsgExists:     true,
		isConvMsgThreadRoot: &threadRoot,
		threadChain: []domain.ChatMessage{
			{Role: "user", Content: "Original prompt"},
			{Role: "assistant", Content: "Original response"},
		},
	}
	cfg := &mockConfig{
		systemPrompt:     "Be helpful",
		openAIMaxHistory: 50,
	}
	rpc := &mockRPC{}
	repo := &mockFilterRepository{}
	logger := &mockLogger{}

	msg := &deltachat.MsgSnapshot{
		ChatId: 100,
		FromId: 42,
		Text:   "Continue the thread",
		Quote: &deltachat.MsgQuote{
			MessageId: 6,
		},
	}

	deps := &domain.Dependencies{
		FilterRepository:       repo,
		MediaStorage:           newMockMediaStorage(),
		AIClient:               aiClient,
		ConversationRepository: convRepo,
		Config:                 cfg,
	}

	handleGroupMessage(rpc, logger, deltachat.AccountId(1), deltachat.MsgId(20), msg, deps)

	if !aiClient.called {
		t.Fatal("expected AI client to be called for thread continuation in group")
	}
	// Should NOT have tried filter matching
	if repo.matchingFilters != nil {
		t.Error("filter matching should not have been attempted")
	}
}

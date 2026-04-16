package bot

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/polpetta/patrizio/internal/domain"
)

// --- Mock Messenger ---

type mockMessenger struct {
	// FetchMessage
	fetchMessageFn func(uint32, uint32) (*domain.IncomingMessage, error)
	// FetchChatType
	fetchChatTypeFn func(uint32, uint32) (domain.ChatType, error)
	// FetchContactDisplayName
	fetchContactDisplayNameFn func(uint32, uint32) (string, error)
	// SendTextMessage
	sentTextMessages   []sentTextMessageEntry
	sendTextMessageErr error
	// SendTextReply
	sentTextReplies  []sentTextReplyEntry
	sendTextReplyErr error
	// SendMediaReply
	sentMediaReplies  []sentMediaReplyEntry
	sendMediaReplyErr error
	// SendReaction
	sentReactions []sentReactionEntry
	reactionErr   error
	// DownloadMessage
	downloadCalled bool
	downloadErr    error
}

type sentTextMessageEntry struct {
	accID  uint32
	chatID uint32
	text   string
}

type sentTextReplyEntry struct {
	accID  uint32
	chatID uint32
	replyTo uint32
	text   string
}

type sentMediaReplyEntry struct {
	accID     uint32
	chatID    uint32
	replyTo   uint32
	filePath  string
	mediaType string
}

type sentReactionEntry struct {
	accID    uint32
	msgID    uint32
	reaction string
}

func (m *mockMessenger) FetchMessage(accountID uint32, msgID uint32) (*domain.IncomingMessage, error) {
	if m.fetchMessageFn != nil {
		return m.fetchMessageFn(accountID, msgID)
	}
	return nil, fmt.Errorf("FetchMessage not configured")
}

func (m *mockMessenger) FetchChatType(accountID uint32, chatID uint32) (domain.ChatType, error) {
	if m.fetchChatTypeFn != nil {
		return m.fetchChatTypeFn(accountID, chatID)
	}
	return "", fmt.Errorf("FetchChatType not configured")
}

func (m *mockMessenger) SendTextMessage(accountID uint32, chatID uint32, text string) error {
	m.sentTextMessages = append(m.sentTextMessages, sentTextMessageEntry{accID: accountID, chatID: chatID, text: text})
	return m.sendTextMessageErr
}

func (m *mockMessenger) SendTextReply(accountID uint32, chatID uint32, replyTo uint32, text string) (uint32, error) {
	m.sentTextReplies = append(m.sentTextReplies, sentTextReplyEntry{accID: accountID, chatID: chatID, replyTo: replyTo, text: text})
	return uint32(1), m.sendTextReplyErr
}

func (m *mockMessenger) SendMediaReply(accountID uint32, chatID uint32, replyTo uint32, filePath string, mediaType string) (uint32, error) {
	m.sentMediaReplies = append(m.sentMediaReplies, sentMediaReplyEntry{accID: accountID, chatID: chatID, replyTo: replyTo, filePath: filePath, mediaType: mediaType})
	return uint32(1), m.sendMediaReplyErr
}

func (m *mockMessenger) SendReaction(accountID uint32, msgID uint32, reaction string) error {
	m.sentReactions = append(m.sentReactions, sentReactionEntry{accID: accountID, msgID: msgID, reaction: reaction})
	return m.reactionErr
}

func (m *mockMessenger) DownloadMessage(_ uint32, _ uint32) error {
	m.downloadCalled = true
	return m.downloadErr
}

func (m *mockMessenger) IsSpecialContact(fromID uint32) bool {
	return fromID <= 9
}

func (m *mockMessenger) FetchContactDisplayName(accountID uint32, contactID uint32) (string, error) {
	if m.fetchContactDisplayNameFn != nil {
		return m.fetchContactDisplayNameFn(accountID, contactID)
	}
	return "", nil
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
	senderName   string
}

func (m *mockConversationRepo) SaveMessage(_ context.Context, threadRootID int64, msgID int64, parentMsgID *int64, role string, content string, senderName string) error {
	m.savedMessages = append(m.savedMessages, savedConversationMessage{
		threadRootID: threadRootID,
		msgID:        msgID,
		parentMsgID:  parentMsgID,
		role:         role,
		content:      content,
		senderName:   senderName,
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
	mock := &mockMessenger{}
	repo := &mockFilterRepository{}
	storage := newMockMediaStorage()
	logger := &mockLogger{}

	msg := &domain.IncomingMessage{
		ChatID: 100,
		FromID: 42,
		Text:   "/filter hello Hi there!",
	}

	deps := &domain.Dependencies{
		FilterRepository: repo,
		MediaStorage:     storage,
		Messenger:        mock,
	}

	handleFilterCommand(logger, uint32(1), uint32(7), msg, deps)

	if !repo.createTextFilterCalled {
		t.Fatal("expected CreateTextFilter to be called")
	}
	if repo.lastResponse != "Hi there!" {
		t.Errorf("expected response 'Hi there!', got %q", repo.lastResponse)
	}
	if len(repo.lastTriggers) != 1 || repo.lastTriggers[0] != domain.NormalizeTrigger("hello") {
		t.Errorf("unexpected triggers: %v", repo.lastTriggers)
	}

	// Should have sent a confirmation as a quote-reply
	if len(mock.sentTextReplies) != 1 {
		t.Fatalf("expected 1 SendTextReply call (confirmation), got %d", len(mock.sentTextReplies))
	}
	if mock.sentTextReplies[0].chatID != 100 {
		t.Errorf("expected confirmation sent to chat 100, got %d", mock.sentTextReplies[0].chatID)
	}
	if mock.sentTextReplies[0].replyTo != 7 {
		t.Errorf("expected replyTo 7, got %d", mock.sentTextReplies[0].replyTo)
	}
}

func TestHandleFilterCommand_ReactionFilter(t *testing.T) {
	mock := &mockMessenger{}
	repo := &mockFilterRepository{}
	storage := newMockMediaStorage()
	logger := &mockLogger{}

	msg := &domain.IncomingMessage{
		ChatID: 100,
		FromID: 42,
		Text:   "/filter hello react:😂",
	}

	deps := &domain.Dependencies{
		FilterRepository: repo,
		MediaStorage:     storage,
		Messenger:        mock,
	}

	handleFilterCommand(logger, uint32(1), uint32(7), msg, deps)

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

	mock := &mockMessenger{}
	repo := &mockFilterRepository{}
	storage := newMockMediaStorage()
	logger := &mockLogger{}

	// Message has /filter cat with no text response, but carries an image attachment.
	msg := &domain.IncomingMessage{
		ChatID:    100,
		FromID:    42,
		Text:      "/filter cat",
		MediaType: domain.MediaTypeImage,
		File:      mediaPath,
	}

	deps := &domain.Dependencies{
		FilterRepository: repo,
		MediaStorage:     storage,
		Messenger:        mock,
	}

	handleFilterCommand(logger, uint32(1), uint32(7), msg, deps)

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

	// Should have sent a confirmation as a quote-reply
	if len(mock.sentTextReplies) != 1 {
		t.Fatalf("expected 1 SendTextReply call (confirmation), got %d", len(mock.sentTextReplies))
	}
	if mock.sentTextReplies[0].replyTo != 7 {
		t.Errorf("expected replyTo 7, got %d", mock.sentTextReplies[0].replyTo)
	}
}

// TestHandleFilterCommand_MediaFromAttachment_MultipleTriggers tests media filter with
// multiple triggers and an attached image.
func TestHandleFilterCommand_MediaFromAttachment_MultipleTriggers(t *testing.T) {
	mediaContent := []byte("another-fake-image")
	mediaPath := writeTempFile(t, mediaContent)
	expectedHash := computeSHA512(mediaContent) + filepath.Ext(mediaPath)

	mock := &mockMessenger{}
	repo := &mockFilterRepository{}
	storage := newMockMediaStorage()
	logger := &mockLogger{}

	msg := &domain.IncomingMessage{
		ChatID:    200,
		FromID:    42,
		Text:      `/filter (cat, dog, "cute animals")`,
		MediaType: domain.MediaTypeImage,
		File:      mediaPath,
	}

	deps := &domain.Dependencies{
		FilterRepository: repo,
		MediaStorage:     storage,
		Messenger:        mock,
	}

	handleFilterCommand(logger, uint32(1), uint32(7), msg, deps)

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

	mock := &mockMessenger{}
	repo := &mockFilterRepository{}
	storage := newMockMediaStorage()
	logger := &mockLogger{}

	// The quoted message returns an image with DownloadDone state
	mock.fetchMessageFn = func(_ uint32, msgID uint32) (*domain.IncomingMessage, error) {
		if msgID == 999 {
			return &domain.IncomingMessage{
				ID:            999,
				MediaType:     domain.MediaTypeImage,
				File:          mediaPath,
				DownloadState: domain.DownloadDone,
			}, nil
		}
		return nil, fmt.Errorf("unexpected msgID %d", msgID)
	}

	// /filter cat with no text, replying to a media message
	msg := &domain.IncomingMessage{
		ChatID: 100,
		FromID: 42,
		Text:   "/filter cat",
		Quote: &domain.QuotedMessage{
			MessageID: 999,
		},
	}

	deps := &domain.Dependencies{
		FilterRepository: repo,
		MediaStorage:     storage,
		Messenger:        mock,
	}

	handleFilterCommand(logger, uint32(1), uint32(7), msg, deps)

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

	mock := &mockMessenger{}
	repo := &mockFilterRepository{}
	storage := newMockMediaStorage()
	logger := &mockLogger{}

	// Quoted message would return a different image, but it should not be used
	mock.fetchMessageFn = func(_ uint32, _ uint32) (*domain.IncomingMessage, error) {
		t.Error("FetchMessage should not be called when attachment is present")
		return nil, fmt.Errorf("should not be called")
	}

	msg := &domain.IncomingMessage{
		ChatID:    100,
		FromID:    42,
		Text:      "/filter cat",
		MediaType: domain.MediaTypeSticker,
		File:      mediaPath,
		Quote: &domain.QuotedMessage{
			MessageID: 999,
		},
	}

	deps := &domain.Dependencies{
		FilterRepository: repo,
		MediaStorage:     storage,
		Messenger:        mock,
	}

	handleFilterCommand(logger, uint32(1), uint32(7), msg, deps)

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
	mock := &mockMessenger{}
	repo := &mockFilterRepository{}
	storage := newMockMediaStorage()
	logger := &mockLogger{}

	// /filter cat with no text and no media anywhere — should produce an error
	msg := &domain.IncomingMessage{
		ChatID: 100,
		FromID: 42,
		Text:   "/filter cat",
	}

	deps := &domain.Dependencies{
		FilterRepository: repo,
		MediaStorage:     storage,
		Messenger:        mock,
	}

	handleFilterCommand(logger, uint32(1), uint32(7), msg, deps)

	if repo.createMediaFilterCalled || repo.createTextFilterCalled {
		t.Fatal("no filter should have been created")
	}

	// Should have sent an error message as a quote-reply
	if len(mock.sentTextReplies) != 1 {
		t.Fatalf("expected 1 SendTextReply call (error message), got %d", len(mock.sentTextReplies))
	}
	if mock.sentTextReplies[0].replyTo != 7 {
		t.Errorf("expected replyTo 7, got %d", mock.sentTextReplies[0].replyTo)
	}
}

func TestHandleDMMessage(t *testing.T) {
	mock := &mockMessenger{}
	logger := &mockLogger{}

	msg := &domain.IncomingMessage{
		ChatID: 50,
		FromID: 42,
		Text:   "hey there",
	}

	deps := &domain.Dependencies{
		FilterRepository: &mockFilterRepository{},
		MediaStorage:     newMockMediaStorage(),
		Messenger:        mock,
	}

	handleDMMessage(logger, uint32(1), uint32(1), msg, deps)

	if len(mock.sentTextMessages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(mock.sentTextMessages))
	}
	if mock.sentTextMessages[0].text != helpText {
		t.Errorf("expected help text, got %q", mock.sentTextMessages[0].text)
	}
}

func TestHandleGroupMessage_TextFilterMatch(t *testing.T) {
	mock := &mockMessenger{}
	repo := &mockFilterRepository{
		matchingFilters: []domain.FilterResponse{
			{
				ResponseType: domain.ResponseTypeText,
				ResponseText: "I love puppies!",
			},
		},
	}
	logger := &mockLogger{}

	msg := &domain.IncomingMessage{
		ChatID: 100,
		FromID: 42,
		Text:   "look at these puppies",
	}

	deps := &domain.Dependencies{
		FilterRepository: repo,
		MediaStorage:     newMockMediaStorage(),
		Messenger:        mock,
	}

	handleGroupMessage(logger, uint32(1), uint32(1), msg, deps)

	if len(mock.sentTextReplies) != 1 {
		t.Fatalf("expected 1 SendTextReply call, got %d", len(mock.sentTextReplies))
	}
	sent := mock.sentTextReplies[0]
	if sent.text != "I love puppies!" {
		t.Errorf("expected text 'I love puppies!', got %q", sent.text)
	}
	if sent.replyTo != 1 {
		t.Errorf("expected replyTo 1, got %d", sent.replyTo)
	}
	if sent.chatID != 100 {
		t.Errorf("expected chatID 100, got %d", sent.chatID)
	}
}

func TestHandleGroupMessage_ReactionFilterMatch(t *testing.T) {
	mock := &mockMessenger{}
	repo := &mockFilterRepository{
		matchingFilters: []domain.FilterResponse{
			{
				ResponseType: domain.ResponseTypeReaction,
				Reaction:     "😂",
			},
		},
	}
	logger := &mockLogger{}

	msg := &domain.IncomingMessage{
		ChatID: 100,
		FromID: 42,
		Text:   "something funny",
	}

	deps := &domain.Dependencies{
		FilterRepository: repo,
		MediaStorage:     newMockMediaStorage(),
		Messenger:        mock,
	}

	handleGroupMessage(logger, uint32(1), uint32(5), msg, deps)

	if len(mock.sentReactions) != 1 {
		t.Fatalf("expected 1 reaction, got %d", len(mock.sentReactions))
	}
	if mock.sentReactions[0].reaction != "😂" {
		t.Errorf("unexpected reaction: %v", mock.sentReactions[0].reaction)
	}
	if mock.sentReactions[0].msgID != 5 {
		t.Errorf("expected reaction on msgID 5, got %d", mock.sentReactions[0].msgID)
	}
}

func TestHandleGroupMessage_MediaFilterMatch(t *testing.T) {
	mediaHash := "abc123hash"
	storage := newMockMediaStorage()
	// Pre-populate storage with the media file so Exists returns true
	storage.saved[mediaHash] = []byte("fake-image-bytes")

	mock := &mockMessenger{}
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

	msg := &domain.IncomingMessage{
		ChatID: 100,
		FromID: 42,
		Text:   "show me a cat",
	}

	deps := &domain.Dependencies{
		FilterRepository: repo,
		MediaStorage:     storage,
		Messenger:        mock,
	}

	handleGroupMessage(logger, uint32(1), uint32(10), msg, deps)

	if len(logger.errors) > 0 {
		t.Fatalf("unexpected errors: %v", logger.errors)
	}
	if len(mock.sentMediaReplies) != 1 {
		t.Fatalf("expected 1 SendMediaReply call, got %d", len(mock.sentMediaReplies))
	}
	sent := mock.sentMediaReplies[0]
	if sent.chatID != 100 {
		t.Errorf("expected chatID 100, got %d", sent.chatID)
	}
	expectedPath := filepath.Join("/mock/media", mediaHash)
	if sent.filePath != expectedPath {
		t.Errorf("expected file path %q, got %q", expectedPath, sent.filePath)
	}
	if sent.mediaType != domain.MediaTypeImage {
		t.Errorf("expected mediaType %q, got %q", domain.MediaTypeImage, sent.mediaType)
	}
	if sent.replyTo != 10 {
		t.Errorf("expected replyTo 10, got %d", sent.replyTo)
	}
}

func TestHandleGroupMessage_MediaFilterMatch_MissingFile(t *testing.T) {
	mediaHash := "nonexistent-hash"
	storage := newMockMediaStorage()
	// Do NOT populate storage — file does not exist

	mock := &mockMessenger{}
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

	msg := &domain.IncomingMessage{
		ChatID: 100,
		FromID: 42,
		Text:   "show me a cat",
	}

	deps := &domain.Dependencies{
		FilterRepository: repo,
		MediaStorage:     storage,
		Messenger:        mock,
	}

	handleGroupMessage(logger, uint32(1), uint32(10), msg, deps)

	// Should have logged an error, not sent anything
	if len(mock.sentMediaReplies) != 0 {
		t.Errorf("expected no SendMediaReply calls, got %d", len(mock.sentMediaReplies))
	}
	if len(logger.errors) == 0 {
		t.Error("expected an error to be logged for missing media file")
	}
}

func TestHandleGroupMessage_CommandRouting(t *testing.T) {
	mock := &mockMessenger{}
	repo := &mockFilterRepository{}
	logger := &mockLogger{}

	msg := &domain.IncomingMessage{
		ChatID: 100,
		FromID: 42,
		Text:   "/filter hello world response text",
	}

	deps := &domain.Dependencies{
		FilterRepository: repo,
		MediaStorage:     newMockMediaStorage(),
		Messenger:        mock,
	}

	handleGroupMessage(logger, uint32(1), uint32(1), msg, deps)

	// The /filter command should have been processed, creating a text filter
	if !repo.createTextFilterCalled {
		t.Fatal("expected /filter command to route to handleFilterCommand and create a text filter")
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
	mock := &mockMessenger{}
	logger := &mockLogger{}

	msg := &domain.IncomingMessage{
		ChatID: 100,
		FromID: 42,
		Text:   "/prompt What is the capital of France?",
	}

	deps := &domain.Dependencies{
		FilterRepository:       &mockFilterRepository{},
		MediaStorage:           newMockMediaStorage(),
		AIClient:               aiClient,
		ConversationRepository: convRepo,
		Config:                 cfg,
		Messenger:              mock,
	}

	handlePromptCommand(logger, uint32(1), uint32(10), msg, deps, domain.ChatTypeSingle)

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
	if len(mock.sentTextReplies) != 1 {
		t.Fatalf("expected 1 SendTextReply call, got %d", len(mock.sentTextReplies))
	}
	if mock.sentTextReplies[0].text != "Paris is the capital of France." {
		t.Errorf("expected AI response text, got %q", mock.sentTextReplies[0].text)
	}
	if mock.sentTextReplies[0].replyTo != 10 {
		t.Errorf("expected replyTo 10, got %d", mock.sentTextReplies[0].replyTo)
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
	// Assistant message: thread_root=10, msg_id=1 (from mockMessenger), parent=10, role=assistant
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
	mock := &mockMessenger{}
	logger := &mockLogger{}

	msg := &domain.IncomingMessage{
		ChatID: 100,
		FromID: 42,
		Text:   "/prompt Hello",
	}

	deps := &domain.Dependencies{
		FilterRepository:       &mockFilterRepository{},
		MediaStorage:           newMockMediaStorage(),
		AIClient:               aiClient,
		ConversationRepository: convRepo,
		Config:                 cfg,
		Messenger:              mock,
	}

	handlePromptCommand(logger, uint32(1), uint32(10), msg, deps, domain.ChatTypeSingle)

	// Without system prompt, only 1 message should be sent
	if len(aiClient.lastMessages) != 1 {
		t.Fatalf("expected 1 message (user only), got %d", len(aiClient.lastMessages))
	}
	if aiClient.lastMessages[0].Role != "user" {
		t.Errorf("expected role 'user', got %q", aiClient.lastMessages[0].Role)
	}
}

func TestHandlePromptCommand_Unconfigured(t *testing.T) {
	mock := &mockMessenger{}
	logger := &mockLogger{}

	msg := &domain.IncomingMessage{
		ChatID: 100,
		FromID: 42,
		Text:   "/prompt Hello",
	}

	deps := &domain.Dependencies{
		FilterRepository: &mockFilterRepository{},
		MediaStorage:     newMockMediaStorage(),
		AIClient:         nil, // Not configured
		Config:           &mockConfig{},
		Messenger:        mock,
	}

	handlePromptCommand(logger, uint32(1), uint32(10), msg, deps, domain.ChatTypeSingle)

	// Should send error about unconfigured AI
	if len(mock.sentTextReplies) != 1 {
		t.Fatalf("expected 1 error message, got %d", len(mock.sentTextReplies))
	}
	if mock.sentTextReplies[0].text == "" {
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
	mock := &mockMessenger{}
	logger := &mockLogger{}

	msg := &domain.IncomingMessage{
		ChatID: 100,
		FromID: 42,
		Text:   "/prompt Hello",
	}

	deps := &domain.Dependencies{
		FilterRepository:       &mockFilterRepository{},
		MediaStorage:           newMockMediaStorage(),
		AIClient:               aiClient,
		ConversationRepository: convRepo,
		Config:                 cfg,
		Messenger:              mock,
	}

	handlePromptCommand(logger, uint32(1), uint32(10), msg, deps, domain.ChatTypeSingle)

	// Should send an error message to user
	if len(mock.sentTextReplies) != 1 {
		t.Fatalf("expected 1 error message, got %d", len(mock.sentTextReplies))
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
	mock := &mockMessenger{}
	logger := &mockLogger{}

	msg := &domain.IncomingMessage{
		ChatID: 100,
		FromID: 42,
		Text:   "/prompt Hello",
	}

	deps := &domain.Dependencies{
		FilterRepository:       &mockFilterRepository{},
		MediaStorage:           newMockMediaStorage(),
		AIClient:               aiClient,
		ConversationRepository: &mockConversationRepo{},
		Config:                 cfg,
		Messenger:              mock,
	}

	handlePromptCommand(logger, uint32(1), uint32(10), msg, deps, domain.ChatTypeSingle)

	// AI should NOT be called
	if aiClient.called {
		t.Fatal("AI client should not be called for denied chat")
	}
	// Should send "not authorized" error
	if len(mock.sentTextReplies) != 1 {
		t.Fatalf("expected 1 error message, got %d", len(mock.sentTextReplies))
	}
}

func TestHandlePromptCommand_AllowlistAllowed(t *testing.T) {
	aiClient := &mockAIClient{response: "Allowed!"}
	cfg := &mockConfig{
		allowedChatIDs:   []int64{100, 200},
		openAIMaxHistory: 50,
	}
	mock := &mockMessenger{}
	logger := &mockLogger{}

	msg := &domain.IncomingMessage{
		ChatID: 100,
		FromID: 42,
		Text:   "/prompt Hello",
	}

	deps := &domain.Dependencies{
		FilterRepository:       &mockFilterRepository{},
		MediaStorage:           newMockMediaStorage(),
		AIClient:               aiClient,
		ConversationRepository: &mockConversationRepo{},
		Config:                 cfg,
		Messenger:              mock,
	}

	handlePromptCommand(logger, uint32(1), uint32(10), msg, deps, domain.ChatTypeSingle)

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
	mock := &mockMessenger{}
	logger := &mockLogger{}

	msg := &domain.IncomingMessage{
		ChatID: 100,
		FromID: 42,
		Text:   "/prompt Hello",
	}

	deps := &domain.Dependencies{
		FilterRepository:       &mockFilterRepository{},
		MediaStorage:           newMockMediaStorage(),
		AIClient:               aiClient,
		ConversationRepository: &mockConversationRepo{},
		Config:                 cfg,
		Messenger:              mock,
	}

	handlePromptCommand(logger, uint32(1), uint32(10), msg, deps, domain.ChatTypeSingle)

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
	mock := &mockMessenger{}
	logger := &mockLogger{}

	msg := &domain.IncomingMessage{
		ChatID: 100,
		FromID: 42,
		Text:   "Tell me more about Go",
		Quote: &domain.QuotedMessage{
			MessageID: 6, // Quoting a Patrizio message
		},
	}

	deps := &domain.Dependencies{
		FilterRepository:       &mockFilterRepository{},
		MediaStorage:           newMockMediaStorage(),
		AIClient:               aiClient,
		ConversationRepository: convRepo,
		Config:                 cfg,
		Messenger:              mock,
	}

	handleThreadContinuation(logger, uint32(1), uint32(20), msg, deps, threadRoot, domain.ChatTypeSingle)

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
	if len(mock.sentTextReplies) != 1 {
		t.Fatalf("expected 1 SendTextReply, got %d", len(mock.sentTextReplies))
	}
	if mock.sentTextReplies[0].text != "Here's more detail." {
		t.Errorf("expected AI response, got %q", mock.sentTextReplies[0].text)
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
	msg := &domain.IncomingMessage{
		ChatID: 100,
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

	msg := &domain.IncomingMessage{
		ChatID: 100,
		Text:   "Replying to something else",
		Quote: &domain.QuotedMessage{
			MessageID: 999,
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
	msg := &domain.IncomingMessage{
		ChatID: 100,
		Text:   "Some message",
		Quote: &domain.QuotedMessage{
			MessageID: 5,
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
	mock := &mockMessenger{}
	logger := &mockLogger{}

	msg := &domain.IncomingMessage{
		ChatID: 100,
		FromID: 42,
		Text:   "Continue thread",
		Quote: &domain.QuotedMessage{
			MessageID: 5,
		},
	}

	deps := &domain.Dependencies{
		FilterRepository:       &mockFilterRepository{},
		MediaStorage:           newMockMediaStorage(),
		AIClient:               nil, // Not configured
		ConversationRepository: &mockConversationRepo{},
		Config:                 &mockConfig{},
		Messenger:              mock,
	}

	handleThreadContinuation(logger, uint32(1), uint32(20), msg, deps, 5, domain.ChatTypeSingle)

	// Should send error about unconfigured AI
	if len(mock.sentTextReplies) != 1 {
		t.Fatalf("expected 1 error message, got %d", len(mock.sentTextReplies))
	}
}

func TestHandleThreadContinuation_AllowlistDenied(t *testing.T) {
	aiClient := &mockAIClient{response: "Should not reach here"}
	cfg := &mockConfig{
		allowedChatIDs: []int64{200}, // Chat 100 NOT allowed
	}
	mock := &mockMessenger{}
	logger := &mockLogger{}

	msg := &domain.IncomingMessage{
		ChatID: 100,
		FromID: 42,
		Text:   "Continue",
		Quote: &domain.QuotedMessage{
			MessageID: 5,
		},
	}

	deps := &domain.Dependencies{
		FilterRepository:       &mockFilterRepository{},
		MediaStorage:           newMockMediaStorage(),
		AIClient:               aiClient,
		ConversationRepository: &mockConversationRepo{},
		Config:                 cfg,
		Messenger:              mock,
	}

	handleThreadContinuation(logger, uint32(1), uint32(20), msg, deps, 5, domain.ChatTypeSingle)

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
	mock := &mockMessenger{}
	logger := &mockLogger{}

	msg := &domain.IncomingMessage{
		ChatID: 50,
		FromID: 42,
		Text:   "/prompt Hello from DM",
	}

	deps := &domain.Dependencies{
		FilterRepository:       &mockFilterRepository{},
		MediaStorage:           newMockMediaStorage(),
		AIClient:               aiClient,
		ConversationRepository: convRepo,
		Config:                 cfg,
		Messenger:              mock,
	}

	handleDMMessage(logger, uint32(1), uint32(1), msg, deps)

	if !aiClient.called {
		t.Fatal("expected AI client to be called for /prompt in DM")
	}
	// Should NOT have sent help text
	if len(mock.sentTextMessages) != 0 {
		t.Errorf("expected no SendTextMessage calls, got %d", len(mock.sentTextMessages))
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
	mock := &mockMessenger{}
	logger := &mockLogger{}

	msg := &domain.IncomingMessage{
		ChatID: 50,
		FromID: 42,
		Text:   "Follow up in DM",
		Quote: &domain.QuotedMessage{
			MessageID: 6,
		},
	}

	deps := &domain.Dependencies{
		FilterRepository:       &mockFilterRepository{},
		MediaStorage:           newMockMediaStorage(),
		AIClient:               aiClient,
		ConversationRepository: convRepo,
		Config:                 cfg,
		Messenger:              mock,
	}

	handleDMMessage(logger, uint32(1), uint32(20), msg, deps)

	if !aiClient.called {
		t.Fatal("expected AI client to be called for thread continuation in DM")
	}
	// Should NOT have sent help text
	if len(mock.sentTextMessages) != 0 {
		t.Errorf("expected no SendTextMessage calls, got %d", len(mock.sentTextMessages))
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
	mock := &mockMessenger{}
	logger := &mockLogger{}

	msg := &domain.IncomingMessage{
		ChatID: 100,
		FromID: 42,
		Text:   "/prompt Tell me a joke",
	}

	deps := &domain.Dependencies{
		FilterRepository:       &mockFilterRepository{},
		MediaStorage:           newMockMediaStorage(),
		AIClient:               aiClient,
		ConversationRepository: convRepo,
		Config:                 cfg,
		Messenger:              mock,
	}

	handleGroupMessage(logger, uint32(1), uint32(10), msg, deps)

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
	mock := &mockMessenger{}
	repo := &mockFilterRepository{}
	logger := &mockLogger{}

	msg := &domain.IncomingMessage{
		ChatID: 100,
		FromID: 42,
		Text:   "Continue the thread",
		Quote: &domain.QuotedMessage{
			MessageID: 6,
		},
	}

	deps := &domain.Dependencies{
		FilterRepository:       repo,
		MediaStorage:           newMockMediaStorage(),
		AIClient:               aiClient,
		ConversationRepository: convRepo,
		Config:                 cfg,
		Messenger:              mock,
	}

	handleGroupMessage(logger, uint32(1), uint32(20), msg, deps)

	if !aiClient.called {
		t.Fatal("expected AI client to be called for thread continuation in group")
	}
	// Should NOT have tried filter matching
	if repo.matchingFilters != nil {
		t.Error("filter matching should not have been attempted")
	}
}

// --- processMessage Tests ---

// makeGroupMessenger returns a mockMessenger configured to successfully return a regular-user
// message in the given chat type, suitable for processMessage tests.
func makeGroupMessenger(msgFromID uint32, chatType domain.ChatType) *mockMessenger {
	mock := &mockMessenger{}
	mock.fetchMessageFn = func(_ uint32, _ uint32) (*domain.IncomingMessage, error) {
		return &domain.IncomingMessage{
			ID:     1,
			ChatID: 100,
			FromID: msgFromID,
			Text:   "hello",
		}, nil
	}
	mock.fetchChatTypeFn = func(_ uint32, _ uint32) (domain.ChatType, error) {
		return chatType, nil
	}
	return mock
}

func TestProcessMessage_GetMessageError(t *testing.T) {
	mock := &mockMessenger{}
	mock.fetchMessageFn = func(_ uint32, _ uint32) (*domain.IncomingMessage, error) {
		return nil, fmt.Errorf("rpc down")
	}
	logger := &mockLogger{}
	deps := &domain.Dependencies{
		FilterRepository: &mockFilterRepository{},
		MediaStorage:     newMockMediaStorage(),
		Messenger:        mock,
	}

	processMessage(logger, uint32(1), uint32(5), deps)

	if len(logger.errors) == 0 {
		t.Error("expected an error to be logged when FetchMessage fails")
	}
	if len(mock.sentTextMessages) != 0 || len(mock.sentTextReplies) != 0 || len(mock.sentMediaReplies) != 0 || len(mock.sentReactions) != 0 {
		t.Error("expected no send calls when FetchMessage fails")
	}
}

func TestProcessMessage_IgnoresSpecialContact(t *testing.T) {
	// The mock's IsSpecialContact returns true for fromID <= 9; use 9 to exercise that path.
	mock := &mockMessenger{}
	mock.fetchMessageFn = func(_ uint32, _ uint32) (*domain.IncomingMessage, error) {
		return &domain.IncomingMessage{
			ID:     1,
			ChatID: 100,
			FromID: 9, // special contact boundary
			Text:   "system message",
		}, nil
	}
	logger := &mockLogger{}
	deps := &domain.Dependencies{
		FilterRepository: &mockFilterRepository{},
		MediaStorage:     newMockMediaStorage(),
		Messenger:        mock,
	}

	processMessage(logger, uint32(1), uint32(5), deps)

	if len(logger.errors) != 0 {
		t.Errorf("expected no errors, got: %v", logger.errors)
	}
	if len(mock.sentTextMessages) != 0 || len(mock.sentTextReplies) != 0 || len(mock.sentMediaReplies) != 0 || len(mock.sentReactions) != 0 {
		t.Error("expected no send calls for special-contact message")
	}
	// FetchChatType should NOT have been called.
	// We verify indirectly: fetchChatTypeFn is nil, so if it were called it would
	// return an error which would appear in logger.errors.
}

func TestProcessMessage_GetChatInfoError(t *testing.T) {
	mock := &mockMessenger{}
	mock.fetchMessageFn = func(_ uint32, _ uint32) (*domain.IncomingMessage, error) {
		return &domain.IncomingMessage{ID: 1, ChatID: 100, FromID: 42, Text: "hi"}, nil
	}
	mock.fetchChatTypeFn = func(_ uint32, _ uint32) (domain.ChatType, error) {
		return "", fmt.Errorf("chat info unavailable")
	}
	logger := &mockLogger{}
	deps := &domain.Dependencies{
		FilterRepository: &mockFilterRepository{},
		MediaStorage:     newMockMediaStorage(),
		Messenger:        mock,
	}

	processMessage(logger, uint32(1), uint32(5), deps)

	if len(logger.errors) == 0 {
		t.Error("expected an error to be logged when FetchChatType fails")
	}
	if len(mock.sentTextMessages) != 0 || len(mock.sentTextReplies) != 0 || len(mock.sentMediaReplies) != 0 || len(mock.sentReactions) != 0 {
		t.Error("expected no send calls when FetchChatType fails")
	}
}

func TestProcessMessage_RoutesGroupChat(t *testing.T) {
	// Use a regular user FromID (> LastSpecialContactID == 9).
	mock := makeGroupMessenger(42, domain.ChatTypeGroup)
	logger := &mockLogger{}
	repo := &mockFilterRepository{
		matchingFilters: []domain.FilterResponse{
			{ResponseType: domain.ResponseTypeText, ResponseText: "hi"},
		},
	}
	deps := &domain.Dependencies{
		FilterRepository: repo,
		MediaStorage:     newMockMediaStorage(),
		Messenger:        mock,
	}

	processMessage(logger, uint32(1), uint32(5), deps)

	// handleGroupMessage was reached: it tried filter matching and sent a response.
	if len(mock.sentTextReplies) == 0 {
		t.Error("expected group handler to be invoked and send a response")
	}
	if len(logger.errors) != 0 {
		t.Errorf("unexpected errors: %v", logger.errors)
	}
}

func TestProcessMessage_RoutesSingleChat(t *testing.T) {
	mock := makeGroupMessenger(42, domain.ChatTypeSingle)
	logger := &mockLogger{}
	deps := &domain.Dependencies{
		FilterRepository: &mockFilterRepository{},
		MediaStorage:     newMockMediaStorage(),
		Messenger:        mock,
	}

	processMessage(logger, uint32(1), uint32(5), deps)

	// handleDMMessage was reached: it sends the help text via SendTextMessage.
	if len(mock.sentTextMessages) == 0 {
		t.Error("expected DM handler to be invoked and send help text")
	}
	if mock.sentTextMessages[0].text != helpText {
		t.Errorf("expected help text, got %q", mock.sentTextMessages[0].text)
	}
}

func TestProcessMessage_UnknownChatTypeWarns(t *testing.T) {
	mock := makeGroupMessenger(42, domain.ChatType("Unknown"))
	logger := &mockLogger{}
	deps := &domain.Dependencies{
		FilterRepository: &mockFilterRepository{},
		MediaStorage:     newMockMediaStorage(),
		Messenger:        mock,
	}

	processMessage(logger, uint32(1), uint32(5), deps)

	if len(logger.warns) == 0 {
		t.Error("expected a warning to be logged for unknown chat type")
	}
	if len(mock.sentTextMessages) != 0 || len(mock.sentTextReplies) != 0 || len(mock.sentMediaReplies) != 0 || len(mock.sentReactions) != 0 {
		t.Error("expected no send calls for unknown chat type")
	}
}

// --- Group Identity Tests ---

func TestHandlePromptCommand_GroupChat_WithDisplayName(t *testing.T) {
	aiClient := &mockAIClient{response: "Here's a joke!"}
	convRepo := &mockConversationRepo{}
	cfg := &mockConfig{
		systemPrompt:     "You are helpful.",
		openAIMaxHistory: 50,
	}
	mock := &mockMessenger{
		fetchContactDisplayNameFn: func(_ uint32, _ uint32) (string, error) {
			return "Mario Rossi", nil
		},
	}
	logger := &mockLogger{}

	msg := &domain.IncomingMessage{
		ChatID: 100,
		FromID: 42,
		Text:   "/prompt Tell me a joke",
	}

	deps := &domain.Dependencies{
		FilterRepository:       &mockFilterRepository{},
		MediaStorage:           newMockMediaStorage(),
		AIClient:               aiClient,
		ConversationRepository: convRepo,
		Config:                 cfg,
		Messenger:              mock,
	}

	handlePromptCommand(logger, uint32(1), uint32(10), msg, deps, domain.ChatTypeGroup)

	if !aiClient.called {
		t.Fatal("expected AI client to be called")
	}

	// System prompt must include group info
	if aiClient.lastMessages[0].Role != "system" {
		t.Errorf("expected first message to be system, got %q", aiClient.lastMessages[0].Role)
	}
	if !strings.Contains(aiClient.lastMessages[0].Content, "<general_group_chat_information>") {
		t.Error("expected system prompt to contain group information block")
	}

	// User message must be prefixed and carry Name field
	userMsg := aiClient.lastMessages[1]
	if userMsg.Role != "user" {
		t.Errorf("expected user message, got %q", userMsg.Role)
	}
	if userMsg.Name != "Mario Rossi" {
		t.Errorf("expected Name = %q, got %q", "Mario Rossi", userMsg.Name)
	}
	if userMsg.Content != "[Mario Rossi]: Tell me a joke" {
		t.Errorf("expected prefixed content, got %q", userMsg.Content)
	}

	// Saved user message must carry senderName and prefixed content
	if len(convRepo.savedMessages) != 2 {
		t.Fatalf("expected 2 saved messages, got %d", len(convRepo.savedMessages))
	}
	saved := convRepo.savedMessages[0]
	if saved.senderName != "Mario Rossi" {
		t.Errorf("expected saved senderName %q, got %q", "Mario Rossi", saved.senderName)
	}
	if saved.content != "[Mario Rossi]: Tell me a joke" {
		t.Errorf("expected saved content %q, got %q", "[Mario Rossi]: Tell me a joke", saved.content)
	}
	// Assistant message should have empty senderName
	if convRepo.savedMessages[1].senderName != "" {
		t.Errorf("expected empty senderName for assistant message, got %q", convRepo.savedMessages[1].senderName)
	}
}

func TestHandlePromptCommand_GroupChat_DisplayNameFetchError(t *testing.T) {
	aiClient := &mockAIClient{response: "OK"}
	convRepo := &mockConversationRepo{}
	cfg := &mockConfig{openAIMaxHistory: 50}
	mock := &mockMessenger{
		fetchContactDisplayNameFn: func(_ uint32, _ uint32) (string, error) {
			return "", fmt.Errorf("contact not found")
		},
	}
	logger := &mockLogger{}

	msg := &domain.IncomingMessage{
		ChatID: 100,
		FromID: 42,
		Text:   "/prompt Hello",
	}

	deps := &domain.Dependencies{
		FilterRepository:       &mockFilterRepository{},
		MediaStorage:           newMockMediaStorage(),
		AIClient:               aiClient,
		ConversationRepository: convRepo,
		Config:                 cfg,
		Messenger:              mock,
	}

	// Should not crash; falls back to no name
	handlePromptCommand(logger, uint32(1), uint32(10), msg, deps, domain.ChatTypeGroup)

	if !aiClient.called {
		t.Fatal("expected AI client to be called despite name fetch failure")
	}
	// Should have logged a warning
	if len(logger.warns) == 0 {
		t.Error("expected warning to be logged for name fetch failure")
	}
	// User message should have no Name and no prefix
	userMsg := aiClient.lastMessages[len(aiClient.lastMessages)-1]
	if userMsg.Name != "" {
		t.Errorf("expected empty Name on fallback, got %q", userMsg.Name)
	}
	if userMsg.Content != "Hello" {
		t.Errorf("expected unmodified content on fallback, got %q", userMsg.Content)
	}
}

func TestHandlePromptCommand_DM_NoNameNoPrefix(t *testing.T) {
	aiClient := &mockAIClient{response: "OK"}
	convRepo := &mockConversationRepo{}
	cfg := &mockConfig{
		systemPrompt:     "Be helpful.",
		openAIMaxHistory: 50,
	}
	mock := &mockMessenger{}
	logger := &mockLogger{}

	msg := &domain.IncomingMessage{
		ChatID: 100,
		FromID: 42,
		Text:   "/prompt Hello from DM",
	}

	deps := &domain.Dependencies{
		FilterRepository:       &mockFilterRepository{},
		MediaStorage:           newMockMediaStorage(),
		AIClient:               aiClient,
		ConversationRepository: convRepo,
		Config:                 cfg,
		Messenger:              mock,
	}

	handlePromptCommand(logger, uint32(1), uint32(10), msg, deps, domain.ChatTypeSingle)

	if !aiClient.called {
		t.Fatal("expected AI client to be called")
	}
	// System prompt must NOT have group info
	if aiClient.lastMessages[0].Role == "system" {
		if strings.Contains(aiClient.lastMessages[0].Content, "<general_group_chat_information>") {
			t.Error("DM system prompt must not contain group information block")
		}
	}
	// User message must have no Name and no prefix
	userMsg := aiClient.lastMessages[len(aiClient.lastMessages)-1]
	if userMsg.Name != "" {
		t.Errorf("expected empty Name in DM, got %q", userMsg.Name)
	}
	if userMsg.Content != "Hello from DM" {
		t.Errorf("expected unmodified content in DM, got %q", userMsg.Content)
	}
}

func TestHandleThreadContinuation_GroupChat_WithDisplayName(t *testing.T) {
	threadRoot := int64(5)
	aiClient := &mockAIClient{response: "Great response!"}
	convRepo := &mockConversationRepo{
		isConvMsgExists:     true,
		isConvMsgThreadRoot: &threadRoot,
		threadChain: []domain.ChatMessage{
			{Role: "user", Name: "Mario Rossi", Content: "[Mario Rossi]: What is Go?"},
			{Role: "assistant", Content: "Go is a programming language."},
		},
	}
	cfg := &mockConfig{
		systemPrompt:     "Be helpful.",
		openAIMaxHistory: 50,
	}
	mock := &mockMessenger{
		fetchContactDisplayNameFn: func(_ uint32, _ uint32) (string, error) {
			return "Luigi Verdi", nil
		},
	}
	logger := &mockLogger{}

	msg := &domain.IncomingMessage{
		ChatID: 100,
		FromID: 99,
		Text:   "Tell me more",
		Quote: &domain.QuotedMessage{MessageID: 6},
	}

	deps := &domain.Dependencies{
		FilterRepository:       &mockFilterRepository{},
		MediaStorage:           newMockMediaStorage(),
		AIClient:               aiClient,
		ConversationRepository: convRepo,
		Config:                 cfg,
		Messenger:              mock,
	}

	handleThreadContinuation(logger, uint32(1), uint32(20), msg, deps, threadRoot, domain.ChatTypeGroup)

	if !aiClient.called {
		t.Fatal("expected AI client to be called")
	}

	// Verify system prompt has group info
	if aiClient.lastMessages[0].Role != "system" {
		t.Errorf("expected system message first, got %q", aiClient.lastMessages[0].Role)
	}
	if !strings.Contains(aiClient.lastMessages[0].Content, "<general_group_chat_information>") {
		t.Error("expected group information in system prompt")
	}

	// Historical messages from chain should be passed through unchanged
	if aiClient.lastMessages[1].Name != "Mario Rossi" {
		t.Errorf("expected historical message to carry Name %q, got %q", "Mario Rossi", aiClient.lastMessages[1].Name)
	}

	// New user message should have Luigi's name and prefixed content
	newMsg := aiClient.lastMessages[len(aiClient.lastMessages)-1]
	if newMsg.Name != "Luigi Verdi" {
		t.Errorf("expected Name = %q, got %q", "Luigi Verdi", newMsg.Name)
	}
	if newMsg.Content != "[Luigi Verdi]: Tell me more" {
		t.Errorf("expected prefixed content, got %q", newMsg.Content)
	}

	// Verify saved messages
	if len(convRepo.savedMessages) != 2 {
		t.Fatalf("expected 2 saved messages, got %d", len(convRepo.savedMessages))
	}
	if convRepo.savedMessages[0].senderName != "Luigi Verdi" {
		t.Errorf("expected senderName %q saved, got %q", "Luigi Verdi", convRepo.savedMessages[0].senderName)
	}
	if convRepo.savedMessages[1].senderName != "" {
		t.Errorf("expected empty senderName for assistant, got %q", convRepo.savedMessages[1].senderName)
	}
}

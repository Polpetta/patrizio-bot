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

	handleFilterCommand(rpc, logger, deltachat.AccountId(1), msg, deps)

	if !repo.createTextFilterCalled {
		t.Fatal("expected CreateTextFilter to be called")
	}
	if repo.lastResponse != "Hi there!" {
		t.Errorf("expected response 'Hi there!', got %q", repo.lastResponse)
	}
	if len(repo.lastTriggers) != 1 || repo.lastTriggers[0] != domain.NormalizeTrigger("hello") {
		t.Errorf("unexpected triggers: %v", repo.lastTriggers)
	}

	// Should have sent a confirmation
	if len(rpc.sentMessages) != 1 {
		t.Fatalf("expected 1 sent message (confirmation), got %d", len(rpc.sentMessages))
	}
	if rpc.sentMessages[0].chatID != 100 {
		t.Errorf("expected confirmation sent to chat 100, got %d", rpc.sentMessages[0].chatID)
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

	handleFilterCommand(rpc, logger, deltachat.AccountId(1), msg, deps)

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
	expectedHash := computeSHA512(mediaContent)

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

	handleFilterCommand(rpc, logger, deltachat.AccountId(1), msg, deps)

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

	// Should have sent a confirmation
	if len(rpc.sentMessages) != 1 {
		t.Fatalf("expected 1 sent message (confirmation), got %d", len(rpc.sentMessages))
	}
}

// TestHandleFilterCommand_MediaFromAttachment_MultipleTriggers tests media filter with
// multiple triggers and an attached image.
func TestHandleFilterCommand_MediaFromAttachment_MultipleTriggers(t *testing.T) {
	mediaContent := []byte("another-fake-image")
	mediaPath := writeTempFile(t, mediaContent)
	expectedHash := computeSHA512(mediaContent)

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

	handleFilterCommand(rpc, logger, deltachat.AccountId(1), msg, deps)

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
	expectedHash := computeSHA512(mediaContent)

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

	handleFilterCommand(rpc, logger, deltachat.AccountId(1), msg, deps)

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
	expectedHash := computeSHA512(mediaContent)

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

	handleFilterCommand(rpc, logger, deltachat.AccountId(1), msg, deps)

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

	handleFilterCommand(rpc, logger, deltachat.AccountId(1), msg, deps)

	if repo.createMediaFilterCalled || repo.createTextFilterCalled {
		t.Fatal("no filter should have been created")
	}

	// Should have sent an error message about needing media
	if len(rpc.sentMessages) != 1 {
		t.Fatalf("expected 1 error message, got %d", len(rpc.sentMessages))
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

	handleDMMessage(rpc, logger, deltachat.AccountId(1), msg)

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

	if len(rpc.sentMessages) != 1 {
		t.Fatalf("expected 1 sent message, got %d", len(rpc.sentMessages))
	}
	if rpc.sentMessages[0].text != "I love puppies!" {
		t.Errorf("expected 'I love puppies!', got %q", rpc.sentMessages[0].text)
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

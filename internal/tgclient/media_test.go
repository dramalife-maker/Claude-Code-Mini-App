package tgclient

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/gotd/td/tg"
	"github.com/jerry12122/Claude-Code-Mini-App/internal/config"
	"github.com/jerry12122/Claude-Code-Mini-App/internal/db"
	"github.com/jerry12122/Claude-Code-Mini-App/internal/media"
)

func TestTelegramMediaPassThrough(t *testing.T) {
	ctx := context.Background()
	old := media.WorkspaceDir
	media.WorkspaceDir = t.TempDir()
	t.Cleanup(func() { media.WorkspaceDir = old })

	database, err := db.Open(t.TempDir() + "/media.db")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	alice := newTGSession(t, database, "alice", "@alice")
	mgr := NewManager(config.TelegramClient{Enabled: true}, database)
	mgr.setAccount(1)
	mgr.authState = AuthStateAuthorized
	mgr.recordUsers([]tg.UserClass{&tg.User{ID: 101, AccessHash: 555, Username: "alice"}})

	photo := &tg.Message{
		ID: 70, PeerID: &tg.PeerUser{UserID: 101}, Date: tgTestDate,
		Media: &tg.MessageMediaPhoto{Photo: &tg.Photo{
			ID: 9, AccessHash: 8, FileReference: []byte{1},
			Sizes: []tg.PhotoSizeClass{&tg.PhotoSize{Type: "y", W: 10, H: 10, Size: 4}},
		}},
	}
	mgr.historyFn = func(_ context.Context, req *tg.MessagesGetHistoryRequest) (tg.MessagesMessagesClass, error) {
		if req.OffsetID != 71 || req.Limit != 1 {
			t.Fatalf("應只取這一則，offset=%d limit=%d", req.OffsetID, req.Limit)
		}
		return &tg.MessagesMessages{Messages: []tg.MessageClass{photo}}, nil
	}
	mgr.fileFn = func(_ context.Context, loc tg.InputFileLocationClass, w io.Writer) error {
		p, ok := loc.(*tg.InputPhotoFileLocation)
		if !ok || p.ID != 9 || p.ThumbSize != "y" {
			t.Fatalf("location = %#v", loc)
		}
		_, err := w.Write([]byte("JPEG"))
		return err
	}
	name, mimeType, body, err := mgr.OpenMedia(ctx, alice, 70)
	if err != nil {
		t.Fatal(err)
	}
	defer body.Close()
	got, _ := io.ReadAll(body)
	if name != "photo.jpg" || mimeType != "image/jpeg" || string(got) != "JPEG" {
		t.Fatalf("name=%s mime=%s body=%q", name, mimeType, got)
	}

	// 歷史 DTO 帶附件網址，不把位元組放進訊息。
	mgr.historyFn = func(_ context.Context, req *tg.MessagesGetHistoryRequest) (tg.MessagesMessagesClass, error) {
		if req.OffsetID != 0 {
			t.Fatalf("最新頁 offset=%d", req.OffsetID)
		}
		return &tg.MessagesMessages{Messages: []tg.MessageClass{photo}}, nil
	}
	page, err := mgr.HistoryPage(ctx, alice, 1, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Messages) != 1 || len(page.Messages[0].Attachments) != 1 {
		t.Fatalf("應有一張圖片卡片，got %+v", page.Messages)
	}
	if page.Messages[0].Content != "" || page.Messages[0].Attachments[0].MimeType != "image/jpeg" {
		t.Fatalf("純圖片不該是佔位文字，got %+v", page.Messages[0])
	}

	path, err := media.SaveUpload(alice, "note.txt", bytes.NewReader([]byte("hello")))
	if err != nil {
		t.Fatal(err)
	}
	_ = path
	att := &db.Attachment{ID: "a1", SessionID: alice, Name: "note.txt", MimeType: "text/plain", Size: 5}
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			att.StorageName = path[i+1:]
			break
		}
	}
	if err := database.AddAttachment(att); err != nil {
		t.Fatal(err)
	}
	var sent *tg.MessagesSendMediaRequest
	mgr.uploadFn = func(_ context.Context, name string, r io.Reader) (tg.InputFileClass, error) {
		b, _ := io.ReadAll(r)
		if name != "note.txt" || string(b) != "hello" {
			t.Fatalf("upload name=%s body=%q", name, b)
		}
		return &tg.InputFile{ID: 1, Name: name}, nil
	}
	mgr.sendMediaFn = func(_ context.Context, req *tg.MessagesSendMediaRequest) (tg.UpdatesClass, error) {
		sent = req
		return &tg.UpdateShortSentMessage{ID: 80, Date: tgTestDate}, nil
	}
	if st, err := mgr.sendTelegramMedia(ctx, alice, 5, "說明", []string{"a1"}); err != nil || st != "sent" {
		t.Fatalf("state=%s err=%v", st, err)
	}
	if _, ok := sent.Media.(*tg.InputMediaUploadedDocument); !ok || sent.Message != "說明" {
		t.Fatalf("一般檔應為 document，caption 是說明不是路徑，got %#v", sent)
	}
}

package tgclient

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/gotd/td/telegram/downloader"
	"github.com/gotd/td/telegram/uploader"
	"github.com/gotd/td/tg"
	"github.com/gotd/td/tgerr"
	"github.com/jerry12122/Claude-Code-Mini-App/internal/db"
	"github.com/jerry12122/Claude-Code-Mini-App/internal/media"
)

// 媒體上傳／下載的注入點。測試不碰真實 Telegram，也不把檔案寫進磁碟快取。
type (
	uploadFunc    func(ctx context.Context, name string, r io.Reader, size int64) (tg.InputFileClass, error)
	sendMediaFunc func(ctx context.Context, req *tg.MessagesSendMediaRequest) (tg.UpdatesClass, error)
	fileFunc      func(ctx context.Context, loc tg.InputFileLocationClass, w io.Writer) error
)

// rpcUpload 必須帶真實大小：FromReader 傳 -1 會被當成大檔案（InputFileBig），Telegram 不接受這種照片（PHOTO_SAVE_FILE_INVALID）。
func (m *Manager) rpcUpload(ctx context.Context, name string, r io.Reader, size int64) (tg.InputFileClass, error) {
	if m.client == nil {
		return nil, errTGNotConnected
	}
	return uploader.NewUploader(m.client.API()).Upload(ctx, uploader.NewUpload(name, r, size))
}

func (m *Manager) rpcSendMedia(ctx context.Context, req *tg.MessagesSendMediaRequest) (tg.UpdatesClass, error) {
	if m.client == nil {
		return nil, errTGNotConnected
	}
	return m.client.API().MessagesSendMedia(ctx, req)
}

func (m *Manager) rpcSaveFile(ctx context.Context, loc tg.InputFileLocationClass, w io.Writer) error {
	if m.client == nil {
		return errTGNotConnected
	}
	_, err := downloader.NewDownloader().Download(m.client.API(), loc).Stream(ctx, w)
	return err
}

func tgMediaURL(sessionID string, msgID int64) string {
	return "/sessions/" + url.PathEscape(sessionID) + "/tg-media/" + strconv.FormatInt(msgID, 10)
}

// fillMedia 把照片／檔案放進 DTO。沒有 caption 時不留佔位文字，卡片本身就是內容。
func fillMedia(sessionID string, src *tg.Message, dst *db.Message) {
	atts := telegramAttachments(src)
	if len(atts) == 0 {
		return
	}
	for i := range atts {
		atts[i].URL = tgMediaURL(sessionID, int64(src.ID))
	}
	dst.Attachments = atts
	if strings.TrimSpace(src.Message) == "" {
		dst.Content = ""
	}
}

func telegramAttachments(src *tg.Message) []db.Attachment {
	if src == nil || src.Media == nil {
		return nil
	}
	id := strconv.Itoa(src.ID)
	switch med := src.Media.(type) {
	case *tg.MessageMediaPhoto:
		photo, ok := med.Photo.(*tg.Photo)
		if !ok {
			return nil
		}
		return []db.Attachment{{ID: id, Name: "photo.jpg", MimeType: "image/jpeg", Size: int64(photoSize(photo))}}
	case *tg.MessageMediaDocument:
		doc, ok := med.Document.(*tg.Document)
		if !ok {
			return nil
		}
		name := "file"
		for _, attr := range doc.Attributes {
			if fn, ok := attr.(*tg.DocumentAttributeFilename); ok && fn.FileName != "" {
				name = fn.FileName
			}
		}
		mimeType := doc.MimeType
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}
		return []db.Attachment{{ID: id, Name: name, MimeType: mimeType, Size: doc.Size}}
	default:
		return nil
	}
}

func photoSize(p *tg.Photo) int {
	best := 0
	for _, s := range p.Sizes {
		if ps, ok := s.(*tg.PhotoSize); ok && ps.Size > best {
			best = ps.Size
		}
	}
	return best
}

// OpenMedia 依目前綁定的私聊讀取單一訊息的檔案，直接寫進 w。不落地。
func (m *Manager) OpenMedia(ctx context.Context, sessionID string, msgID int) (name, mimeType string, r io.ReadCloser, err error) {
	// 失敗原因只在這裡看得到：HTTP 端一律回 503，沒有日誌就無法分辨是解析對象、查訊息還是下載失敗。
	unavailable := func(step string, cause error) (string, string, io.ReadCloser, error) {
		slog.Warn(fmt.Sprintf("[telegram] 讀取媒體失敗（%s）msg=%d: %v", step, msgID, cause))
		return "", "", nil, ErrHistoryUnavailable
	}
	if msgID <= 0 || msgID > 1<<31-1 {
		return "", "", nil, ErrHistoryInvalidCursor
	}
	if m.db == nil {
		return unavailable("資料庫未就緒", nil)
	}
	s, err := m.db.GetSession(sessionID)
	if err != nil {
		return unavailable("讀取會話", err)
	}
	peer, err := m.liveHistoryPeer(ctx, s)
	if err != nil {
		slog.Warn(fmt.Sprintf("[telegram] 讀取媒體失敗（解析對象）msg=%d: %v", msgID, err))
		return "", "", nil, err
	}
	if m.historyFn == nil || m.fileFn == nil {
		return unavailable("上游函式未接線", nil)
	}
	resp, err := m.historyFn(ctx, &tg.MessagesGetHistoryRequest{
		Peer:     &tg.InputPeerUser{UserID: peer.peerID, AccessHash: peer.accessHash},
		OffsetID: msgID + 1,
		Limit:    1,
	})
	if err != nil {
		if d, ok := tgerr.AsFloodWait(err); ok {
			return "", "", nil, &HistoryRateLimitedError{RetryAfter: d}
		}
		slog.Warn(fmt.Sprintf("[telegram] 讀取媒體失敗（查詢訊息）msg=%d: %v", msgID, err))
		return "", "", nil, fmt.Errorf("%w: %v", ErrHistoryUnavailable, err)
	}
	if m.currentAccount() != peer.account {
		return unavailable("請求途中帳號改變", nil)
	}
	msg := messageByID(resp, msgID)
	if msg == nil {
		return unavailable("上游沒回這則訊息", fmt.Errorf("回應 %T", resp))
	}
	loc, name, mimeType := mediaLocation(msg)
	if loc == nil {
		return unavailable("訊息沒有可下載的媒體", fmt.Errorf("media=%T", msg.Media))
	}
	pr, pw := io.Pipe()
	go func() {
		err := m.fileFn(ctx, loc, pw)
		if err != nil {
			slog.Warn(fmt.Sprintf("[telegram] 讀取媒體失敗（下載 %T）msg=%d: %v", loc, msgID, err))
		}
		pw.CloseWithError(err)
	}()
	return name, mimeType, pr, nil
}

func messageByID(resp tg.MessagesMessagesClass, msgID int) *tg.Message {
	var raw []tg.MessageClass
	switch r := resp.(type) {
	case *tg.MessagesMessages:
		raw = r.Messages
	case *tg.MessagesMessagesSlice:
		raw = r.Messages
	}
	for _, mc := range raw {
		if msg, ok := mc.(*tg.Message); ok && msg.ID == msgID {
			return msg
		}
	}
	return nil
}

func mediaLocation(msg *tg.Message) (tg.InputFileLocationClass, string, string) {
	switch med := msg.Media.(type) {
	case *tg.MessageMediaPhoto:
		photo, ok := med.Photo.(*tg.Photo)
		if !ok {
			return nil, "", ""
		}
		thumb := ""
		for _, s := range photo.Sizes {
			if ps, ok := s.(*tg.PhotoSize); ok {
				thumb = ps.Type
			}
		}
		return &tg.InputPhotoFileLocation{
			ID: photo.ID, AccessHash: photo.AccessHash, FileReference: photo.FileReference, ThumbSize: thumb,
		}, "photo.jpg", "image/jpeg"
	case *tg.MessageMediaDocument:
		doc, ok := med.Document.(*tg.Document)
		if !ok {
			return nil, "", ""
		}
		name := "file"
		for _, attr := range doc.Attributes {
			if fn, ok := attr.(*tg.DocumentAttributeFilename); ok && fn.FileName != "" {
				name = fn.FileName
			}
		}
		mimeType := doc.MimeType
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}
		return &tg.InputDocumentFileLocation{ID: doc.ID, AccessHash: doc.AccessHash, FileReference: doc.FileReference}, name, mimeType
	default:
		return nil, "", ""
	}
}

// sendTelegramMedia 用既有上傳檔送出。圖片走 photo，其餘走 document。不把伺服器路徑送進訊息文字。
func (m *Manager) sendTelegramMedia(ctx context.Context, sessionID string, clientID int64, caption string, ids []string) (string, error) {
	fail := func(id int64, step string, err error) (string, error) {
		slog.Warn(fmt.Sprintf("[telegram] 送出附件失敗（%s）: %v", step, err))
		if id > 0 {
			publishDelivery(sessionID, strconv.FormatInt(id, 10), "failed", "訊息傳送失敗，請重新發送")
		}
		return "failed", nil
	}
	if m.db == nil {
		return "failed", nil
	}
	s, err := m.db.GetSession(sessionID)
	if err != nil {
		return fail(clientID, "讀取會話", err)
	}
	peer, err := m.liveHistoryPeer(ctx, s)
	if err != nil || m.uploadFn == nil || m.sendMediaFn == nil {
		return fail(clientID, "解析對象或上傳未接線", err)
	}
	for i, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		att, err := m.db.GetAttachment(sessionID, id)
		if err != nil {
			return fail(clientID, "讀取附件", err)
		}
		path, err := media.ExistingUploadPath(att.SessionID, att.StorageName)
		if err != nil {
			return fail(clientID, "附件路徑", err)
		}
		f, err := os.Open(path)
		if err != nil {
			return fail(clientID, "開啟檔案", err)
		}
		st, err := f.Stat()
		if err != nil {
			f.Close()
			return fail(clientID, "讀取檔案大小", err)
		}
		input, err := m.uploadFn(ctx, att.Name, f, st.Size())
		f.Close()
		if err != nil {
			return fail(clientID, "上傳檔案", err)
		}
		text := ""
		if i == 0 {
			text = caption
		}
		var mediaInput tg.InputMediaClass
		if isTelegramPhoto(att.MimeType) {
			mediaInput = &tg.InputMediaUploadedPhoto{File: input}
		} else {
			mediaInput = &tg.InputMediaUploadedDocument{
				File: input,
				Attributes: []tg.DocumentAttributeClass{
					&tg.DocumentAttributeFilename{FileName: att.Name},
				},
			}
		}
		randomID, err := newRandomID()
		if err != nil {
			return "", err
		}
		replaces := int64(0)
		if i == 0 {
			replaces = clientID
		}
		m.mu.Lock()
		m.pending[randomID] = memSend{
			sessionID: sessionID, clientID: replaces, account: peer.account,
			peerID: peer.peerID, access: peer.accessHash, text: text,
		}
		m.mu.Unlock()
		result, err := m.sendMediaFn(ctx, &tg.MessagesSendMediaRequest{
			Peer:     &tg.InputPeerUser{UserID: peer.peerID, AccessHash: peer.accessHash},
			Media:    mediaInput,
			Message:  text,
			RandomID: randomID,
		})
		if err != nil {
			m.dropPending(randomID)
			return fail(replaces, "sendMedia", err)
		}
		if short, ok := result.(*tg.UpdateShortSentMessage); ok {
			m.finishSend(randomID, int64(short.ID), int64(short.Date))
			continue
		}
		_ = m.handleUpdate(ctx, result)
		m.mu.Lock()
		_, still := m.pending[randomID]
		m.mu.Unlock()
		if still {
			m.dropPending(randomID)
			return fail(replaces, "已送出但未收到送達確認", nil)
		}
	}
	return "sent", nil
}

func isTelegramPhoto(mimeType string) bool {
	switch strings.ToLower(mimeType) {
	case "image/jpeg", "image/png", "image/webp":
		return true
	default:
		return false
	}
}

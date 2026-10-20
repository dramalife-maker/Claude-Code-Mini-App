package tgclient

import (
	"context"
	"fmt"
	"io"
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
	uploadFunc    func(ctx context.Context, name string, r io.Reader) (tg.InputFileClass, error)
	sendMediaFunc func(ctx context.Context, req *tg.MessagesSendMediaRequest) (tg.UpdatesClass, error)
	fileFunc      func(ctx context.Context, loc tg.InputFileLocationClass, w io.Writer) error
)

func (m *Manager) rpcUpload(ctx context.Context, name string, r io.Reader) (tg.InputFileClass, error) {
	if m.client == nil {
		return nil, errTGNotConnected
	}
	return uploader.NewUploader(m.client.API()).FromReader(ctx, name, r)
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
	if msgID <= 0 || msgID > 1<<31-1 {
		return "", "", nil, ErrHistoryInvalidCursor
	}
	if m.db == nil {
		return "", "", nil, ErrHistoryUnavailable
	}
	s, err := m.db.GetSession(sessionID)
	if err != nil {
		return "", "", nil, ErrHistoryUnavailable
	}
	peer, err := m.liveHistoryPeer(ctx, s)
	if err != nil {
		return "", "", nil, err
	}
	if m.historyFn == nil || m.fileFn == nil {
		return "", "", nil, ErrHistoryUnavailable
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
		return "", "", nil, fmt.Errorf("%w: %v", ErrHistoryUnavailable, err)
	}
	if m.currentAccount() != peer.account {
		return "", "", nil, fmt.Errorf("%w: account changed during request", ErrHistoryUnavailable)
	}
	msg := messageByID(resp, msgID)
	if msg == nil {
		return "", "", nil, ErrHistoryUnavailable
	}
	loc, name, mimeType := mediaLocation(msg)
	if loc == nil {
		return "", "", nil, ErrHistoryUnavailable
	}
	pr, pw := io.Pipe()
	go func() {
		pw.CloseWithError(m.fileFn(ctx, loc, pw))
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
	if m.db == nil {
		return "failed", nil
	}
	s, err := m.db.GetSession(sessionID)
	if err != nil {
		publishDelivery(sessionID, strconv.FormatInt(clientID, 10), "failed", "訊息傳送失敗，請重新發送")
		return "failed", nil
	}
	peer, err := m.liveHistoryPeer(ctx, s)
	if err != nil || m.uploadFn == nil || m.sendMediaFn == nil {
		publishDelivery(sessionID, strconv.FormatInt(clientID, 10), "failed", "訊息傳送失敗，請重新發送")
		return "failed", nil
	}
	for i, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		att, err := m.db.GetAttachment(sessionID, id)
		if err != nil {
			publishDelivery(sessionID, strconv.FormatInt(clientID, 10), "failed", "訊息傳送失敗，請重新發送")
			return "failed", nil
		}
		path, err := media.ExistingUploadPath(att.SessionID, att.StorageName)
		if err != nil {
			publishDelivery(sessionID, strconv.FormatInt(clientID, 10), "failed", "訊息傳送失敗，請重新發送")
			return "failed", nil
		}
		f, err := os.Open(path)
		if err != nil {
			publishDelivery(sessionID, strconv.FormatInt(clientID, 10), "failed", "訊息傳送失敗，請重新發送")
			return "failed", nil
		}
		input, err := m.uploadFn(ctx, att.Name, f)
		f.Close()
		if err != nil {
			publishDelivery(sessionID, strconv.FormatInt(clientID, 10), "failed", "訊息傳送失敗，請重新發送")
			return "failed", nil
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
			if replaces > 0 {
				publishDelivery(sessionID, strconv.FormatInt(replaces, 10), "failed", "訊息傳送失敗，請重新發送")
			}
			return "failed", nil
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
			if replaces > 0 {
				publishDelivery(sessionID, strconv.FormatInt(replaces, 10), "failed", "訊息傳送失敗，請重新發送")
			}
			return "failed", nil
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

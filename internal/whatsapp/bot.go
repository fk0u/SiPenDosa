package whatsapp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

// BotReplyHandler handles incoming messages and provides automated academic responses
func (c *Client) handleIncomingMessage(evt *events.Message) {
	if evt == nil || evt.Info.IsFromMe {
		return
	}

	senderJID := evt.Info.Sender.ToNonAD().String()
	senderName := evt.Info.PushName
	if senderName == "" {
		senderName = senderJID
	}

	// Extract incoming message text or interactive response button
	rawText := extractMessageContent(evt.Message)
	if rawText == "" {
		return
	}

	c.Touch()
	cleaned := strings.ToLower(strings.TrimSpace(rawText))
	slog.Info("Incoming WhatsApp message received", "sender", senderJID, "name", senderName, "content", cleaned)

	var replyText string
	var choiceLabel string

	switch {
	case cleaned == "btn_hadir_offline" || cleaned == "1" || strings.Contains(cleaned, "hadir di kelas") || cleaned == "hadir":
		choiceLabel = "Hadir di Kelas"
		replyText = "✅ *Konfirmasi Kehadiran Diterima*\n\nTerima kasih Bapak/Ibu. Kehadiran tatap muka di kelas telah dicatat di sistem SiPenDosa. Mahasiswa siap mengikuti perkuliahan.\n\n_Selamat mengajar!_"

	case cleaned == "btn_hadir_online" || cleaned == "2" || strings.Contains(cleaned, "online") || strings.Contains(cleaned, "zoom") || strings.Contains(cleaned, "meet") || strings.Contains(cleaned, "daring"):
		choiceLabel = "Kuliah Online (Daring)"
		replyText = "💻 *Konfirmasi Kuliah Online Diterima*\n\nTerima kasih Bapak/Ibu. Perkuliahan daring telah dicatat di SiPenDosa. Mahasiswa akan bersiap di tautan kelas/ruang virtual.\n\n_Terima kasih atas informasinya._"

	case cleaned == "btn_reschedule" || cleaned == "3" || strings.Contains(cleaned, "reschedule") || strings.Contains(cleaned, "ganti") || strings.Contains(cleaned, "izin") || strings.Contains(cleaned, "batal"):
		choiceLabel = "Minta Reschedule"
		replyText = "🔄 *Permintaan Reschedule Diterima*\n\nBaik Bapak/Ibu. Permintaan perubahan jadwal perkuliahan telah kami teruskan ke Ketua Tingkat / PJ Mata Kuliah untuk koordinasi jadwal pengganti.\n\n_PJ akan segera menghubungi Bapak/Ibu._"

	case cleaned == "menu" || cleaned == "bantuan" || cleaned == "help" || cleaned == "info":
		replyText = "🤖 *SiPenDosa Academic Assistant Bot*\n\nBot ini mengirimkan pengingat jadwal perkuliahan secara otomatis. Untuk konfirmasi jadwal mengajar, Anda dapat membalas:\n• *1* atau *Hadir* : Hadir di kelas\n• *2* atau *Online* : Kuliah daring (Zoom/Meet)\n• *3* atau *Reschedule* : Minta ganti jadwal\n\n_SiPenDosa — Sistem Pengingat Dosen Saatnya_"

	default:
		// Do not auto-reply to random conversational messages to prevent spam loops
		return
	}

	// Broadcast live toast to Web dashboard via WebSocket
	if c.hub != nil {
		c.hub.BroadcastToast("info", fmt.Sprintf("💬 Respon Dosen (%s): %s", senderName, choiceLabel))
		c.hub.Broadcast("bot_reply", map[string]interface{}{
			"sender": senderJID,
			"name":   senderName,
			"choice": choiceLabel,
			"time":   time.Now().Format("15:04:05"),
		})
	}

	// Send acknowledgment reply back to WhatsApp
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		jid, err := FormatJID(senderJID)
		if err != nil {
			return
		}

		replyMsg := &waE2E.Message{
			Conversation: proto.String(replyText),
		}

		c.mu.RLock()
		client := c.client
		c.mu.RUnlock()

		if client != nil && client.IsConnected() {
			_, err = client.SendMessage(ctx, jid, replyMsg)
			if err != nil {
				slog.Warn("Failed sending bot reply", "to", senderJID, "err", err)
			}
		}
	}()
}

// extractMessageContent extracts text or button selection from various WhatsApp message structures
func extractMessageContent(msg *waE2E.Message) string {
	if msg == nil {
		return ""
	}

	// 1. Plain text conversation
	if text := msg.GetConversation(); text != "" {
		return text
	}

	// 2. Extended text (replies, links)
	if ext := msg.GetExtendedTextMessage(); ext != nil && ext.GetText() != "" {
		return ext.GetText()
	}

	// 3. Interactive Response (NativeFlow Buttons)
	if inter := msg.GetInteractiveResponseMessage(); inter != nil {
		if native := inter.GetNativeFlowResponseMessage(); native != nil {
			params := native.GetParamsJSON()
			if params != "" {
				var data struct {
					ID string `json:"id"`
				}
				if json.Unmarshal([]byte(params), &data) == nil && data.ID != "" {
					return data.ID
				}
				return params
			}
		}
	}

	// 4. Buttons Response Message
	if btn := msg.GetButtonsResponseMessage(); btn != nil {
		if id := btn.GetSelectedButtonID(); id != "" {
			return id
		}
		return btn.GetSelectedDisplayText()
	}

	// 5. Template Button Reply Message
	if tmpl := msg.GetTemplateButtonReplyMessage(); tmpl != nil {
		return tmpl.GetSelectedID()
	}

	// 6. List Response Message
	if list := msg.GetListResponseMessage(); list != nil && list.GetSingleSelectReply() != nil {
		return list.GetSingleSelectReply().GetSelectedRowID()
	}

	return ""
}

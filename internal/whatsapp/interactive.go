package whatsapp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

// InteractiveButton defines an action button for WhatsApp messages
type InteractiveButton struct {
	ID   string `json:"id"`
	Text string `json:"text"`
	URL  string `json:"url,omitempty"` // If set, renders as a CTA URL button
}

// DefaultReminderButtons returns standard academic attendance confirmation buttons
func DefaultReminderButtons(schedulePortalURL string) []InteractiveButton {
	buttons := []InteractiveButton{
		{ID: "btn_hadir_offline", Text: "✅ Hadir di Kelas"},
		{ID: "btn_hadir_online", Text: "💻 Hadir Online"},
		{ID: "btn_reschedule", Text: "🔄 Minta Reschedule"},
	}
	if schedulePortalURL != "" {
		buttons = append(buttons, InteractiveButton{
			ID:   "btn_portal",
			Text: "🌐 Buka Papan Jadwal",
			URL:  schedulePortalURL,
		})
	}
	return buttons
}

// SendInteractiveMessage sends a rich interactive WhatsApp message with NativeFlow buttons
// and includes a structured fallback text menu for clients that do not render native buttons.
func (c *Client) SendInteractiveMessage(ctx context.Context, to string, title string, body string, footer string, buttons []InteractiveButton) error {
	c.Touch()
	if err := c.EnsureConnected(ctx); err != nil {
		return err
	}

	c.mu.RLock()
	client := c.client
	c.mu.RUnlock()

	jid, err := FormatJID(to)
	if err != nil {
		return fmt.Errorf("invalid recipient JID (%s): %w", to, err)
	}

	// 1. Presence simulation (anti-ban human-like behavior)
	_ = client.SendPresence(ctx, types.PresenceAvailable)
	_ = client.SendChatPresence(ctx, jid, types.ChatPresenceComposing, types.ChatPresenceMediaText)

	jitterMs := 1200 + rand.Intn(1000)
	select {
	case <-time.After(time.Duration(jitterMs) * time.Millisecond):
	case <-ctx.Done():
		return ctx.Err()
	}

	// 2. Build structured fallback text menu
	fallbackText := body
	if len(buttons) > 0 {
		var menuBuilder strings.Builder
		menuBuilder.WriteString("\n\n────────────────────\n💬 *Balas pesan ini untuk konfirmasi cepat:*\n")
		num := 1
		for _, btn := range buttons {
			if btn.URL != "" {
				menuBuilder.WriteString(fmt.Sprintf("🔗 *%s*: %s\n", btn.Text, btn.URL))
			} else {
				menuBuilder.WriteString(fmt.Sprintf("%d️⃣ *%s* (balas %d atau ketik \"%s\")\n", num, btn.Text, num, strings.TrimPrefix(btn.Text, "✅ ")))
				num++
			}
		}
		fallbackText += menuBuilder.String()
	}

	if footer == "" {
		footer = "SiPenDosa Academic Assistant Bot"
	}

	// 3. Assemble NativeFlow buttons
	var nativeButtons []*waE2E.InteractiveMessage_NativeFlowMessage_NativeFlowButton
	for _, b := range buttons {
		if b.URL != "" {
			paramJSON, _ := json.Marshal(map[string]string{
				"display_text": b.Text,
				"url":          b.URL,
			})
			nativeButtons = append(nativeButtons, &waE2E.InteractiveMessage_NativeFlowMessage_NativeFlowButton{
				Name:             proto.String("cta_url"),
				ButtonParamsJSON: proto.String(string(paramJSON)),
			})
		} else {
			paramJSON, _ := json.Marshal(map[string]string{
				"display_text": b.Text,
				"id":           b.ID,
			})
			nativeButtons = append(nativeButtons, &waE2E.InteractiveMessage_NativeFlowMessage_NativeFlowButton{
				Name:             proto.String("quick_reply"),
				ButtonParamsJSON: proto.String(string(paramJSON)),
			})
		}
	}

	// 4. Construct Protobuf message with ViewOnce wrapper for maximum cross-client compatibility
	headerTitle := title
	if headerTitle == "" {
		headerTitle = "📚 PENGINGAT PERKULIAHAN"
	}

	interactiveMsg := &waE2E.InteractiveMessage{
		Header: &waE2E.InteractiveMessage_Header{
			Title:              proto.String(headerTitle),
			HasMediaAttachment: proto.Bool(false),
		},
		Body: &waE2E.InteractiveMessage_Body{
			Text: proto.String(fallbackText),
		},
		Footer: &waE2E.InteractiveMessage_Footer{
			Text: proto.String(footer),
		},
		InteractiveMessage: &waE2E.InteractiveMessage_NativeFlowMessage_{
			NativeFlowMessage: &waE2E.InteractiveMessage_NativeFlowMessage{
				Buttons: nativeButtons,
			},
		},
	}

	msg := &waE2E.Message{
		ViewOnceMessage: &waE2E.FutureProofMessage{
			Message: &waE2E.Message{
				InteractiveMessage: interactiveMsg,
			},
		},
	}

	resp, err := client.SendMessage(ctx, jid, msg)
	_ = client.SendChatPresence(ctx, jid, types.ChatPresencePaused, types.ChatPresenceMediaText)

	if err != nil {
		// Fallback to plain conversation message if interactive proto is rejected
		slog.Warn("Interactive message failed, falling back to rich text message", "to", jid.String(), "err", err)
		plainMsg := &waE2E.Message{
			Conversation: proto.String(fallbackText),
		}
		resp, err = client.SendMessage(ctx, jid, plainMsg)
		if err != nil {
			return fmt.Errorf("failed delivering message: %w", err)
		}
	}

	c.Touch()
	slog.Info("Interactive WhatsApp message delivered", "to", jid.String(), "msg_id", resp.ID)
	return nil
}

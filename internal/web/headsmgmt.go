package web

import (
	"fmt"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/starfederation/datastar-go/datastar"

	"github.com/alexradunet/balaur/internal/conversation"
	"github.com/alexradunet/balaur/internal/store"
	"github.com/alexradunet/balaur/internal/turn"
)

// headView is the template payload for one head record on the /heads page.
type headView struct {
	ID            string
	Name          string
	Purpose       string
	Status        string
	Expires       string // human-readable; empty when not set
	AvatarURL     string
	BalaurOptions []AvatarOption
}

type headsData struct {
	Title     string
	MainClass string
	Dock      homeData
	Heads     []headView
}

// headsPage renders GET /heads — lists all non-revoked heads.
func (h *handlers) headsPage(e *core.RequestEvent) error {
	data, err := h.buildHeadsData()
	if err != nil {
		return e.InternalServerError("loading heads", err)
	}
	return h.render(e, "heads.html", data)
}

func (h *handlers) buildHeadsData() (headsData, error) {
	recs, err := h.app.FindRecordsByFilter(
		"heads",
		"status = 'active'",
		"-@rowid", 0, 0,
	)
	if err != nil {
		return headsData{}, fmt.Errorf("listing heads: %w", err)
	}
	views := make([]headView, 0, len(recs))
	for _, r := range recs {
		hv := headViewFrom(h.app, r)
		views = append(views, hv)
	}
	dock, _ := h.dockData()
	return headsData{Title: "Heads", MainClass: "profile-page", Dock: dock, Heads: views}, nil
}

func headViewFrom(app core.App, r *core.Record) headView {
	var exp string
	if t := r.GetDateTime("expires"); !t.IsZero() {
		d := time.Until(t.Time())
		switch {
		case d < 0:
			exp = "expired"
		case d < time.Hour:
			exp = fmt.Sprintf("%dm", int(d.Minutes()))
		case d < 24*time.Hour:
			exp = fmt.Sprintf("%dh", int(d.Hours()))
		default:
			exp = t.Time().Format("Jan 2")
		}
	}
	pref := r.GetString("balaur_avatar")
	if pref == "" {
		pref = store.GetOwnerSetting(app, "balaur_avatar", "balaur-01")
	}
	return headView{
		ID:            r.Id,
		Name:          r.GetString("name"),
		Purpose:       r.GetString("purpose"),
		Status:        r.GetString("status"),
		Expires:       exp,
		AvatarURL:     store.HeadBalaurAvatarURL(app, r.Id),
		BalaurOptions: buildBalaurHeadOptionsFor(pref), // roster lives in models.go
	}
}

// headChatPage renders GET /heads/{id}/chat — the focused conversation channel
// for a sub-head. The page is the head's own branch conversation.
func (h *handlers) headChatPage(e *core.RequestEvent) error {
	headID := e.Request.PathValue("id")
	head, err := h.app.FindRecordById("heads", headID)
	if err != nil {
		return e.NotFoundError("head not found", nil)
	}
	if head.GetString("status") != "active" {
		return e.ForbiddenError("head is not active", nil)
	}

	conv, err := conversation.ForHead(h.app, head)
	if err != nil {
		return e.InternalServerError("loading head conversation", err)
	}

	recs, _ := conversation.History(h.app, conv.Id, historyWindow)
	history := h.messageViewsForHead(recs, head)

	client, clientErr := h.clients.Active(h.app)
	data := headChatData{
		Title:           head.GetString("name") + " · Balaur",
		HeadID:          headID,
		HeadName:        head.GetString("name"),
		HeadPurpose:     head.GetString("purpose"),
		BalaurAvatarURL: store.HeadBalaurAvatarURL(h.app, headID),
		SoulAvatarURL:   store.SoulAvatarURL(h.app),
		OwnerName:       store.OwnerName(h.app),
		History:         history,
		ChatReady:       clientErr == nil && client != nil,
	}
	return h.render(e, "head-chat.html", data)
}

type headChatData struct {
	Title           string
	HeadID          string
	HeadName        string
	HeadPurpose     string
	BalaurAvatarURL string
	SoulAvatarURL   string
	OwnerName       string
	History         []messageView
	ChatReady       bool
}

// headChat handles POST /ui/heads/{id}/chat — one turn in the head's
// conversation. It mirrors the master chat gateway (chat.go): the shared turn
// pipeline (internal/turn) stays the source of truth and its agent.Event stream
// is adapted into a Datastar SSE stream of element patches via chatStream.
func (h *handlers) headChat(e *core.RequestEvent) error {
	headID := e.Request.PathValue("id")
	msg := readChatMessage(e)
	if msg == "" {
		return e.BadRequestError("empty message", nil)
	}

	head, err := h.app.FindRecordById("heads", headID)
	if err != nil {
		return e.NotFoundError("head not found", nil)
	}
	if head.GetString("status") != "active" {
		return e.ForbiddenError("head is not active", nil)
	}

	conv, err := conversation.ForHead(h.app, head)
	if err != nil {
		return e.InternalServerError("loading head conversation", err)
	}

	balaURL := store.HeadBalaurAvatarURL(h.app, headID)
	soulURL := store.SoulAvatarURL(h.app)
	ownerName := store.OwnerName(h.app)
	headName := head.GetString("name")

	cs := h.newChatStream(e, balaURL, headName, soulURL, ownerName)

	client, err := h.clients.Active(h.app)
	if err != nil {
		cs.appendChat("chat-msg-user", messageView{
			SoulAvatarURL: soulURL, OwnerName: ownerName, Content: msg,
		})
		_ = cs.sse.MarshalAndPatchSignals(chatSignals{Message: ""})
		cs.note("", h.chatErrText(err))
		return nil
	}

	cs.start(msg)
	_, runErr := turn.RunFor(e.Request.Context(), h.app, client, conv,
		headName, head.GetString("purpose"), msg, cs.emit)
	cs.finish()
	if runErr != nil {
		h.app.Logger().Warn("head chat: turn failed", "head", headID, "error", runErr)
	}
	return nil
}

// messageViewsForHead is like messageViews but uses the head's Balaur avatar
// instead of the owner's preference, and the head's name for the "who" label.
func (h *handlers) messageViewsForHead(recs []*core.Record, head *core.Record) []messageView {
	soulURL := store.SoulAvatarURL(h.app)
	headURL := store.HeadBalaurAvatarURL(h.app, head.Id)
	ownerName := store.OwnerName(h.app)
	out := make([]messageView, 0, len(recs))
	headName := head.GetString("name")
	for _, r := range recs {
		mv := messageView{
			Role:            r.GetString("role"),
			Tool:            r.GetString("tool_name"),
			Content:         r.GetString("content"),
			Origin:          r.GetString("origin"),
			SoulAvatarURL:   soulURL,
			BalaurAvatarURL: headURL,
			OwnerName:       ownerName,
			WhoLabel:        headName,
		}
		if mv.Role == "assistant" && mv.Content == "" {
			continue
		}
		out = append(out, mv)
	}
	return out
}

// setHeadAvatar handles POST /ui/heads/{id}/avatar — saves the head's
// balaur_avatar and re-renders just that head's card fragment.
func (h *handlers) setHeadAvatar(e *core.RequestEvent) error {
	headID := e.Request.PathValue("id")
	key := e.Request.FormValue("balaur_avatar")
	if !store.ValidBalaurAvatarKey(key) {
		return e.BadRequestError("invalid balaur avatar", nil)
	}
	if err := store.SetHeadBalaurAvatar(h.app, headID, key); err != nil {
		return e.InternalServerError("saving head avatar", err)
	}
	r, err := h.app.FindRecordById("heads", headID)
	if err != nil {
		return e.InternalServerError("reloading head", err)
	}
	hv := headViewFrom(h.app, r)

	var buf strings.Builder
	if err := h.tmpl.ExecuteTemplate(&buf, "head_card", hv); err != nil {
		return e.InternalServerError("rendering head card", err)
	}
	sse := datastar.NewSSE(e.Response, e.Request)
	return sse.PatchElements(buf.String(),
		datastar.WithSelectorID("head-"+headID), datastar.WithModeOuter())
}

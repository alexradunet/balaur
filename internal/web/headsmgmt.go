package web

import (
	"fmt"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/store"
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
	Title string
	Heads []headView
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
		"status != 'revoked'",
		"-created", 0, 0,
	)
	if err != nil {
		return headsData{}, fmt.Errorf("listing heads: %w", err)
	}
	views := make([]headView, 0, len(recs))
	for _, r := range recs {
		hv := headViewFrom(h.app, r)
		views = append(views, hv)
	}
	return headsData{Title: "Heads", Heads: views}, nil
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
		BalaurOptions: buildBalaurHeadOptionsFor(app, pref),
	}
}

// buildBalaurHeadOptionsFor is like buildBalaurHeadOptions but takes an
// explicit active key rather than reading from owner_settings.
func buildBalaurHeadOptionsFor(_ core.App, activePref string) []AvatarOption {
	roster := []struct{ key, label string }{
		{"balaur-01", "Wise"}, {"balaur-02", "Ancient"},
		{"balaur-03", "Guardian"}, {"balaur-04", "Scholar"},
		{"balaur-05", "Wild"}, {"balaur-06", "Storm"},
		{"balaur-07", "Night"}, {"balaur-08", "Young"},
		{"balaur-09", "Ember"}, {"balaur-10", "Frost"},
		{"balaur-11", "Healer"}, {"balaur-12", "Trickster"},
		{"balaur-13", "Dreamer"}, {"balaur-14", "Forest"},
		{"balaur-15", "Dawn"}, {"balaur-16", "Sage"},
	}
	opts := make([]AvatarOption, len(roster))
	for i, r := range roster {
		opts[i] = AvatarOption{
			Key:    r.key,
			Label:  r.label,
			URL:    "/static/avatars/" + r.key + ".png",
			Active: r.key == activePref,
		}
	}
	return opts
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
	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	return h.tmpl.ExecuteTemplate(e.Response, "head_card", hv)
}

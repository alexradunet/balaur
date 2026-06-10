package web

import (
	"fmt"
	"os"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"

	"github.com/alexradunet/balaur/internal/conversation"
	"github.com/alexradunet/balaur/internal/recap"
)

const recapSeedMarker = "[demo recap seed]"

type seedTurn struct {
	At        time.Time
	Topic     string
	Assistant string
}

func (h *handlers) seedRecaps(e *core.RequestEvent) error {
	if !devSeedEnabled() {
		return e.NotFoundError("not found", nil)
	}
	master, err := conversation.Master(h.app)
	if err != nil {
		return e.InternalServerError("master conversation", err)
	}
	now := time.Now()
	for _, turn := range demoTurns(now) {
		if err := h.seedTurn(master.Id, turn); err != nil {
			return e.InternalServerError("seeding recap turn", err)
		}
	}
	for _, s := range demoSummaries(now) {
		if err := h.seedSummary(master.Id, s.period, s.content, s.count); err != nil {
			return e.InternalServerError("seeding recap summary", err)
		}
	}
	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(e.Response, `<p class="card-note">Demo recap data seeded. Return to chat and scroll up to view the telescope.</p>`)
	return nil
}

func devSeedEnabled() bool {
	return os.Getenv("BALAUR_DEV_SEED") == "1"
}

func (h *handlers) seedTurn(conversationID string, turn seedTurn) error {
	userContent := recapSeedMarker + " " + turn.Topic
	if _, err := h.app.FindFirstRecordByFilter("messages",
		"conversation = {:conv} && role = 'user' && content = {:content}",
		dbx.Params{"conv": conversationID, "content": userContent}); err == nil {
		return nil
	}
	col, err := h.app.FindCollectionByNameOrId("messages")
	if err != nil {
		return fmt.Errorf("finding messages collection: %w", err)
	}
	for _, msg := range []struct {
		role    string
		content string
	}{
		{"user", userContent},
		{"assistant", turn.Assistant},
	} {
		rec := core.NewRecord(col)
		rec.Set("conversation", conversationID)
		rec.Set("role", msg.role)
		rec.Set("content", msg.content)
		if err := h.app.Save(rec); err != nil {
			return fmt.Errorf("saving demo message: %w", err)
		}
		if _, err := h.app.DB().NewQuery("UPDATE messages SET created = {:at} WHERE id = {:id}").
			Bind(dbx.Params{"at": pbDate(turn.At), "id": rec.Id}).Execute(); err != nil {
			return fmt.Errorf("backdating demo message: %w", err)
		}
	}
	return nil
}

type seedSummary struct {
	period  recap.Period
	content string
	count   int
}

func (h *handlers) seedSummary(conversationID string, p recap.Period, content string, count int) error {
	if recap.Find(h.app, conversationID, p) != nil {
		return nil
	}
	col, err := h.app.FindCollectionByNameOrId("summaries")
	if err != nil {
		return fmt.Errorf("finding summaries collection: %w", err)
	}
	rec := core.NewRecord(col)
	rec.Set("conversation", conversationID)
	rec.Set("period_type", p.Type)
	rec.Set("period_start", p.Start.UTC())
	rec.Set("period_end", p.End.UTC())
	rec.Set("content", content)
	rec.Set("message_count", count)
	if err := h.app.Save(rec); err != nil {
		return fmt.Errorf("saving demo summary: %w", err)
	}
	return nil
}

func demoTurns(now time.Time) []seedTurn {
	loc := now.Location()
	points := []struct {
		monthsAgo int
		day       int
		topic     string
		reply     string
	}{
		{0, 1, "reviewed the month plan", "I kept the June plan compact: health, writing, and house repairs."},
		{0, 8, "prepared the weekly review", "The week centered on errands, a model setup check, and one open writing thread."},
		{1, 6, "mapped the garden work", "May's garden notes grouped into soil, fence, and seedling follow-ups."},
		{1, 19, "reworked the budget notes", "I marked the recurring costs and kept the uncertain items separate."},
		{2, 8, "sorted the project backlog", "April's backlog narrowed to the three tasks with visible next actions."},
		{3, 12, "captured travel ideas", "March's travel ideas stayed tentative, with dates left uncommitted."},
	}
	out := make([]seedTurn, 0, len(points))
	for _, p := range points {
		base := now.AddDate(0, -p.monthsAgo, 0)
		day := p.day
		last := time.Date(base.Year(), base.Month()+1, 0, 0, 0, 0, 0, loc).Day()
		if day > last {
			day = last
		}
		out = append(out, seedTurn{
			At:        time.Date(base.Year(), base.Month(), day, 10, 30, 0, 0, loc),
			Topic:     p.topic,
			Assistant: p.reply,
		})
	}
	return out
}

func demoSummaries(now time.Time) []seedSummary {
	var out []seedSummary
	add := func(p recap.Period, content string, count int) {
		if p.End.After(now) {
			return
		}
		out = append(out, seedSummary{period: p, content: content, count: count})
	}
	for _, d := range []int{1, 8} {
		p := recap.Day(time.Date(now.Year(), now.Month(), d, 10, 30, 0, 0, now.Location()))
		add(p, "Demo day recap: a short planning exchange and one concrete follow-up.", 2)
	}
	for i := 1; i <= 4; i++ {
		p := recap.Week(now.AddDate(0, 0, -7*i))
		add(p, fmt.Sprintf("Demo weekly recap: week %s had a visible thread for the recap telescope.", p.Start.Format("Jan 2")), 2)
	}
	for i := 1; i <= 4; i++ {
		p := recap.Month(now.AddDate(0, -i, 0))
		add(p, fmt.Sprintf("Demo monthly recap: %s grouped several small conversations into a monthly card.", p.Start.Format("January 2006")), 4)
	}
	return out
}

func pbDate(t time.Time) string {
	return t.UTC().Format(types.DefaultDateLayout)
}

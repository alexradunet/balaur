package taskcards

import (
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
	g "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/tasks"
	"github.com/alexradunet/balaur/internal/ui"
)

// renderTasks draws matching tasks as a BARE stack of individual TaskCards —
// no container/header chrome (contrast the quests summary card which wraps in
// an Article.kcard.ucard with head + footer). This is the "draw the cards for
// THOSE quests" surface, reachable by card_show, /ui/show, and show_cards.
func renderTasks(app core.App, params map[string]string) g.Node {
	now := time.Now()
	// cards.Validate already clamped limit to [1,50]; intParam handles missing/invalid.
	limit := intParam(params, "limit", 12)
	status := params["status"]
	if status == "" {
		status = "open"
	}

	var recs []*core.Record
	switch status {
	case "done":
		recs, _ = app.FindRecordsByFilter("tasks", "status = 'done'", "-updated", limit, 0, nil)
	case "all":
		recs, _ = app.FindRecordsByFilter("tasks", "status != 'dropped'", "-updated", limit, 0, nil)
	default: // open
		var terms []string
		if t := strings.TrimSpace(params["terms"]); t != "" {
			terms = strings.Fields(t)
		}
		open, _ := tasks.OpenTasks(app, terms)
		open = filterBucket(open, params["bucket"], now)
		if len(open) > limit {
			open = open[:limit]
		}
		recs = open
	}

	rows := viewsOf(recs, now)
	if len(rows) == 0 {
		return ui.EmptyState(ui.EmptyProps{Compact: true, Line: "No tasks match."})
	}
	items := make([]g.Node, 0, len(rows))
	for _, r := range rows {
		items = append(items, TaskCard(r))
	}
	// Bare stack — no card container/head/footer. CSS in basm.css.
	return Div(Class("tasks-stack"), g.Group(items))
}

// filterBucket narrows OPEN tasks to one due bucket via tasks.Bucket. An empty
// bucket string returns recs unchanged. (Only meaningful for status=open.)
func filterBucket(recs []*core.Record, bucket string, now time.Time) []*core.Record {
	if bucket == "" {
		return recs
	}
	bk := tasks.Bucket(recs, now)
	switch bucket {
	case "overdue":
		return bk.Overdue
	case "today":
		return bk.Today
	case "upcoming":
		return bk.Upcoming
	case "someday":
		return bk.Someday
	}
	return recs
}

func registerTasks(app core.App) {
	ui.RegisterCard("tasks", func(_ ui.CardSize, params map[string]string) (g.Node, error) {
		return renderTasks(app, params), nil
	})
}

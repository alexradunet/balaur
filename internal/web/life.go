package web

import (
	"fmt"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/life"
	"github.com/alexradunet/balaur/internal/tasks"
)

// /life mirrors what the owner actually logs — nothing is predefined.
// Numeric kinds chart as sparklines, text kinds list their recent lines,
// open habits show their live streaks. A kind exists because the owner
// logged it; the page imposes no taxonomy.

const (
	lifeWindowDays = 90
	sparkW, sparkH = 240, 48
)

type lifeKindView struct {
	Kind, Unit             string
	Count                  int
	Numeric                bool
	Points                 string // sparkline polyline points
	SparkLastX, SparkLastY string
	LastVal, LastAt        string
	Change                 string
	Recent                 []string
}

type lifeHabitView struct {
	Title     string
	Streak    int
	RecurLine string
}

func (h *handlers) lifePage(e *core.RequestEvent) error {
	now := time.Now()
	kinds, err := life.Kinds(h.app)
	if err != nil {
		return e.InternalServerError("loading life", err)
	}

	views := make([]lifeKindView, 0, len(kinds))
	for _, k := range kinds {
		recs, err := life.Series(h.app, k.Kind, now.AddDate(0, 0, -lifeWindowDays))
		if err != nil {
			continue
		}
		v := lifeKindView{Kind: k.Kind, Unit: k.Unit, Count: k.Count}
		if s := life.Summarize(recs); s.Points > 0 {
			v.Numeric = true
			v.LastVal = fmt.Sprintf("%g", s.Last)
			v.LastAt = s.LastAt.In(now.Location()).Format("Jan 2")
			if s.Points > 1 {
				v.Change = fmt.Sprintf("%+.4g over %dd", s.Last-s.First, lifeWindowDays)
				v.Points, v.SparkLastX, v.SparkLastY = sparkPoints(numericValues(recs), sparkW, sparkH)
			}
		} else {
			for i := len(recs) - 1; i >= 0 && len(v.Recent) < 5; i-- {
				line := recs[i].GetDateTime("noted_at").Time().In(now.Location()).Format("Jan 2")
				if t := recs[i].GetString("text"); t != "" {
					line += " — " + clipText(t, 120)
				}
				v.Recent = append(v.Recent, line)
			}
		}
		views = append(views, v)
	}

	dock, _ := h.dockData()
	return h.render(e, "life.html", map[string]any{
		"Title": "Life", "Dock": dock, "Kinds": views, "Habits": h.buildHabits(now),
	})
}

// buildHabits returns the owner's recurring tasks with their current streak,
// shared by the /life page and the habits card.
func (h *handlers) buildHabits(now time.Time) []lifeHabitView {
	recs, err := tasks.OpenTasks(h.app, nil)
	if err != nil {
		return nil
	}
	var recurring []*core.Record
	recurLines := make(map[string]string)
	for _, r := range recs {
		rule, err := tasks.Parse(r.GetString("recur"))
		if err != nil || rule.IsZero() {
			continue
		}
		recurring = append(recurring, r)
		recurLines[r.Id] = tasks.Describe(rule)
	}
	streaks := tasks.StreaksFor(h.app, recurring, now)
	habits := make([]lifeHabitView, 0, len(recurring))
	for _, r := range recurring {
		habits = append(habits, lifeHabitView{
			Title:     r.GetString("title"),
			Streak:    streaks[r.Id],
			RecurLine: recurLines[r.Id],
		})
	}
	return habits
}

func numericValues(recs []*core.Record) []float64 {
	var out []float64
	for _, r := range recs {
		if v := r.GetFloat("value_num"); v != 0 {
			out = append(out, v)
		}
	}
	return out
}

// sparkPoints maps values onto a w×h polyline, index-spaced — a trend
// glance, not a time chart. Returns the points and the last point's
// coordinates (for the gold dot).
func sparkPoints(vals []float64, w, h int) (points, lastX, lastY string) {
	if len(vals) < 2 {
		return "", "", ""
	}
	lo, hi := vals[0], vals[0]
	for _, v := range vals {
		if v < lo {
			lo = v
		}
		if v > hi {
			hi = v
		}
	}
	span := hi - lo
	if span == 0 {
		span = 1
	}
	const pad = 4.0
	var b strings.Builder
	var px, py string
	for i, v := range vals {
		x := float64(i)*float64(w-8)/float64(len(vals)-1) + 4
		y := pad + (1-(v-lo)/span)*(float64(h)-2*pad)
		px, py = fmt.Sprintf("%.1f", x), fmt.Sprintf("%.1f", y)
		fmt.Fprintf(&b, "%s,%s ", px, py)
	}
	return strings.TrimSpace(b.String()), px, py
}

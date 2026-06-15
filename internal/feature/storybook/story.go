package storybook

import g "maragu.dev/gomponents"

// Story is one component's storybook entry: its url/anchor ID, sidebar Group and
// Title, and the Canvas that renders its variants. The ordered registry below is
// the single source for both the sidebar nav and the /storybook/{id} routes.
type Story struct {
	ID     string
	Group  string
	Title  string
	Canvas func() g.Node
}

var stories = []Story{
	{"button", "Atoms", "Button", buttonCanvas},
	{"tag", "Atoms", "Tag", tagCanvas},
	{"pips", "Atoms", "Pips", pipsCanvas},
	{"card", "Atoms", "Card", cardCanvas},
	{"stitch", "Atoms", "Stitch", stitchCanvas},
	{"folkband", "Atoms", "FolkBand", folkbandCanvas},
	{"avatar", "Atoms", "Avatar", avatarCanvas},
	{"icon", "Atoms", "Icon", iconCanvas},
	{"badge", "Feedback", "Badge", badgeCanvas},
	{"alert", "Feedback", "Alert", alertCanvas},
	{"tooltip", "Feedback", "Tooltip", tooltipCanvas},
	{"skeleton", "Feedback", "Skeleton", skeletonCanvas},
	{"textfield", "Forms", "TextField", textfieldCanvas},
	{"select", "Forms", "Select", selectCanvas},
	{"toggle", "Forms", "Toggle", toggleCanvas},
	{"tabs", "Navigation", "Tabs", tabsCanvas},
}

// Stories returns the ordered registry.
func Stories() []Story { return stories }

// Lookup returns the story with the given ID.
func Lookup(id string) (Story, bool) {
	for _, s := range stories {
		if s.ID == id {
			return s, true
		}
	}
	return Story{}, false
}

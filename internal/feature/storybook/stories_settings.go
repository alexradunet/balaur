package storybook

import (
	"github.com/alexradunet/balaur/internal/feature/modelcards"
	h "maragu.dev/gomponents/html"
)

// Story builders for the Models group — the settings surface where the owner
// sees the local models on this box, selects the active one, and adds a GGUF
// file to run in-process via the embedded engine.

func modelcardStory() Story {
	return Story{
		ID: "modelcard", Group: "Models", Title: "ModelCard",
		Blurb: "One local model: its name, the GGUF file and where it lives, and a status-appropriate action — In use (active), Use this model (available), or Get this model (missing).",
		Variants: []Variant{
			{"active · in use", modelcards.ModelCard(modelcards.ModelView{
				ID: "m1", Name: "Qwen3 0.6B", Detail: "qwen3-0.6b-q8_0.gguf · on this box",
				Kind: "local", Status: modelcards.StatusActive, VRAM: "~1 GB",
			})},
			{"available", modelcards.ModelCard(modelcards.ModelView{
				ID: "m2", Name: "Qwen3.5 4B", Detail: "Qwen3.5-4B-Q4_K_M.gguf · on this box",
				Kind: "local", Status: modelcards.StatusAvailable, VRAM: "~3 GB",
			})},
			{"missing · file not found", modelcards.ModelCard(modelcards.ModelView{
				ID: "m3", Name: "Llama 3.1 8B", Detail: "/models/llama-3.1-8b.gguf · file not found",
				Kind: "missing", Status: modelcards.StatusMissing,
			})},
			{"downloading · progress meter", modelcards.ModelCard(modelcards.ModelView{
				ID: "dl1", Name: "Qwen3.5 4B", Detail: "Downloading…",
				Kind: "local", Status: modelcards.StatusDownloading,
				Progress: 42, ProgressLabel: "1.1 GB / 2.7 GB · 4.1 MB/s",
			})},
		},
		Props: []Prop{
			{"ID", "string", "—", "Model record id; drives the element id and the action posts."},
			{"Name", "string", "—", "Display name."},
			{"Detail", "string", "—", "One line: GGUF file + location, or 'file not found'."},
			{"Kind", "string", "—", "Small kicker label, e.g. 'local' or 'missing'."},
			{"Status", "string", `"available"`, "active → In use (disabled); available → Use this model; missing → no action; downloading → progress meter + Cancel."},
			{"VRAM", "string", "—", "Optional estimate (e.g. '~6 GB'); rendered as a tag when set."},
			{"Progress", "int", "0", "0..100 download progress; only shown when Status == downloading."},
			{"ProgressLabel", "string", "—", "Human progress line shown under the progress meter."},
		},
		Dos: []string{
			"Make the active model unmistakable — only one is In use.",
			"Show the GGUF filename so the owner knows exactly what runs.",
			"Show real bytes + speed so a multi-GB download feels alive.",
		},
		Donts: []string{
			"Offer Use on a missing file — gate selection behind presence on disk.",
			"Hide why a model is unavailable.",
			"Mark a model 'In use' before its checksum verifies.",
		},
	}
}

func modelspanelStory() Story {
	cpuGpu := []modelcards.ProcessorOption{
		{Key: "cpu", Installed: true, Selected: true},
		{Key: "vulkan", Installed: true},
	}
	catalog := []modelcards.OfficialCTA{
		{Key: "small", Name: "Qwen3.5 2B", Tagline: "Small & fast", Meta: "Q4_K_M · 2B · Apache-2.0", SizeLabel: "1.3 GB"},
		{Key: "medium", Name: "Qwen3.5 4B", Tagline: "Balanced · recommended", Meta: "Q4_K_M · 4B · Apache-2.0", SizeLabel: "2.7 GB"},
	}
	return Story{
		ID: "modelspanel", Group: "Models", Title: "Models panel", Wide: true,
		Blurb: "The Models settings section: the runtime rows, the \"Run on\" CPU/GPU control (with a restart note when the saved choice differs from what's live), the grid of local models (or an empty state on a fresh box), and a download/install card per curated model (small/medium tiers). It is the SSE patch target for every model action. There is no manual GGUF-path form — the curated catalog is the supported path.",
		Variants: []Variant{
			{"populated", modelcards.Panel(modelcards.PanelView{
				ProcessorRunning: "cpu",
				Processors:       cpuGpu,
				Models: []modelcards.ModelView{
					{ID: "m1", Name: "Qwen3.5 2B", Detail: "Qwen3.5-2B-Q4_K_M.gguf · on this box", Kind: "local", Status: modelcards.StatusActive, VRAM: "~1.5 GB"},
					{ID: "m2", Name: "Qwen3.5 4B", Detail: "Qwen3.5-4B-Q4_K_M.gguf · on this box", Kind: "local", Status: modelcards.StatusAvailable, VRAM: "~3 GB"},
				},
			})},
			{"empty · fresh box · both tiers", modelcards.Panel(modelcards.PanelView{
				ProcessorRunning: "cpu",
				Processors:       []modelcards.ProcessorOption{{Key: "cpu", Installed: true, Selected: true}, {Key: "vulkan", Installed: false}},
				OfficialCTAs:     catalog,
			})},
			{"run on GPU · restart pending", modelcards.Panel(modelcards.PanelView{
				ProcessorRunning: "cpu",
				RestartPending:   true,
				Processors:       []modelcards.ProcessorOption{{Key: "cpu", Installed: true}, {Key: "vulkan", Installed: true, Selected: true}},
				Models: []modelcards.ModelView{
					{ID: "m1", Name: "Qwen3.5 4B", Detail: "Qwen3.5-4B-Q4_K_M.gguf · on this box", Kind: "local", Status: modelcards.StatusActive, VRAM: "~3 GB"},
				},
			})},
			{"already downloaded · install", modelcards.Panel(modelcards.PanelView{
				ProcessorRunning: "cpu",
				Processors:       cpuGpu,
				OfficialCTAs: []modelcards.OfficialCTA{
					{Key: "medium", Name: "Qwen3.5 4B", Tagline: "Balanced · recommended", Meta: "Q4_K_M · 4B · Apache-2.0", SizeLabel: "2.7 GB", OnDisk: true},
				},
			})},
			{"error", modelcards.Panel(modelcards.PanelView{ProcessorRunning: "cpu", Processors: cpuGpu, Error: "local inference engine not initialized"})},
			{"downloading · official model", modelcards.Panel(modelcards.PanelView{
				ProcessorRunning: "cpu",
				Processors:       cpuGpu,
				Models: []modelcards.ModelView{
					{ID: "official-dl", Name: "Qwen3.5 4B", Detail: "Downloading…",
						Kind: "local", Status: modelcards.StatusDownloading,
						Progress: 67, ProgressLabel: "1.8 GB / 2.7 GB · 5.2 MB/s"},
				},
			})},
			{"runtime not installed · both install-first", modelcards.Panel(modelcards.PanelView{
				ProcessorRunning: "cpu",
				Processors:       []modelcards.ProcessorOption{{Key: "cpu"}, {Key: "vulkan"}},
				RuntimeMissing:   true,
				OfficialCTAs:     catalog,
			})},
			{"GPU unsupported on this platform", modelcards.Panel(modelcards.PanelView{
				ProcessorRunning: "cpu",
				Processors:       []modelcards.ProcessorOption{{Key: "cpu", Installed: true, Selected: true}, {Key: "vulkan", Unsupported: true}},
				Models: []modelcards.ModelView{
					{ID: "m1", Name: "Qwen3.5 4B", Detail: "Qwen3.5-4B-Q4_K_M.gguf · on this box", Kind: "local", Status: modelcards.StatusActive, VRAM: "~3 GB"},
				},
			})},
		},
		Props: []Prop{
			{"Processors", "[]ProcessorOption", "nil", "The \"Run on\" choices (cpu/gpu) with Installed + Selected flags; empty hides the control."},
			{"ProcessorRunning", "string", "—", "The variant the live engine actually loaded — shown in the restart note."},
			{"RestartPending", "bool", "false", "When true, the saved processor differs from the running one → shows the restart note."},
			{"Models", "[]ModelView", "nil", "Available/active/missing models; empty renders the empty state."},
			{"Error", "string", "—", "Optional error banner above the grid."},
			{"OfficialCTAs", "[]OfficialCTA", "nil", "One download/install card per curated model not yet registered (Key/Name/Tagline/Meta/SizeLabel/OnDisk). Empty hides the section."},
			{"RuntimeMissing", "bool", "false", "When true, shows the runtime-not-installed alert."},
		},
		Dos: []string{
			"Offer CPU vs GPU as one clear control, and say plainly that switching needs a restart.",
			"Show each tier's size + role so the owner can trade speed against capability.",
			"Reuse an already-downloaded file: offer Install, never a second multi-GB download.",
		},
		Donts: []string{
			"Imply remote/API models — v1 runs local GGUF only.",
			"Offer a GPU pill whose runtime isn't installed as if it were selectable.",
		},
	}
}

func runtimesectionStory() Story {
	return Story{
		ID: "runtimesection", Group: "Models", Title: "Runtime section", Wide: true,
		Blurb: "The Local AI runtime section inside the Models panel: one row per variant (CPU, Vulkan) showing install status and an Install action. Vulkan shows a host-loader note. Unsupported triples show a disabled button.",
		Variants: []Variant{
			{"cpu available · vulkan available", h.Section(h.Class("k-section"),
				modelcards.RuntimeCard(modelcards.RuntimeView{Processor: "cpu", Status: modelcards.StatusAvailable}),
				modelcards.RuntimeCard(modelcards.RuntimeView{Processor: "vulkan", Status: modelcards.StatusAvailable, NeedsHostLoader: true}),
			)},
			{"cpu installed · vulkan available", h.Section(h.Class("k-section"),
				modelcards.RuntimeCard(modelcards.RuntimeView{Processor: "cpu", Status: modelcards.StatusInstalled, Version: "b9664"}),
				modelcards.RuntimeCard(modelcards.RuntimeView{Processor: "vulkan", Status: modelcards.StatusAvailable, NeedsHostLoader: true}),
			)},
			{"both installed", h.Section(h.Class("k-section"),
				modelcards.RuntimeCard(modelcards.RuntimeView{Processor: "cpu", Status: modelcards.StatusInstalled, Version: "b9664"}),
				modelcards.RuntimeCard(modelcards.RuntimeView{Processor: "vulkan", Status: modelcards.StatusInstalled, Version: "b9664", NeedsHostLoader: true}),
			)},
			{"cpu installing", h.Section(h.Class("k-section"),
				modelcards.RuntimeCard(modelcards.RuntimeView{Processor: "cpu", Status: modelcards.StatusInstalling}),
				modelcards.RuntimeCard(modelcards.RuntimeView{Processor: "vulkan", Status: modelcards.StatusAvailable, NeedsHostLoader: true}),
			)},
			{"vulkan unsupported", h.Section(h.Class("k-section"),
				modelcards.RuntimeCard(modelcards.RuntimeView{Processor: "cpu", Status: modelcards.StatusAvailable}),
				modelcards.RuntimeCard(modelcards.RuntimeView{Processor: "vulkan", Status: modelcards.StatusUnsupported, NeedsHostLoader: true}),
			)},
		},
		Props: []Prop{
			{"Processor", "string", "—", `"cpu" or "vulkan".`},
			{"Status", "string", `"available"`, "installed · available · installing · unsupported."},
			{"Version", "string", "—", "Installed version tag, e.g. b9664."},
			{"NeedsHostLoader", "bool", "false", "When true, renders the Vulkan host-loader note."},
		},
		Dos: []string{
			"Show both cpu and vulkan rows so the owner knows what's available.",
			"State the Vulkan host-loader requirement clearly so the owner knows what's needed.",
		},
		Donts: []string{
			"Offer Install for an unsupported triple — show a disabled 'Not supported' button instead.",
			"Start install on boot — owner-click only.",
		},
	}
}

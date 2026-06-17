package storybook

import (
	"github.com/alexradunet/balaur/internal/feature/modelcards"
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
				ID: "m2", Name: "Gemma 4 E4B", Detail: "gemma4-e4b.gguf · on this box",
				Kind: "local", Status: modelcards.StatusAvailable, VRAM: "~6 GB",
			})},
			{"missing · file not found", modelcards.ModelCard(modelcards.ModelView{
				ID: "m3", Name: "Llama 3.1 8B", Detail: "/models/llama-3.1-8b.gguf · file not found",
				Kind: "missing", Status: modelcards.StatusMissing,
			})},
			{"downloading · progress meter", modelcards.ModelCard(modelcards.ModelView{
				ID: "dl1", Name: "Gemma 4 E4B", Detail: "Downloading…",
				Kind: "local", Status: modelcards.StatusDownloading,
				Progress: 42, ProgressLabel: "2.2 GB / 5.3 GB · 4.1 MB/s",
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
	return Story{
		ID: "modelspanel", Group: "Models", Title: "Models panel", Wide: true,
		Blurb: "The Models settings section: the active processor (cpu or vulkan), the grid of local models (or an empty state on a fresh box), the official-model CTA when applicable, and the add-a-local-model form. It is the SSE patch target for every model action.",
		Variants: []Variant{
			{"populated", modelcards.Panel(modelcards.PanelView{
				Processor: "cpu",
				Models: []modelcards.ModelView{
					{ID: "m1", Name: "Qwen3 0.6B", Detail: "qwen3-0.6b-q8_0.gguf · on this box", Kind: "local", Status: modelcards.StatusActive, VRAM: "~1 GB"},
					{ID: "m2", Name: "Gemma 4 E4B", Detail: "gemma4-e4b.gguf · on this box", Kind: "local", Status: modelcards.StatusAvailable, VRAM: "~6 GB"},
				},
			})},
			{"empty · fresh box", modelcards.Panel(modelcards.PanelView{Processor: "vulkan"})},
			{"error", modelcards.Panel(modelcards.PanelView{Processor: "cpu", Error: "local inference engine not initialized"})},
			{"downloading · official model", modelcards.Panel(modelcards.PanelView{
				Processor: "cpu",
				Models: []modelcards.ModelView{
					{ID: "official-dl", Name: "Gemma 4 E4B", Detail: "Downloading…",
						Kind: "local", Status: modelcards.StatusDownloading,
						Progress: 67, ProgressLabel: "3.6 GB / 5.3 GB · 5.2 MB/s"},
				},
			})},
			{"download error", modelcards.Panel(modelcards.PanelView{
				Processor:       "cpu",
				Error:           "sha256 mismatch: want abc123 got def456",
				ShowOfficialCTA: true,
				OfficialCTAName: "Gemma 4 E4B",
				OfficialCTAMeta: "Q4_K_M · E4B (~4.5B eff.) · Apache-2.0",
			})},
			{"runtime not installed", modelcards.Panel(modelcards.PanelView{
				Processor:       "cpu",
				RuntimeMissing:  true,
				ShowOfficialCTA: true,
				OfficialCTAName: "Gemma 4 E4B",
				OfficialCTAMeta: "Q4_K_M · E4B (~4.5B eff.) · Apache-2.0",
			})},
		},
		Props: []Prop{
			{"Processor", "string", `"cpu"`, "The active llama.cpp variant — cpu or vulkan."},
			{"Models", "[]ModelView", "nil", "Available/active/missing models; empty renders the empty state."},
			{"Error", "string", "—", "Optional error banner above the grid."},
			{"ShowOfficialCTA", "bool", "false", "When true, shows the Get our official model CTA."},
			{"OfficialCTAName", "string", "—", "Official model display name in the CTA."},
			{"OfficialCTAMeta", "string", "—", "One-line: quant + params + license."},
			{"RuntimeMissing", "bool", "false", "When true, shows the runtime-not-installed alert."},
		},
		Dos: []string{
			"Show the processor so GPU owners can confirm Vulkan is live.",
			"Lead a fresh box to the Download & install CTA — one click to the curated model.",
		},
		Donts: []string{
			"Imply remote/API models — v1 runs local GGUF only.",
		},
	}
}

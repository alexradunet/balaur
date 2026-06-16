package kronk

import (
	"fmt"
	"strings"

	"github.com/ardanlabs/kronk/sdk/kronk/vram"
)

// EstimateVRAM returns a human "~X.X GB" estimate of the VRAM a GGUF model needs
// at a 4096-token context with all layers on the GPU, or "" when path is not a
// readable GGUF or estimation fails. It is pure Go — it reads only the GGUF
// header/tensor metadata, loads no model, and needs no native library, so it is
// safe to call while rendering the models list.
func EstimateVRAM(ggufPath string) string {
	if !strings.HasSuffix(strings.ToLower(ggufPath), ".gguf") {
		return ""
	}
	res, err := vram.FromFiles([]string{ggufPath}, vram.Config{
		ContextWindow:   4096,
		BytesPerElement: 2, // f16 KV cache
		Slots:           1,
		GPULayers:       0, // all layers on the GPU
	})
	if err != nil {
		return ""
	}
	return fmt.Sprintf("~%.1f GB", float64(res.TotalVRAM)/(1<<30))
}

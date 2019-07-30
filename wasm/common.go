package wasm

var (
	MAX_INIATIAL_CAP = uint32(10 * 1024 * 1024)
)

func min(x, y uint32) uint32 {
	if x < y {
		return x
	}
	return y
}

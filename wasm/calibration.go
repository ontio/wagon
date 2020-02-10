package wasm

const (
	MaxTableSize       = 1024
	MaxLocalEntryCount = 1024
	// As per the WebAssembly spec: https://github.com/WebAssembly/design/blob/27ac254c854994103c24834a994be16f74f54186/Semantics.md#linear-memory
	MaxMemorySize = 10 * 1024 * 1024
	WasmPageSize  = 65536
	MaxPageNum    = MaxMemorySize / WasmPageSize
)

func checkTableLimits(m *Module) error {
	if m.Table != nil {
		for i, e := range m.Table.Entries {
			if e.Limits.Initial > MaxTableSize {
				return OutsizeError{"First Calibration Table", uint64(e.Limits.Initial), uint64(MaxTableSize)}
			}

			if e.Limits.Flags&0x1 != 0 && e.Limits.Maximum > MaxTableSize {
				return OutsizeError{"First Calibration Table", uint64(e.Limits.Maximum), uint64(MaxTableSize)}
			} else {
				m.Table.Entries[i].Limits.Flags = 1
				m.Table.Entries[i].Limits.Maximum = MaxTableSize
			}
		}
	}

	return nil
}

func checkMemoryLimits(m *Module) error {
	if m.Memory != nil {
		for i, e := range m.Memory.Entries {
			if e.Limits.Initial > MaxPageNum {
				return OutsizeError{"First Calibration Memory", uint64(e.Limits.Initial), uint64(MaxPageNum)}
			}

			if e.Limits.Flags == 1 {
				if e.Limits.Maximum > MaxPageNum {
					return OutsizeError{"First Calibration Memory", uint64(e.Limits.Maximum), uint64(MaxPageNum)}
				}
			} else {
				m.Memory.Entries[i].Limits.Flags = 1
				m.Memory.Entries[i].Limits.Maximum = MaxPageNum
			}
		}
	}

	return nil
}

func WasmCalibration(m *Module) error {
	err := checkTableLimits(m)
	if err != nil {
		return err
	}

	err = checkMemoryLimits(m)
	if err != nil {
		return err
	}

	return nil
}

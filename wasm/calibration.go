package wasm

const (
	MAX_TABLE_SIZE       = 1024
	MAX_LOCALENTRY_COUNT = 1024
	// As per the WebAssembly spec: https://github.com/WebAssembly/design/blob/27ac254c854994103c24834a994be16f74f54186/Semantics.md#linear-memory
	MAX_MEMORY_SIZE = 10 * 1024 * 1024 // (10 MB)
	WASM_PAGE_SIZE  = 65536            // (64 KB)
	MAX_PAGE_NUM    = MAX_MEMORY_SIZE / WASM_PAGE_SIZE
)

func firstStepCalibrationOfSectionTables(m *Module) error {
	for i, e := range m.Table.Entries {
		if e.Limits.Initial > uint32(MAX_TABLE_SIZE) {
			return SizeOverFlowError{"First Calibration Table", uint64(e.Limits.Initial), uint64(MAX_TABLE_SIZE)}
		}

		if e.Limits.Flags&0x1 != 0 && e.Limits.Maximum > uint32(MAX_TABLE_SIZE) {
			return SizeOverFlowError{"First Calibration Table", uint64(e.Limits.Maximum), uint64(MAX_TABLE_SIZE)}
		} else {
			m.Table.Entries[i].Limits.Flags = 1
			m.Table.Entries[i].Limits.Maximum = MAX_TABLE_SIZE
		}
	}

	return nil
}

func firstStepCalibrationOfSectionMemory(m *Module) error {
	for i, e := range m.Memory.Entries {
		if e.Limits.Initial > uint32(MAX_PAGE_NUM) {
			return SizeOverFlowError{"First Calibration Memory", uint64(e.Limits.Initial), uint64(MAX_PAGE_NUM)}
		}

		if e.Limits.Flags&0x1 != 0 && e.Limits.Maximum > uint32(MAX_PAGE_NUM) {
			return SizeOverFlowError{"First Calibration Memory", uint64(e.Limits.Maximum), uint64(MAX_PAGE_NUM)}
		} else {
			m.Memory.Entries[i].Limits.Flags = 1
			m.Memory.Entries[i].Limits.Maximum = MAX_PAGE_NUM
		}
	}

	return nil
}

func FirstStepCalibration(m *Module) error {
	err := firstStepCalibrationOfSectionTables(m)
	if err != nil {
		return err
	}

	err = firstStepCalibrationOfSectionMemory(m)
	if err != nil {
		return err
	}

	return nil
}

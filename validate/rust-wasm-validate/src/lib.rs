use std::mem;

use wasmparser::{OperatorValidatorConfig, ValidatingParserConfig};

#[no_mangle]
pub extern "C" fn alloc_buffer(n: usize) -> *mut u8 {
    let mut buff = vec![0u8; n];
    let p = buff.as_mut_ptr();
    mem::forget(buff);

    p
}

// validate wasm code, return 0 if valid
// note: code_ptr must be the return value of alloc_buffer function
// code_len must be the param of alloc_buffer function
// the mem allocated by alloc_buffer will be freed in this function
#[no_mangle]
pub extern "C" fn wasm_validate(code_ptr: *mut u8, code_len: usize) -> u32 {
    let code = unsafe { Vec::from_raw_parts(code_ptr, code_len, code_len) };
    if wasmi_validate(&code).is_err() {
        return 1;
    }

    if wasmparser::validate(
        &code,
        Some(ValidatingParserConfig {
            operator_config: OperatorValidatorConfig {
                enable_threads: false,
                enable_reference_types: false,
                enable_simd: false,
                enable_bulk_memory: false,
                enable_multi_value: false,
                deterministic_only: false,
            },
        }),
    ).is_err() {
        return 2;
    }

    0
}

fn wasmi_validate(code: &[u8]) -> Result<(), wasmi::Error> {
    let module = wasmi::Module::from_buffer(code)?;
    module.deny_floating_point()?;

    Ok(())
}


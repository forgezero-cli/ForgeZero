// Copyright (c) 2026 AlexVoste. All Rights Reserved.
// This module is used as a static library, for now the basic code for file caching is implemented here, later it will be used as a static lib.

use sha2::{Digest, Sha256};
use std::ffi::CStr;
use std::fs::File;
use std::io::Read;
use std::os::raw::c_char;
use std::path::Path;

#[no_mangle]
extern "C" fn rust_hash_file(path: *const c_char, out: *mut c_char, out_len: usize) -> i32 {
    let c_str = unsafe { CStr::from_ptr(path) };
    let path_str = match c_str.to_str() {
        Ok(s) => s,
        Err(_) => return -1,
    };

    let path = Path::new(path_str);

    match path.to_str() {
        Some(s) => println!("{}", s),
        None => return -2,
    }

    let mut file = match File::open(path) {
        Ok(f) => f,
        Err(_) => return -3,
    };
    let mut hasher = Sha256::new();
    let mut buffer = [0; 8192];
    loop {
        let bytes_read = match file.read(&mut buffer) {
            Ok(n) => n,
            Err(_) => return -4,
        };
        if bytes_read == 0 {
            break;
        }
        hasher.update(&buffer[..bytes_read]);
    }

    let result = hasher.finalize();
    let hex = format!("{:x}", result);
    if hex.len() + 1 > out_len {
        return -5;
    }
    unsafe {
        std::ptr::copy_nonoverlapping(hex.as_ptr(), out as *mut u8, hex.len());
        *out.add(hex.len()) = 0;
    }

    0
}

#[no_mangle]
extern "C" fn rust_copy_file(src: *const c_char, dst: *const c_char) -> i32 {
    let src_cstr = unsafe { CStr::from_ptr(src) };
    let dst_cst = unsafe { CStr::from_ptr(dst) };
    let src_str = match src_cstr.to_str() {
        Ok(s) => s,
        Err(_) => return -1,
    };

    let dst_str = match dst_cst.to_str() {
        Ok(s) => s,
        Err(_) => return -2,
    };

    match std::fs::copy(src_str, dst_str) {
        Ok(_) => 0,
        Err(_) => -3,
    }
}

#[cfg(test)]
mod tests {
    #[test]
    fn placeholder_test() {
        assert!(true);
    }
}

fn main() {
    println!("static lib complited and linked successfully!");
}

use std::os::raw::c_char;

#[no_mangle]
extern "C" fn rust_hash_file(_path: *const c_char, _out: *mut c_char, _out_len: usize) -> i32 {
    return 1;
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

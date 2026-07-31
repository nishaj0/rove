fn main() {
    println!("cargo:rerun-if-changed=updater-public-key.txt");
    tauri_build::build()
}

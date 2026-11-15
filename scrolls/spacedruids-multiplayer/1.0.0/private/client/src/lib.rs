mod protocol;
#[cfg(target_arch = "wasm32")]
wit_bindgen::generate!({ path: env!("DRUID_WIT_PATH"), world: "websocket-client", generate_all });
#[cfg(target_arch = "wasm32")]
mod browser;
#[cfg(target_arch = "wasm32")]
use browser::Client;
#[cfg(target_arch = "wasm32")]
export!(Client);

#[cfg(target_arch = "wasm32")]
wit_bindgen::generate!({
    path: env!("DRUID_WIT_PATH"),
    world: "openui-client",
});

#[cfg(target_arch = "wasm32")]
struct PublicPage;

#[cfg(target_arch = "wasm32")]
impl exports::druid::openui::client::Guest for PublicPage {
    async fn init() {}
    async fn handle_action(_action: String, _params: String) {}
}

#[cfg(target_arch = "wasm32")]
export!(PublicPage);

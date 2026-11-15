wit_bindgen::generate!({ path: env!("DRUID_WIT_PATH"), world: "browser-client", generate_all });
mod invaders;

struct SpaceInvaders;
impl exports::druid::openui::client::Guest for SpaceInvaders {
    async fn init() {
        if let Err(error) = invaders::init() {
            invaders::report(error);
        }
    }
    async fn handle_action(action: String, params: String) {
        if let Err(error) = invaders::action(&action, &params).await {
            invaders::report(error);
        }
    }
}
impl exports::druid::openui::keyboard::Guest for SpaceInvaders {
    async fn handle_event(event: exports::druid::openui::keyboard::KeyEvent) {
        if event.region == "invaders" {
            if let Err(error) = invaders::key(event).await {
                invaders::report(error);
            }
        }
    }
}
export!(SpaceInvaders);

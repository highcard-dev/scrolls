-- Generic Sleep Handler
-- Simple handler that accepts any TCP connection and triggers server wake-up
-- Used by game servers without specific packet handlers

function handle(ctx, data)
    debug_print("Generic sleep handler: received connection")
    debug_print("Data length: " .. #data .. " bytes")
    
    -- Trigger server wake-up
    finish()
end

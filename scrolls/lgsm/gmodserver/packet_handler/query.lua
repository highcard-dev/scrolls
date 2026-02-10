-- Source Engine Query Protocol Handler
-- Handles A2S_INFO queries for Source engine games
-- Used by CS:GO, 7 Days to Die, Terraria, etc.

function handle(ctx, data)
    hex = string.tohex(data)
    
    debug_print("Query handler: received packet")
    debug_print("Hex: " .. hex)
    debug_print("Length: " .. #data .. " bytes")
    
    -- Check for Source Engine A2S_INFO query (0xFFFFFFFF54...)
    if #data >= 5 and data:byte(1) == 0xFF and data:byte(2) == 0xFF and 
       data:byte(3) == 0xFF and data:byte(4) == 0xFF and data:byte(5) == 0x54 then
        debug_print("Detected A2S_INFO query")
        
        -- Send a minimal response to keep client alive
        -- Format: 0xFFFFFFFF (header) + 0x49 (A2S_INFO response) + basic server info
        local response = string.char(
            0xFF, 0xFF, 0xFF, 0xFF, -- Header
            0x49, -- A2S_INFO response type
            0x11, -- Protocol version
            0x00 -- Null terminator for server name
        ) .. "Server Starting..." .. string.char(0x00) .. -- Server name
           "druid" .. string.char(0x00) .. -- Map name  
           "game" .. string.char(0x00) .. -- Folder
           "Game" .. string.char(0x00) .. -- Game description
            string.char(0x00, 0x00) .. -- App ID (2 bytes, 0)
            string.char(0x00) .. -- Players
            string.char(0x10) .. -- Max players
            string.char(0x00) .. -- Bots
            string.char(0x64) -- Server type (d = dedicated)
        
        sendData(response)
    end
    
    -- Trigger server wake-up
    finish()
end

function string.tohex(str)
    return (str:gsub('.', function(c)
        return string.format('%02X', string.byte(c))
    end))
end

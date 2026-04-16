-- WebRTC / STUN packet handler for Druid Voice Server
-- Detects STUN binding requests and ICE candidates

function is_stun_packet(data)
    if #data < 20 then
        return false
    end
    
    -- STUN packets start with 0x00 or 0x01 (message type)
    -- Check for STUN magic cookie: 0x2112A442
    local b1, b2, b3, b4 = string.byte(data, 5, 8)
    
    if b1 == 0x21 and b2 == 0x12 and b3 == 0xA4 and b4 == 0x42 then
        return true
    end
    
    return false
end

function is_dtls_packet(data)
    if #data < 13 then
        return false
    end
    
    -- DTLS content types: 20-26
    local content_type = string.byte(data, 1)
    
    if content_type >= 20 and content_type <= 26 then
        -- Check DTLS version (major.minor)
        local version = string.byte(data, 2) * 256 + string.byte(data, 3)
        -- DTLS 1.0: 0xFEFF, DTLS 1.2: 0xFEFD
        if version == 0xFEFF or version == 0xFEFD then
            return true
        end
    end
    
    return false
end

-- Main handler function called by ColdStarter
function handle(data, info)
    -- Check if this is a STUN packet (ICE connectivity)
    if is_stun_packet(data) then
        return {
            wake = true,
            reason = "STUN/ICE connection attempt"
        }
    end
    
    -- Check if this is a DTLS packet (WebRTC handshake)
    if is_dtls_packet(data) then
        return {
            wake = true,
            reason = "DTLS/WebRTC handshake"
        }
    end
    
    -- Unknown packet - don't wake
    return {
        wake = false
    }
end

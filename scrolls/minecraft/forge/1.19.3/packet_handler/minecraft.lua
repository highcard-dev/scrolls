json = require("json")

function string.fromhex(str)
    return (str:gsub('..', function(cc)
        return string.char(tonumber(cc, 16))
    end))
end

function string.tohex(str)
    return (str:gsub('.', function(c)
        return string.format('%02X', string.byte(c))
    end))
end

-- Bitwise AND
local function band(a, b)
    local result = 0
    local bitval = 1
    while a > 0 and b > 0 do
        local abit = a % 2
        local bbit = b % 2
        if abit == 1 and bbit == 1 then
            result = result + bitval
        end
        a = math.floor(a / 2)
        b = math.floor(b / 2)
        bitval = bitval * 2
    end
    return result
end

-- Bitwise OR
local function bor(a, b)
    local result = 0
    local bitval = 1
    while a > 0 or b > 0 do
        local abit = a % 2
        local bbit = b % 2
        if abit == 1 or bbit == 1 then
            result = result + bitval
        end
        a = math.floor(a / 2)
        b = math.floor(b / 2)
        bitval = bitval * 2
    end
    return result
end

-- Right Shift
local function rshift(value, shift)
    return math.floor(value / (2 ^ shift))
end

-- Left Shift
local function lshift(value, shift)
    return value * (2 ^ shift)
end

function encodeLEB128(value)
    local bytes = {}
    repeat
        local byte = band(value, 0x7F)
        value = rshift(value, 7)
        if value ~= 0 then
            byte = bor(byte, 0x80)
        end
        table.insert(bytes, byte)
    until value == 0
    return bytes
end

function handle(data)

    hex = string.tohex(data)

    -- check if hex starts with 0x01 0x00
    if hex:sub(1, 4) == "FE01" then
        debug_print("Received Legacy Ping Packet")
        sendData(string.fromhex(
            "ff002300a7003100000034003700000031002e0034002e0032000000410020004d0069006e006500630072006100660074002000530065007200760065007200000030000000320030"))
    elseif hex:sub(1, 4) == "0100" then
        debug_print("Received Status Packet")
        sendData(pingResponse())
        return
        -- check if second byte is 0x01
    elseif hex:sub(3, 4) == "01" then
        debug_print("Received Ping Packet")
        -- send same packet back
        close(data)
        return
        -- login packet 0x20 0x00
    elseif hex:sub(1, 4) == "2000" then
        debug_print("Received Login Packet")

        sendData(disconnectResponse())

        -- sleep for a sec before closing

        finish()
        return
    else
        debug_print("Received unknown packet")
        debug_print("String: " .. data)
        debug_print("Hex: " .. hex)
        return
    end

end

function formatResponse(jsonObj)
    local response = json.encode(jsonObj)
    local responseBuffer = {string.byte(response, 1, -1)}
    local additional = {0x00}
    local responseBufferLength = encodeLEB128(#responseBuffer)
    local packetLenthBuffer = encodeLEB128(#responseBuffer + #responseBufferLength + 1)

    local concatedBytes = {}

    for i = 1, #packetLenthBuffer do
        table.insert(concatedBytes, packetLenthBuffer[i])
    end

    for i = 1, #additional do
        table.insert(concatedBytes, additional[i])
    end

    for i = 1, #responseBufferLength do
        table.insert(concatedBytes, responseBufferLength[i])
    end

    for i = 1, #responseBuffer do
        table.insert(concatedBytes, responseBuffer[i])
    end

    -- convert back to string
    local finalString = string.char(unpack(concatedBytes))

    return finalString
end

function pingResponse()
    -- https://minecraft.fandom.com/wiki/Formatting_codes
    local messageList = {"🔴 Druid IDLE Minecraft", "To be or not to be 🤡", "Hi Marvin 💉", "Hi Adrian 💻",
                         "Hi Mark 📰", "§kKein Principle, weil es schon zu viele gibt"}

    realMessage = {
        color = "red",
        extra = {"\n", {
            color = "gray",
            extra = {{
                bold = true,
                text = "HINT"
            }, ":", " ", {
                color = "white",
                text = "Get free servers at:"
            }, " ", {
                color = "green",
                text = "druid.gg"
            }},
            text = ""
        }},
        text = "This server is in standby."
    }

    -- add realMessage to messageList

    table.insert(messageList, realMessage)

    local description = messageList[math.random(#messageList)]

    local obj = {
        version = {
            name = "§9🕐 Waiting...",
            protocol = -1
        },
        description = description,
        players = {
            max = 0,
            online = 1
        },
        favicon = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAEAAAABACAMAAACdt4HsAAAAAXNSR0IArs4c6QAAAMlQTFRFR3BM6ndq5Wxb3WBQ6HFi0EUvvVxI8IBzzTwm0EUv11RC3GBQ7X1w00w50EUv42pa1lRB3mNT4WZV0Ugz2VlH0ks22lpJ0ks332RU1VI/6XZo8oV4421e63Zn32JR0046ytvZ2FZEieHa5nBgb+fZFerZ1NrZDOrZDurZ1tjYQunZztrZO+jZruDZFOrZDOrZDOrZ6HVoDOrZ09rZ0cvJn+LZbebZi+PZkOPZC+rZ942B7Xpr9op98oR29Id67n1uz9vZH+rZjeTZHadAYQAAADl0Uk5TAOr9sP4WBv4CDXqV8kcf3m277CmGPaAzx1Pg8tD90lw3YxDx/mzTQ+aq/nYk/bT50NSS71SwxIbiWYkesQAABERJREFUeNqll2tfozgUxkshIeF+vxWoiNfRUaszuztDC7rf/0PtISAlpR1dfPLzTZLzz3POIUgXp0XD2PJUkGetfbT4fyJI9+xNsuqVbGx1beDPh7uKnazq7e+96lWSqj79XLihpKv691SrRPU/4YLGtsbCp9quNp5BPjreE1j4KYT9ZxPYDbQt7GObW9XwxxHqTUz/EB/a8hbC2+iVJpiRbUdpokE92RwbdVJQcjp+x3Ztay0N1iFClFLk6oqYMEa3thUKeqp74q7zLYjQdUzIgjBhGiqRBohOdaLjo/FIldm6FhWIEH4NG8pGHgiReywJagnd8eqwzCF0cTAhq/TIDt+stzAE79Rz76pAYKMW4ukZKJDr9nzldJcMIHSd3dloYiAWapCm8iu83ECrO00tIHEH87JojCfP78/O7u/x/pQw3bEcYCM9MKALANht9HH42d3Pn389PF9enw/bLNjWapf4vAUcyDCreaMGn91dfb/49gv09HxNegAS5ZohNIUHuGlrIHVH8bcv/0I40+MDEDoVYGEHkkXMZbAWYBIMjOJfIX7Qw3W/0YjkHSBqOTW4DFQNAElIhvxvX76z+MHDfU+AnUyJPwZQG7jjyv64er34NdbNZb/CvMJmYT0GGCkANAXvDbyCAU7vFkJTZgRNGQP8RAamTsYVeOPiH5/6KqD2LNiteWNALMCUaewBXAZcDjTtHajjJhSCLMvRtARTAAEAEwdYWABoRPwhgJWrkYcUeEAAgNMpPF0P5WLii7g+AJxzReS6AGcxCRZXxKQZAwi5ezlo4+Mz7i9NxeKbRB8DQrPhasD1kcsgTJsOwD/KKAcAdGGv9iq+jUvYG1AE2Amj4l8IWKyaxkRkNANJ7Ak3z+e9gahqmAT+OhMAN6VPRjOYvQ7euqfwso9HQdZ0Mn0eoJtVkymYmzu7vfrn4tvNDbxP+gWqJL0BlgF/HbPJJI5/3N39fXk5vBSRBcd0KteEBxClrCoz5Gf1IEYLMvBc7z2+ykQ0eWPnVVUqmLcV5J6PujnqFmJZNf0wdXIIwB5YyN3FQWWWqWrFuh4Xnlhm1btKDx/51xxl/QJPlcrSNM1SyqpBknjsQwdbZZWZOk81RKmaSLLDaTzrsVSVosFT/UiqMhhVto8/9ZlEQpYE5Qk6EDpl3XACLp7vu5llpoUPPKgOIDIIbSHLyOLy50ULJ5PMNTmoQ6zmzlICLR3bCunitAi1gJDH+MAZaj+7PU8pdJd+9I2ttIQ1nmRHEUIUk8WHQpYjSXlBF3NFaGFKkqkgMhtB41ySnMDFswlYt5fSMorpbBPEDRww4bl4LgKakbcm1gh/IY3WhKjPRhDDa004wXwE1kWzQxhzEciynRYhFuHcx8JQGGKZe7FLZ3a0RbB7qIRzERbUorURWWhuQ9Zq5CyXS0dBs++HbwU5EKwv3FJDh2rk/uILoqFlT38O/QdGyOZnTVzZRwAAAABJRU5ErkJggg=="
    }
    return formatResponse(obj)
end

function disconnectResponse()
    local obj = "Our super cool system will start now... please wait"
    return formatResponse(obj)
end
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

function pack_uint64_le(n)
    local bytes = {}
    for i = 1, 8 do
        bytes[i] = string.char(n % 256)
        n = math.floor(n / 256)
    end
    return table.concat(bytes)
end

function handle(ctx, data)

    -- prtocol begins with FFFFFFFF and the packedid

    -- get packet index

    -- check if start with FFFFFFFF

    hex = string.tohex(data)

    startOnUnknownPacket = get_var("StartOnUnknownPacket")
    if string.sub(hex, 1, 8) ~= "FFFFFFFF" then
        debug_print("Invalid Packet " .. hex)

        if startOnUnknownPacket == "yes" then
            print("Starting server on invalid packet: " .. hex)
            finish()
        end
        return
    end

    packetId = string.sub(hex, 9, 10)

    payload = string.sub(hex, 11)

    -- check if packet is 54

    debug_print("Packet ID: " .. packetId)

    if packetId == "55" then

        if payload == "FFFFFFFF" or payload == "00000000" then
            debug_print("Received Packet: " .. hex)
            resHex = string.fromhex("FFFFFFFF414BA1D522") -- this is not good, as we allways pass the same key for the challenge
            ctx.sendData(resHex)
            return
        end

        if payload == "4BA1D522" then
            debug_print("Received Packet: " .. hex)
            resHex = string.fromhex("FFFFFFFF4400") -- this is not good to be hardcoded, but fine for now

            ctx.sendData(resHex)
            return
        end
        debug_print("Bad challenge: " .. hex)
        return
    end

    if packetId == "56" then

        if payload == "FFFFFFFF" or payload == "00000000" then
            debug_print("Received Packet: " .. hex)
            resHex = string.fromhex("FFFFFFFF414BA1D522") -- this is not good, as we allways pass the same key for the challenge
            ctx.sendData(resHex)
            return
        end

        if payload == "4BA1D522" then
            debug_print("Received Packet: " .. hex)
            resHex = string.fromhex(
                "FFFFFFFF451A00414C4C4F57444F574E4C4F414443484152535F69003100414C4C4F57444F574E4C4F41444954454D535F69003100436C757374657249645F73004B4150323032326E76637738393233386E3332726677653900435553544F4D5345525645524E414D455F73006B617020707670202F20342D6D616E202F2078352D783235202F20776F726B65727320667269656E646C79207365727665720044617954696D655F730037360047616D654D6F64655F73005465737447616D654D6F64655F43004841534143544956454D4F44535F690031004C45474143595F690030004D4154434854494D454F55545F66003132302E303030303030004D4F44305F7300323839373838353837383A4544393730443545343845324143433334333545374339373345434135373637004D4F44315F7300323536343534363435353A3934413336414236343933453241443335364631343142313932383633453445004D4F44325F7300333034363539363536343A3832453245393730343446444139463642464237353439443730433337423133004D4F44335F7300313939393434373137323A3836453432424644343646453430363338443639344141384342453634344134004D6F6449645F6C0030004E6574776F726B696E675F690030004E554D4F50454E505542434F4E4E003530004F4646494349414C5345525645525F690030004F574E494E474944003930323032313035363131373133353337004F574E494E474E414D45003930323032313035363131373133353337005032504144445200393032303231303536313137313335333700503250504F52540037373837005345415243484B4559574F5244535F7300437573746F6D0053657276657250617373776F72645F620066616C73650053455256455255534553424154544C4559455F6200747275650053455353494F4E464C41475300313730370053455353494F4E49535056455F69003000") -- this is not good to be hardcoded, but fine for now

            ctx.sendData(resHex)
            return
        end
        debug_print("Bad challenge: " .. hex)
        return
    end

    if packetId == "54" then



        local snapshotMode = get_snapshot_mode()
        local snapshotPercentage = get_snapshot_percentage()


        queue = get_queue()
        name = get_var("ServerListName") or "Coldstarter is cool (server is idle, join to start)"

        map = get_var("MapName") or "server idle"

        local finishSec = get_finish_sec()

        if finishSec ~= nil then
            finishSec = math.ceil(finishSec)
        end

        if snapshotMode ~= "noop" then
            if snapshotMode == "restore" then
                if snapshotPercentage == nil or snapshotPercentage == 100 then
                    name = get_var("ServerListNameRestoring") or "EXTRACTING snapshot, this might take a moment"
                    map = get_var("MapNameRestoring") or "extracting snapshot"
                else
                    name = get_var("ServerListNameRestoring") or "DOWNLOADING snapshot - " .. string.format("%.2f", snapshotPercentage) .. "%"
                    map = get_var("MapNameRestoring") or "downloading snapshot"
                end
            else 
                if snapshotPercentage == nil or snapshotPercentage == 100 then
                    name = get_var("ServerListNameBackingUp") or "BACKING UP, this might take a moment"
                else
                    name = get_var("ServerListNameBackingUp") or "BACKING UP - " .. string.format("%.2f", snapshotPercentage) .. "%"
                end
                map = get_var("MapNameBackingUp") or "backing up server"
            end
        elseif queue ~= nil and queue["install"] == "running" then
            if finishSec ~= nil then
                -- finish sec is not necissary applicable, but it's better to show something I guess
                name = get_var("ServerListNameInstalling") or
                           string.format("INSTALLING, this might take a moment - %ds", finishSec)
            else
                name = get_var("ServerListNameInstalling") or "INSTALLING, this might take a moment"
            end

            map = get_var("MapNameInstalling") or "installing server"
        elseif finishSec ~= nil then
            nameTemplate = get_var("ServerListNameStarting") or "Druid Gameserver (starting) - %ds"
            name = string.format(nameTemplate, finishSec)
        end

        folder = get_var("GameSteamFolder") or "ark_survival_evolved"

        gameName = get_var("GameName") or "ARK: Survival Evolved"

        steamIdString = get_var("GameSteamId") or "0"
        gameVersion = get_var("GameVersion") or "1.0.0"

        steamId = tonumber(steamIdString)
        steamIdNum = tonumber(steamIdString)
        versionPrefix = get_var("GameVersionPrefix")
        serverPort = get_port("main")


        edfGameIdStr = get_var("SteamAppId")
        edfGameId = nil
        if edfGameIdStr ~= nil then
            edfGameId = tonumber(edfGameIdStr)
        end


        -- EDF & 0x80: Port
        -- EDF & 0x10: SteamID
        -- EDF & 0x20 Keywords
        -- EDF & 0x01 GameID

        edfSteamId = "4025ba0000003002"
        

        ---rust: "mp0,cp0,ptrak,qp0,$r?,v2592,born0,gmrust,cs1337420" 
        edfKeywords = get_var("GameKeywords") or ",OWNINGID:90202064633057281,OWNINGNAME:90202064633057281,NUMOPENPUBCONN:50,P2PADDR:90202064633057281,P2PPORT:" ..
                serverPort .. ",LEGACY_i:0"


        serverinfopacket = ServeInfoPacket:new()
        serverinfopacket.name = name
        serverinfopacket.map = map
        serverinfopacket.folder = folder
        serverinfopacket.gameName = gameName
        serverinfopacket.steamId = steamIdNum
        serverinfopacket.player = 0x00
        serverinfopacket.maxPlayer = 0x00
        serverinfopacket.bot = 0x00
        serverinfopacket.serverType = 0x64 -- 64 for dedicated server
        serverinfopacket.os = 0x6C -- 6C for linux, 77 for windows
        serverinfopacket.visibility = 0x00
        serverinfopacket.version = gameVersion
        if versionPrefix ~= nil then
            serverinfopacket.versionPrefix = versionPrefix
        else
            serverinfopacket.versionPrefix = nil
        end

        serverinfopacket.edfPort = serverPort
        serverinfopacket.edfSteamId = edfSteamId
        serverinfopacket.edfKeywords = edfKeywords
        serverinfopacket.edfGameId = edfGameId


        b = serverinfopacket:GetRawPacket()

        ctx.sendData(b)
        return
    end

    print("Unknown Packet: " .. hex)
    if startOnUnknownPacket == "yes" then
        print("Starting server on unknown packet: " .. hex)
        finish()
    end

end

function number_to_little_endian_short(num)
    -- Ensure the number is in the 16-bit range for unsigned short
    if num < 0 or num > 65535 then
        error("Number " .. num .. " out of range for 16-bit unsigned short")
    end

    -- Convert the number to two bytes in little-endian format
    local low_byte = num % 256 -- Least significant byte
    local high_byte = math.floor(num / 256) % 256 -- Most significant byte

    -- Format as hexadecimal string
    return string.format("%02X%02X", low_byte, high_byte)
end

Packet = {
  bytes = ""
}


function Packet:new (packetId)
    local o = {}
    setmetatable(o, self)
    self.__index = self
    o.bytes = string.fromhex("FFFFFFFF") .. packetId -- 0xFFFFFFFF + packetId
    return o
end

function Packet:appendString(data)
    self.bytes = self.bytes .. data .. string.char(0)
end

function Packet:appendByte(data)
    self.bytes = self.bytes .. string.char(data)
end

function Packet:appendRawBytes(data)
    self.bytes = self.bytes .. data
end

function Packet:appendShort(num)
    self.bytes = self.bytes .. string.fromhex(number_to_little_endian_short(num))
end

function Packet:appendHex(hex)
    self.bytes = self.bytes .. string.fromhex(hex)
end

ServeInfoPacket = {
    name = "",
    map = "",
    folder = "",
    gameName = "",
    steamId = 0,
    player = 0x00,
    maxPlayer = 0x00,
    bot = 0x00,
    serverType = 0x64,
    os = 0x6C, -- 6C for linux, 77 for windows
    visibility = 0x00, -- 01 for private, 00 for public
    version = "1.0.0",
    versionPrefix = nil,
    edfPort = nil,
    edfSteamId = nil,
    edfSourceTv = nil,
    edfKeywords = nil,
    edfGameId = nil
}

function ServeInfoPacket:new ()
   o = {}
   setmetatable(o, self)
   self.__index = self
   return o
end


function ServeInfoPacket:GetRawPacket()

    p = Packet:new(string.fromhex("4911")) -- 0x49 0x11 is the packet id for server info
    p:appendString(self.name)
    p:appendString(self.map)
    p:appendString(self.folder)
    p:appendString(self.gameName)
    p:appendShort(self.steamId)
    p:appendByte(self.player)
    p:appendByte(self.maxPlayer)
    p:appendByte(self.bot)
    p:appendByte(self.serverType)
    p:appendByte(self.os)
    p:appendByte(self.visibility) -- 01 for private, 00 for public
    --p:appendHex("01323032352E30332E323600")
    if self.versionPrefix ~= nil then
        debug_print("Using version prefix: " .. self.versionPrefix)
        p:appendHex(self.versionPrefix)
    end
    p:appendString(self.version)
    debug_print(string.tohex(p.bytes))

    edfByte = 0x00

    if self.edfPort ~= nil then
        edfByte = edfByte + 0x80
    end
    if self.edfSteamId ~= nil then
        edfByte = edfByte + 0x10
    end
    if self.edfSourceTv ~= nil then
        edfByte = edfByte + 0x40
    end
    if self.edfKeywords ~= nil then
        edfByte = edfByte + 0x20
    end
    if self.edfGameId ~= nil then
        edfByte = edfByte + 0x01
    end

    p:appendByte(edfByte)

    if self.edfPort ~= nil then
        p:appendShort(self.edfPort)
    end
    if self.edfSteamId ~= nil then
        p:appendHex(self.edfSteamId)
    end
    if self.edfSourceTv ~= nil then
        p:appendHex(self.edfSourceTv)
    end
    if self.edfKeywords ~= nil then
        p:appendString(self.edfKeywords)
    end
    if self.edfGameId ~= nil then
        local bytes = pack_uint64_le(self.edfGameId)
        p:appendRawBytes(bytes)
    end

    
    return p.bytes
end

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	cp "github.com/otiai10/copy"
	"gopkg.in/yaml.v3"
)

type manifest struct {
	Version int          `json:"version"`
	Server  serverSchema `json:"server"`
	Files   []fileSchema `json:"files"`
}

type serverSchema struct {
	Family      string `json:"family"`
	DisplayName string `json:"displayName"`
	AppVersion  string `json:"appVersion,omitempty"`
}

type fileSchema struct {
	Path          string          `json:"path"`
	Format        string          `json:"format"`
	Label         string          `json:"label"`
	Description   string          `json:"description,omitempty"`
	Documentation string          `json:"documentation,omitempty"`
	Sections      []sectionSchema `json:"sections"`
}

type sectionSchema struct {
	ID          string        `json:"id"`
	Label       string        `json:"label"`
	Description string        `json:"description,omitempty"`
	Fields      []fieldSchema `json:"fields"`
}

type fieldSchema struct {
	Key             string   `json:"key"`
	Label           string   `json:"label"`
	Description     string   `json:"description"`
	Documentation   string   `json:"documentation"`
	Type            string   `json:"type"`
	Values          []string `json:"values,omitempty"`
	Min             *float64 `json:"min,omitempty"`
	Max             *float64 `json:"max,omitempty"`
	Sensitive       bool     `json:"sensitive,omitempty"`
	RestartRequired bool     `json:"restartRequired,omitempty"`
}

func number(value float64) *float64 { return &value }

func field(key, label, kind, description string) fieldSchema {
	return fieldSchema{
		Key: key, Label: label, Type: kind, Description: description,
		Documentation: "https://minecraft.wiki/w/Server.properties", RestartRequired: true,
	}
}

func documentedField(key, label, kind, description, documentation string) fieldSchema {
	result := field(key, label, kind, description)
	result.Documentation = documentation
	return result
}

func arkManifest(version string) manifest {
	const docs = "https://ark.wiki.gg/wiki/Server_configuration"
	server := []fieldSchema{
		documentedField("ServerSettings.ShowMapPlayerLocation", "Show players on map", "boolean", "Show each player's position on their map.", docs),
		documentedField("ServerSettings.AllowThirdPersonPlayer", "Allow third person", "boolean", "Allow players to use the third-person camera.", docs),
		documentedField("ServerSettings.ServerCrosshair", "Server crosshair", "boolean", "Enable the server-side crosshair option.", docs),
		documentedField("ServerSettings.ServerPassword", "Join password", "secret", "Password required to join; leave empty for a public server.", docs),
		documentedField("ServerSettings.ServerAdminPassword", "Admin password", "secret", "Password used for in-game administrator access.", docs),
		documentedField("ServerSettings.RCONEnabled", "Enable RCON", "boolean", "Enable remote console administration.", docs),
		documentedField("ServerSettings.KickIdlePlayersPeriod", "Idle kick seconds", "number", "Seconds before an idle player is removed.", docs),
		documentedField("ServerSettings.AutoSavePeriodMinutes", "Autosave interval", "number", "Minutes between automatic saves.", docs),
		documentedField("ServerSettings.MaxTamedDinos", "Maximum tamed dinos", "integer", "Global tamed dinosaur limit.", docs),
		documentedField("ServerSettings.ItemStackSizeMultiplier", "Item stack multiplier", "number", "Multiplier applied to normal item stack sizes.", docs),
	}
	server[3].Sensitive, server[4].Sensitive = true, true
	server[6].Min, server[7].Min, server[8].Min, server[9].Min = number(0), number(0), number(0), number(0)

	transfers := []fieldSchema{}
	for _, item := range []struct{ key, label, description string }{
		{"noTributeDownloads", "Disable tribute downloads", "Disable all CrossARK tribute downloads."},
		{"PreventDownloadDinos", "Prevent dino downloads", "Prevent downloading uploaded creatures."},
		{"PreventDownloadItems", "Prevent item downloads", "Prevent downloading uploaded items."},
		{"PreventDownloadSurvivors", "Prevent survivor downloads", "Prevent downloading uploaded survivors."},
		{"PreventUploadDinos", "Prevent dino uploads", "Prevent uploading creatures."},
		{"PreventUploadItems", "Prevent item uploads", "Prevent uploading items."},
		{"PreventUploadSurvivors", "Prevent survivor uploads", "Prevent uploading survivors."},
	} {
		transfers = append(transfers, documentedField("ServerSettings."+item.key, item.label, "boolean", item.description, docs))
	}

	gameplay := []fieldSchema{
		documentedField("/script/shootergame.shootergamemode.DifficultyOffset", "Difficulty offset", "number", "Base world difficulty offset.", docs),
		documentedField("/script/shootergame.shootergamemode.OverrideOfficialDifficulty", "Official difficulty override", "number", "Override the maximum wild creature difficulty.", docs),
		documentedField("/script/shootergame.shootergamemode.XPMultiplier", "XP multiplier", "number", "Multiplier for experience gains.", docs),
		documentedField("/script/shootergame.shootergamemode.TamingSpeedMultiplier", "Taming speed", "number", "Multiplier for taming progress.", docs),
		documentedField("/script/shootergame.shootergamemode.HarvestAmountMultiplier", "Harvest amount", "number", "Multiplier for harvested resource amounts.", docs),
		documentedField("/script/shootergame.shootergamemode.HarvestHealthMultiplier", "Harvest health", "number", "Multiplier for resource-node health.", docs),
		documentedField("/script/shootergame.shootergamemode.MatingIntervalMultiplier", "Mating interval", "number", "Multiplier for the delay between mating attempts.", docs),
		documentedField("/script/shootergame.shootergamemode.EggHatchSpeedMultiplier", "Egg hatch speed", "number", "Multiplier for egg incubation speed.", docs),
		documentedField("/script/shootergame.shootergamemode.BabyMatureSpeedMultiplier", "Baby maturation speed", "number", "Multiplier for baby creature maturation.", docs),
		documentedField("/script/shootergame.shootergamemode.ResourcesRespawnPeriodMultiplier", "Resource respawn period", "number", "Multiplier for resource respawn time.", docs),
		documentedField("/script/shootergame.shootergamemode.bAllowUnlimitedRespecs", "Unlimited respecs", "boolean", "Allow repeated Mindwipe respecs.", docs),
		documentedField("/script/shootergame.shootergamemode.bUseSingleplayerSettings", "Single-player scaling", "boolean", "Apply the additional single-player balancing multipliers.", docs),
	}
	for index := 0; index < 10; index++ {
		gameplay[index].Min = number(0)
	}

	return manifest{Version: 1, Server: serverSchema{Family: "ark", DisplayName: "ARK: Survival Evolved", AppVersion: version}, Files: []fileSchema{
		{
			Path: "data/serverfiles/ShooterGame/Saved/Config/LinuxServer/GameUserSettings.ini", Format: "unreal-ini", Label: "GameUserSettings.ini",
			Description: "ARK server access, administration, and transfer settings. Every additional option remains editable in Raw mode.", Documentation: docs,
			Sections: []sectionSchema{
				{ID: "server", Label: "Server and administration", Fields: server},
				{ID: "transfers", Label: "CrossARK transfers", Fields: transfers},
				{ID: "session", Label: "Session", Fields: []fieldSchema{
					documentedField("SessionSettings.SessionName", "Session name", "string", "Name advertised in the ARK and Steam server browsers.", docs),
					documentedField("/Script/Engine.GameSession.MaxPlayers", "Maximum players", "integer", "Maximum simultaneous players.", docs),
				}},
			},
		},
		{
			Path: "data/serverfiles/ShooterGame/Saved/Config/LinuxServer/Game.ini", Format: "unreal-ini", Label: "Game.ini",
			Description: "ARK progression, breeding, harvesting, and balance settings. Advanced arrays and overrides remain editable in Raw mode.", Documentation: docs,
			Sections: []sectionSchema{{ID: "gameplay", Label: "World and progression", Fields: gameplay}},
		},
	}}
}

func minecraftManifest(version string) manifest {
	general := []fieldSchema{
		field("motd", "Message of the day", "string", "Text shown in the multiplayer server list."),
		field("server-ip", "Bind address", "string", "Address the server binds to; leave empty for all interfaces."),
		field("server-port", "Server port", "integer", "TCP game port."),
		field("max-players", "Maximum players", "integer", "Maximum simultaneous players."),
		field("online-mode", "Online mode", "boolean", "Verify players with Minecraft account services."),
		field("prevent-proxy-connections", "Prevent proxy connections", "boolean", "Reject connections whose ISP or ASN differs from authentication data."),
		field("white-list", "Allowlist", "boolean", "Only allow players in the allowlist."),
		field("enforce-whitelist", "Enforce allowlist", "boolean", "Remove non-allowlisted online players when the allowlist reloads."),
		field("enable-status", "Server-list status", "boolean", "Answer server-list status requests."),
		field("hide-online-players", "Hide online players", "boolean", "Hide the player sample in status replies."),
	}
	general[2].Min, general[2].Max = number(1), number(65535)
	general[3].Min, general[3].Max = number(1), number(100000)

	world := []fieldSchema{
		field("level-name", "World name", "string", "World directory name."),
		field("level-seed", "World seed", "string", "Seed used when generating a new world."),
		field("level-type", "World type", "string", "World-generation preset or type."),
		field("generator-settings", "Generator settings", "string", "JSON world-generator settings."),
		field("generate-structures", "Generate structures", "boolean", "Generate villages and other structures."),
		field("difficulty", "Difficulty", "enum", "Default world difficulty."),
		field("gamemode", "Game mode", "enum", "Default game mode."),
		field("force-gamemode", "Force game mode", "boolean", "Force the default game mode when players join."),
		field("hardcore", "Hardcore", "boolean", "Enable hardcore mode."),
		field("pvp", "Player versus player", "boolean", "Allow players to damage each other."),
		field("allow-nether", "Allow Nether", "boolean", "Allow travel to the Nether."),
		field("allow-flight", "Allow flight", "boolean", "Do not kick players detected as flying."),
		field("spawn-animals", "Spawn animals", "boolean", "Enable passive animal spawning."),
		field("spawn-monsters", "Spawn monsters", "boolean", "Enable hostile mob spawning."),
		field("spawn-npcs", "Spawn NPCs", "boolean", "Enable villager spawning."),
		field("spawn-protection", "Spawn protection", "integer", "Protected radius around world spawn."),
		field("max-world-size", "Maximum world size", "integer", "Maximum world border radius."),
	}
	world[5].Values = []string{"peaceful", "easy", "normal", "hard"}
	world[6].Values = []string{"survival", "creative", "adventure", "spectator"}
	world[15].Min, world[15].Max = number(0), number(29999984)
	world[16].Min, world[16].Max = number(1), number(29999984)

	performance := []fieldSchema{
		field("view-distance", "View distance", "integer", "Server-side chunk view distance."),
		field("simulation-distance", "Simulation distance", "integer", "Chunk distance in which entities and ticks are simulated."),
		field("max-tick-time", "Maximum tick time", "integer", "Watchdog threshold in milliseconds; -1 disables it."),
		field("network-compression-threshold", "Compression threshold", "integer", "Packet size at which network compression begins."),
		field("entity-broadcast-range-percentage", "Entity broadcast range", "integer", "Percentage multiplier for entity broadcast distance."),
		field("player-idle-timeout", "Idle timeout", "integer", "Minutes before an idle player is kicked; 0 disables it."),
		field("rate-limit", "Packet rate limit", "integer", "Maximum packets per second before a client is kicked; 0 disables it."),
		field("sync-chunk-writes", "Synchronous chunk writes", "boolean", "Write chunk data synchronously."),
		field("use-native-transport", "Native transport", "boolean", "Use Linux native packet transport when available."),
	}
	performance[0].Min, performance[0].Max = number(2), number(32)
	performance[1].Min, performance[1].Max = number(2), number(32)
	performance[3].Min = number(-1)
	performance[4].Min, performance[4].Max = number(10), number(1000)
	performance[5].Min, performance[6].Min = number(0), number(0)

	admin := []fieldSchema{
		field("enable-rcon", "Enable RCON", "boolean", "Enable remote console access."),
		field("rcon.port", "RCON port", "integer", "Remote console TCP port."),
		field("rcon.password", "RCON password", "secret", "Remote console password."),
		field("broadcast-rcon-to-ops", "Broadcast RCON to operators", "boolean", "Show RCON command output to operators."),
		field("enable-query", "Enable query", "boolean", "Enable the GameSpy4 query protocol."),
		field("query.port", "Query port", "integer", "GameSpy4 query UDP port."),
		field("enable-command-block", "Command blocks", "boolean", "Enable command blocks."),
		field("broadcast-console-to-ops", "Broadcast console to operators", "boolean", "Show console command output to operators."),
		field("function-permission-level", "Function permission level", "integer", "Permission level used by functions."),
		field("op-permission-level", "Operator permission level", "integer", "Default permission level for operators."),
	}
	admin[1].Min, admin[1].Max = number(1), number(65535)
	admin[2].Sensitive = true
	admin[5].Min, admin[5].Max = number(1), number(65535)
	admin[8].Min, admin[8].Max = number(1), number(4)
	admin[9].Min, admin[9].Max = number(1), number(4)

	return manifest{Version: 1, Server: serverSchema{Family: "minecraft", DisplayName: "Minecraft Server", AppVersion: version}, Files: []fileSchema{{
		Path: "data/server.properties", Format: "java-properties", Label: "server.properties",
		Description:   "Minecraft dedicated-server settings. Version-specific and unknown keys remain editable in Raw mode.",
		Documentation: "https://minecraft.wiki/w/Server.properties",
		Sections: []sectionSchema{
			{ID: "general", Label: "Server access", Fields: general},
			{ID: "world", Label: "World and gameplay", Fields: world},
			{ID: "performance", Label: "Performance and networking", Fields: performance},
			{ID: "administration", Label: "Administration", Fields: admin},
		},
	}}}
}

func rawFile(path, label, format, description, documentation string) fileSchema {
	return fileSchema{Path: path, Label: label, Format: format, Description: description, Documentation: documentation, Sections: []sectionSchema{}}
}

func familyManifest(source string, appVersion string) (manifest, error) {
	source = filepath.ToSlash(source)
	if strings.Contains(source, "/minecraft/") {
		if strings.Contains(source, "/cuberite/") {
			return manifest{1, serverSchema{"minecraft-cuberite", "Cuberite Server", appVersion}, []fileSchema{
				rawFile("data/settings.ini", "settings.ini", "ini", "Cuberite server settings. All settings remain editable in Raw mode.", "https://book.cuberite.org/#2.2"),
			}}, nil
		}
		return minecraftManifest(appVersion), nil
	}

	type family struct{ name, file, format, docs string }
	families := map[string]family{
		"arkserver":  {"ARK: Survival Evolved", "data/serverfiles/ShooterGame/Saved/Config/LinuxServer/GameUserSettings.ini", "unreal-ini", "https://ark.wiki.gg/wiki/Server_configuration"},
		"cs2server":  {"Counter-Strike 2", "data/lgsm/config-lgsm/cs2server/cs2server.cfg", "key-value", "https://docs.linuxgsm.com/configuration/game-server-config"},
		"csgoserver": {"Counter-Strike: Global Offensive", "data/lgsm/config-lgsm/csgoserver/csgoserver.cfg", "key-value", "https://docs.linuxgsm.com/configuration/game-server-config"},
		"dayzserver": {"DayZ", "data/lgsm/config-lgsm/dayzserver/common.cfg", "key-value", "https://community.bistudio.com/wiki/DayZ:Server_Configuration"},
		"gmodserver": {"Garry's Mod", "data/lgsm/config-lgsm/gmodserver/gmodserver.cfg", "key-value", "https://docs.linuxgsm.com/configuration/game-server-config"},
		"pwserver":   {"Palworld", "data/serverfiles/Pal/Saved/Config/LinuxServer/PalWorldSettings.ini", "unreal-ini", "https://tech.palworldgame.com/settings-and-operation/configuration/"},
		"pzserver":   {"Project Zomboid", "data/serverfiles/Zomboid/Server/servertest.ini", "ini", "https://pzwiki.net/wiki/Server_settings"},
		"sdtdserver": {"7 Days to Die", "data/serverfiles/serverconfig.xml", "raw", "https://developer.valvesoftware.com/wiki/7_Days_to_Die_Dedicated_Server"},
		"untserver":  {"Unturned", "data/config-lgsm/untserver/untserver.cfg", "key-value", "https://docs.linuxgsm.com/configuration/game-server-config"},
	}
	for id, spec := range families {
		if strings.Contains(source, "/lgsm/"+id) {
			if id == "arkserver" {
				return arkManifest(appVersion), nil
			}
			files := []fileSchema{rawFile(spec.file, filepath.Base(spec.file), spec.format, spec.name+" configuration. Typed coverage can grow without losing access to any option in Raw mode.", spec.docs)}
			return manifest{1, serverSchema{"lgsm-" + id, spec.name, appVersion}, files}, nil
		}
	}
	if strings.Contains(source, "/rust/rust-") {
		return manifest{1, serverSchema{"rust", "Rust Server", appVersion}, []fileSchema{
			rawFile("data/server/druid/cfg/server.cfg", "server.cfg", "key-value", "Rust server convars. Every convar remains editable in Raw mode.", "https://wiki.facepunch.com/rust/Creating-a-server"),
		}}, nil
	}
	if strings.Contains(source, "/hytale/") {
		return manifest{1, serverSchema{"hytale", "Hytale Server", appVersion}, []fileSchema{
			rawFile("data/config.json", "config.json", "json", "Hytale server configuration.", "https://support.hytale.com/"),
		}}, nil
	}
	return manifest{}, fmt.Errorf("no configuration UI catalog entry for %s", source)
}

func stage(source, destination, bundle string) error {
	if source == "" || destination == "" || bundle == "" {
		return errors.New("source, destination, and bundle are required")
	}
	if _, err := os.Stat(filepath.Join(source, "scroll.yaml")); err != nil {
		return fmt.Errorf("source scroll.yaml: %w", err)
	}
	if err := cp.Copy(source, destination); err != nil {
		return fmt.Errorf("copy Scroll: %w", err)
	}

	scrollPath := filepath.Join(destination, "scroll.yaml")
	scrollBytes, err := os.ReadFile(scrollPath)
	if err != nil {
		return err
	}
	var scroll map[string]any
	if err := yaml.Unmarshal(scrollBytes, &scroll); err != nil {
		return fmt.Errorf("parse scroll.yaml: %w", err)
	}
	appVersion, _ := scroll["app_version"].(string)
	configManifest, err := familyManifest(filepath.ToSlash(source), appVersion)
	if err != nil {
		return err
	}
	scroll["ui"] = map[string]any{"private": map[string]any{"path": "private/dist/app.wasm"}}
	updatedYAML, err := yaml.Marshal(scroll)
	if err != nil {
		return err
	}
	if err := os.WriteFile(scrollPath, updatedYAML, 0644); err != nil {
		return err
	}

	privateDir := filepath.Join(destination, "private")
	if err := os.MkdirAll(filepath.Join(privateDir, "dist"), 0755); err != nil {
		return err
	}
	if err := copyFile(bundle, filepath.Join(privateDir, "dist", "app.wasm")); err != nil {
		return err
	}
	manifestBytes, err := json.MarshalIndent(configManifest, "", "  ")
	if err != nil {
		return err
	}
	manifestBytes = append(manifestBytes, '\n')
	if err := os.WriteFile(filepath.Join(privateDir, "config-editor.manifest.json"), manifestBytes, 0644); err != nil {
		return err
	}
	return ensureConfigFiles(destination, configManifest)
}

func ensureConfigFiles(destination string, configManifest manifest) error {
	for _, file := range configManifest.Files {
		target := filepath.Join(destination, filepath.FromSlash(file.Path))
		if _, err := os.Stat(target); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			return err
		}
		if _, err := os.Stat(target + ".scroll_template"); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		content := ""
		if file.Format == "json" {
			content = "{}\n"
		}
		if strings.Contains(filepath.ToSlash(target), "/server/druid/cfg/server.cfg") {
			content = "// Every Rust server convar is supported here.\nserver.hostname Druid Rust Server\nserver.description A server hosted on druid.gg\nserver.maxplayers 75\nserver.worldsize 1000\nserver.saveinterval 300\nserver.globalchat true\n"
		}
		if err := os.WriteFile(target, []byte(content), 0644); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(source, destination string) error {
	in, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("open UI bundle: %w", err)
	}
	defer in.Close()
	out, err := os.Create(destination)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func main() {
	if len(os.Args) != 4 {
		fmt.Fprintln(os.Stderr, "usage: go run ./scripts/stage-scroll-ui <source-scroll> <destination> <app.wasm>")
		os.Exit(2)
	}
	if err := stage(os.Args[1], os.Args[2], os.Args[3]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

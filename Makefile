build-tree:
	go run generate-scrolls.go ./scrolls/lgsm
	go run generate-scrolls.go ./scrolls/minecraft/forge
	go run generate-scrolls.go ./scrolls/minecraft/minecraft-spigot
	go run generate-scrolls.go ./scrolls/minecraft/minecraft-vanilla
	go run generate-scrolls.go ./scrolls/minecraft/papermc

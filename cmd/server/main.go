package main

import "honnef.co/go/tools/config"

func main() {
	_ = godotenv.Load()
	cfg, err := config.Load()

}

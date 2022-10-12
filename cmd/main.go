package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/sleep2death/dream"
	"github.com/spf13/viper"
)

func main() {
	// set default configs and read
	dream.Config()

	// set gin's running mode
	gin.SetMode(viper.GetString("mode"))

	r := gin.Default()
	dream.Setup(r)

	if err := r.Run(viper.GetString("apiAddr")); err != nil {
		log.Fatal(err)
	}
}

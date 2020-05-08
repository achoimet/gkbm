package main

import (
	"github.com/gofiber/fiber"
)

func main() {
	app := fiber.New()

	app.Get("/", func(c *fiber.Ctx) {
		c.Send("Hello, World!")
	})

	err := app.Listen(3000)
	if err != nil {
		panic(err.Error())
	}
}

package main

import (
	"log"

	"github.com/BTDevelop/go-cdn/handler"
	"github.com/BTDevelop/go-cdn/service"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/favicon"
)

var (
	minioClient = service.MinioClient()
	fileHandler = handler.NewFile(minioClient)
)

func main() {

	app := fiber.New(fiber.Config{
		BodyLimit: 25 * 1024 * 2014,
	})

	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "*",
	}))

	app.Static("/", "./public")

	app.Use(favicon.New(favicon.Config{
		File: "./public/favicon.png",
	}))

	// Minio
	app.Get("/:bucket/*", fileHandler.GetFile)

	app.Delete("delete", fileHandler.DeleteImage)

	app.Post("/upload", fileHandler.UploadImage)

	app.Post("/upload-url", fileHandler.UploadImageWithUrl)

	app.Post("/resize", fileHandler.ResizeImage)

	// Index
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendFile("index.html")
	})

	log.Fatal(app.Listen(":9090"))
}

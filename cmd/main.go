package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/favicon"
	"github.com/mstgnz/go-minio-cdn/handler"
	"github.com/mstgnz/go-minio-cdn/service"
)

var (
	// awsService   = service.NewAwsService()
	minioClient  = service.MinioClient()
	fileHandler = handler.NewFile(minioClient)
	// awsHandler   = handler.NewAwsHandler(awsService)
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

	// Aws
	// app.Get("/aws/bucket-list", awsHandler.BucketList)
	// app.Get("/aws/get-vault-list", awsHandler.GlacierVaultList)

	// Minio
	app.Get("/:bucket/*", fileHandler.GetFile)

	app.Delete("delete", fileHandler.DeleteImage)
	// app.Delete("delete-with-aws", fileHandler.DeleteImageWithAws)

	app.Post("/upload", fileHandler.UploadImage)
	// app.Post("/upload-with-aws", fileHandler.UploadImageWithAws)
	app.Post("/upload-url", fileHandler.UploadImageWithUrl)

	app.Post("/resize", fileHandler.ResizeImage)

	// Index
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendFile("index.html")
	})

	log.Fatal(app.Listen(":9090"))

}

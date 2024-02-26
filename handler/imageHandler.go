// Package handler /*
/*
## License
This project is licensed under the APACHE Licence. Refer to github.com/BTDevelop/go-cdn/blob/main/LICENSE for more information.
*/
package handler

import (
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/BTDevelop/go-cdn/service"
	"github.com/gofiber/fiber/v2"
	"github.com/minio/minio-go/v7"
)

type File interface {
	GetFile(c *fiber.Ctx) error
	GetImage(c *fiber.Ctx) error
	UploadImage(c *fiber.Ctx) error
	ResizeImage(c *fiber.Ctx) error
	DeleteImage(c *fiber.Ctx) error
	UploadImageWithUrl(c *fiber.Ctx) error
}

type file struct {
	minioService minio.Client
}

func NewFile(minioService *minio.Client) File {
	return &file{
		minioService: *minioService,
	}
}

func (i file) GetFile(c *fiber.Ctx) error {
	ctx := context.Background()
	bucket := c.Params("bucket")
	objectName := c.Params("*")

	// Bucket check
	if found, err := i.minioService.BucketExists(ctx, bucket); !found || err != nil {
		return c.SendFile("./public/notfound.png") // Veya daha genel bir hata mesajı
	}

	object, err := i.minioService.GetObject(ctx, bucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		return c.SendFile("./public/notfound.png") // Veya daha genel bir hata mesajı
	}

	byteContent := service.StreamToByte(object)
	if len(byteContent) == 0 {
		return c.SendFile("./public/notfound.png") // Veya daha genel bir hata mesajı
	}

	contentType := http.DetectContentType(byteContent)
	c.Set("Content-Type", contentType)
	c.Set("Cache-Control", "max-age=2592000, public")

	return c.Send(byteContent)
}

func (i file) GetImage(c *fiber.Ctx) error {
	ctx := context.Background()

	width := 0
	height := 0
	resize := false
	bucket := c.Params("bucket")
	objectName := c.Params("*")

	obj := strings.Split(objectName, "/")

	if len(obj) >= 3 {
		getWith, wErr := strconv.Atoi(obj[0])
		getHeight, hErr := strconv.Atoi(obj[1])

		if wErr == nil && hErr == nil {
			resize = true
			width = getWith
			height = getHeight
			objectName = strings.Join(obj[2:], "/")
		}
	}

	// Bucket exists
	if found, err := i.minioService.BucketExists(ctx, bucket); !found || err != nil {
		return c.SendFile("./public/notfound.png")
	}

	// Get Object
	object, err := i.minioService.GetObject(ctx, bucket, objectName, minio.GetObjectOptions{})

	if err != nil {
		return c.SendFile("./public/notfound.png")
	}

	// Convert Byte
	getByte := service.StreamToByte(object)
	if len(getByte) == 0 {
		return c.SendFile("./public/notfound.png")
	}

	// Set Content Type
	c.Set("Content-Type", http.DetectContentType(getByte))

	// Send Resized Image
	if resize {
		return c.Send(service.ImagickResize(getByte, uint(width), uint(height)))
	}

	// Send Original Image
	return c.Send(getByte)
}

func (i file) DeleteImage(c *fiber.Ctx) error {

	ctx := context.Background()

	getToken := strings.Split(c.Get("Authorization"), " ")
	if len(getToken) != 2 || !strings.EqualFold(getToken[1], service.GetEnv("TOKEN")) {
		return c.JSON(fiber.Map{
			"status":  false,
			"message": "Invalid Token",
		})
	}

	bucket := c.FormValue("bucket")
	object := c.FormValue("object")

	if len(bucket) == 0 || len(object) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  false,
			"message": "invalid path or bucket or file.",
		})
	}

	// Minio Bucket Exists
	if found, _ := i.minioService.BucketExists(ctx, bucket); !found {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  false,
			"message": "Bucket Not Found On Minio!",
		})
	}

	err := i.minioService.RemoveObject(ctx, bucket, object, minio.RemoveObjectOptions{})
	if err != nil {
		return c.JSON(fiber.Map{
			"status":  false,
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"status":  true,
		"message": "File Successfully Deleted",
	})
}

func (i file) UploadImage(c *fiber.Ctx) error {
	ctx := context.Background()

	getToken := strings.Split(c.Get("Authorization"), " ")
	if len(getToken) != 2 || !strings.EqualFold(getToken[1], service.GetEnv("TOKEN")) {
		return c.JSON(fiber.Map{
			"status":  false,
			"message": "Invalid Token",
		})
	}

	path := c.FormValue("path")
	bucket := c.FormValue("bucket")
	file, err := c.FormFile("file")

	if file == nil || err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  false,
			"message": "File Not Found!",
		})
	}

	if len(path) == 0 || len(bucket) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  false,
			"message": "invalid path or bucket or file.",
		})
	}

	// Check to see if already exist bucket
	exists, err := i.minioService.BucketExists(ctx, bucket)
	if err != nil && !exists {
		// Bucket not found so Make a new bucket
		err = i.minioService.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  false,
				"message": "Bucket Not Found And Not Created!",
			})
		}
	}

	// Get Buffer from file
	fileBuffer, err := file.Open()
	defer func(fileBuffer multipart.File) {
		_ = fileBuffer.Close()
	}(fileBuffer)

	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  false,
			"message": err.Error(),
		})
	}

	parseFileName := strings.Split(file.Filename, ".")

	if len(parseFileName) < 2 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  false,
			"message": "File extension not found!",
		})
	}

	randomName := service.RandomName(10)
	imageName := randomName + "." + parseFileName[len(parseFileName)-1]
	objectName := path + "/" + imageName
	contentType := file.Header["Content-Type"][0]
	fileSize := file.Size

	// Minio Upload
	_, err = i.minioService.PutObject(ctx, bucket, objectName, fileBuffer, fileSize, minio.PutObjectOptions{ContentType: contentType})
	minioResult := "Minio Successfully Uploaded"

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  false,
			"message": err.Error(),
		})
	}

	url := service.GetEnv("PROJECT_ENDPOINT")
	url = strings.TrimSuffix(url, "/")
	link := url + "/" + bucket + "/" + objectName

	return c.JSON(fiber.Map{
		"status":      true,
		"minioResult": minioResult,
		"imageName":   imageName,
		"objectName":  objectName,
		"link":        link,
	})
}

func (i file) ResizeImage(c *fiber.Ctx) error {

	getToken := strings.Split(c.Get("Authorization"), " ")
	if len(getToken) != 2 || !strings.EqualFold(getToken[1], service.GetEnv("TOKEN")) {
		return c.JSON(fiber.Map{
			"status":  false,
			"message": "Invalid Token",
		})
	}

	width := c.FormValue("width")
	height := c.FormValue("height")
	file, err := c.FormFile("file")

	if file == nil || err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  false,
			"message": "File Not Found!",
		})
	}

	width, height = service.SetWidthToHeight(width, height)
	hWidth, wErr := strconv.ParseUint(width, 10, 16)

	hHeight, hErr := strconv.ParseUint(height, 10, 16)

	if wErr != nil || hErr != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  false,
			"message": "width or height invalid!",
		})
	}

	fileBuffer, err := file.Open()
	defer func(fileBuffer multipart.File) {
		_ = fileBuffer.Close()
	}(fileBuffer)

	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  false,
			"message": err.Error(),
		})
	}

	c.Set("Content-Type", http.DetectContentType(service.StreamToByte(fileBuffer)))
	return c.Send(service.ImagickResize(service.StreamToByte(fileBuffer), uint(hWidth), uint(hHeight)))
}

func (i file) UploadImageWithUrl(c *fiber.Ctx) error {
	ctx := context.Background()

	getToken := strings.Split(c.Get("Authorization"), " ")
	if len(getToken) != 2 || !strings.EqualFold(getToken[1], service.GetEnv("TOKEN")) {
		return c.JSON(fiber.Map{
			"error": true,
			"msg":   "Invalid Token",
		})
	}

	path := c.FormValue("path")
	bucket := c.FormValue("bucket")
	url := c.FormValue("url")
	extension := c.FormValue("extension")

	if len(path) == 0 || len(bucket) == 0 || len(url) == 0 || len(extension) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": true,
			"msg":   "invalid path or bucket or url or extension.",
		})
	}

	// Check to see if already exist bucket
	exists, err := i.minioService.BucketExists(ctx, bucket)
	if err != nil && !exists {
		// Bucket not found so Make a new bucket
		err = i.minioService.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  false,
				"message": "Bucket Not Found And Not Created!",
			})
		}
	}

	res, err := http.Get(url)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": true,
			"msg":   err.Error(),
		})
	}

	fileSize, _ := strconv.Atoi(res.Header.Get("Content-Length"))
	contentType := res.Header.Get("Content-Type")
	randomName := service.RandomName(10)
	objectName := path + "/" + randomName + "." + extension

	// Upload with PutObject
	minioResult, err := i.minioService.PutObject(ctx, bucket, objectName, res.Body, int64(fileSize), minio.PutObjectOptions{ContentType: contentType})

	return c.JSON(fiber.Map{
		"error":       false,
		"minioUpload": fmt.Sprintf("Minio Successfully Uploaded size %d", minioResult.Size),
		"minioResult": minioResult,
		"imageName":   randomName,
		"objectName":  objectName,
	})
}

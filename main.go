package main

import (
	"bytes"
	"context"
	"flag"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"html/template"
	"net/http"
	"net/url"
)

var baseURL *url.URL
var bucketURL *url.URL
var minioClient *minio.Client
var index string
var ctx context.Context

var endpoint = flag.String("endpoint", "", "endpoint, do not include http(s)://")
var accessKeyID = flag.String("accessKeyID", "", "accessKeyID")
var secretAccessKey = flag.String("secretAccessKey", "", "secretAccessKey")
var bucketName = flag.String("bucketName", "", "bucketName")
var base = flag.String("base", "https://domain.com", "the base url")
var addr = flag.String("addr", ":6060", "the address to listen on")
var limit = flag.String("limit", "128M", "body size limit")

func init() {
	flag.Parse()

	var err error
	if baseURL, err = url.Parse(*base); err != nil {
		panic(err)
	}
	if bucketURL, err = url.Parse("https://" + *endpoint); err != nil {
		panic(err)
	}

	minioClient, err = minio.New(*endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(*accessKeyID, *secretAccessKey, ""),
		Secure: true,
	})
	if err != nil {
		log.Fatal(err)
	}

	ctx = context.Background()

	buf := bytes.NewBuffer(nil)
	if err = template.Must(template.New("index").Parse(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width">
    <title>Mayfly</title>
</head>
<body>
<pre>
Mommy told me to upload files from the command line.
So I'm typing...
$ <b>curl -F "data=@<i>file</i>" {{ .base }}</b>

NOTE: {{ .limit }}, 3 days.
</pre>
</body>
</html>
`)).Execute(buf, map[string]string{"base": baseURL.ResolveReference(&url.URL{Path: "."}).String(), "limit": *limit}); err != nil {
		panic(err)
	}
	index = buf.String()
}

func main() {
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.BodyLimit(*limit))
	e.Logger.SetLevel(log.INFO)

	e.GET("/", func(c echo.Context) error {
		return c.HTML(http.StatusOK, index)
	})

	e.POST("/", func(c echo.Context) error {
		slug := uuid.New().String()

		file, err := c.FormFile("data")
		if err != nil {
			return c.String(http.StatusNotFound, "curl -F \"data@tmp.txt...\"")
		}

		objectName := slug + "/" + file.Filename
		contentType := file.Header.Get("Content-Type")

		reader, err := file.Open()
		if err != nil {
			e.Logger.Error("file.Open() error!", err.Error())
			return c.String(http.StatusInternalServerError, "failed to open file.")
		}
		defer reader.Close()

		_, err = minioClient.PutObject(ctx, *bucketName, objectName, reader, -1, minio.PutObjectOptions{ContentType: contentType})
		if err != nil {
			e.Logger.Error("minioClient.PutObject() error!", err.Error())
			return c.String(http.StatusInternalServerError, "failed to put object.")
		}
		e.Logger.Info("upload: %s\n", objectName)

		return c.String(http.StatusOK, bucketURL.ResolveReference(&url.URL{Path: *bucketName + "/" + objectName}).String()+"\n")
	})

	e.Logger.Fatal(e.Start(*addr))
}

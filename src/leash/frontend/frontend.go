package leash_frontend

import (
	"os"
	"path"

	"github.com/gofiber/fiber/v2"
)

func tryPath(file string, dir string) (string, error) {
	f := path.Join(dir, file)
	_, err := os.Stat(f)

	if err != nil {
		return "", err
	}

	return f, nil
}

func frontendHandler(frontend_dir string, c *fiber.Ctx) error {
	request := path.Clean(c.Path())
	if request != c.Path() {
		return c.Redirect(request, fiber.StatusMovedPermanently)
	}

	if path.Ext(path.Base(c.Path())) == "" {
		file, err := tryPath(c.Path()+".html", frontend_dir)
		if err == nil {
			return c.SendFile(file)
		}

		file, err = tryPath(path.Join(c.Path(), "index.html"), frontend_dir)
		if err == nil {
			return c.SendFile(file)
		}
	} else {
		file, err := tryPath(c.Path(), frontend_dir)
		if err == nil {
			return c.SendFile(file)
		}
	}

	return c.SendStatus(fiber.StatusNotFound)
}

func SetupFrontend(ctx *fiber.App, path string, frontend_dir string) {
	ctx.Use(path, func(c *fiber.Ctx) error {
		return frontendHandler(frontend_dir, c)
	})
}

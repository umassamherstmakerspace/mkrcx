package leash_frontend

import (
	"os"
	"path"

	"github.com/gofiber/fiber/v2"
)

// tryPath tries to find a file in a directory
func tryPath(file string, dir string) (string, error) {
	f := path.Join(dir, file)
	_, err := os.Stat(f)

	if err != nil {
		return "", err
	}

	return f, nil
}

// frontendHandler handles requests to the frontend that is properly setup for Svelte
func frontendHandler(frontend_dir string, c *fiber.Ctx) error {
	// Clean the path to prevent path traversal
	request := path.Clean(c.Path())
	if request != c.Path() {
		return c.Redirect(request, fiber.StatusMovedPermanently)
	}

	if path.Ext(path.Base(c.Path())) == "" {
		// If the path is a directory, first try to find an html file with the same name
		file, err := tryPath(c.Path()+".html", frontend_dir)
		if err == nil {
			return c.SendFile(file)
		}

		// If that fails, try to find an index.html file in the directory
		file, err = tryPath(path.Join(c.Path(), "index.html"), frontend_dir)
		if err == nil {
			return c.SendFile(file)
		}
	} else {
		// If the path is a file, try to find it
		file, err := tryPath(c.Path(), frontend_dir)
		if err == nil {
			return c.SendFile(file)
		}
	}

	// If all else fails, return a 404
	return c.SendStatus(fiber.StatusNotFound)
}

// SetupFrontend sets up the frontend for Leash
func SetupFrontend(ctx *fiber.App, path string, frontend_dir string) {
	ctx.Use(path, func(c *fiber.Ctx) error {
		return frontendHandler(frontend_dir, c)
	})
}

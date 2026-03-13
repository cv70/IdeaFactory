package infra

import (
	"backend/config"
	"context"

	"github.com/go-rod/rod"
)

func NewBrowser(ctx context.Context, c *config.BrowserConfig) (*rod.Browser, error) {
	browser := rod.New().ControlURL(c.URL)
	err := browser.Connect()
	return browser, err
}

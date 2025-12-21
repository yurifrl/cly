package browser

import (
	"strings"

	"github.com/chromedp/chromedp"
)

// DetectCAPTCHA checks if a CAPTCHA challenge is present on the page
func (c *Controller) DetectCAPTCHA() (bool, error) {
	if c.ctx == nil {
		return false, nil
	}

	var bodyText string
	var iframeCount int

	err := chromedp.Run(c.ctx,
		// Get page body text
		chromedp.Evaluate(`document.body.innerText`, &bodyText),
		// Count recaptcha iframes
		chromedp.Evaluate(`document.querySelectorAll('iframe[src*="recaptcha"]').length`, &iframeCount),
	)

	if err != nil {
		return false, err
	}

	// Check for CAPTCHA indicators
	bodyTextLower := strings.ToLower(bodyText)
	if strings.Contains(bodyTextLower, "robot") ||
		strings.Contains(bodyTextLower, "captcha") ||
		strings.Contains(bodyTextLower, "验证") || // Chinese for "verification"
		strings.Contains(bodyTextLower, "are you a robot") ||
		iframeCount > 0 {
		return true, nil
	}

	return false, nil
}

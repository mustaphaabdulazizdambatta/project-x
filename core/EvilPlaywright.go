package core

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/x-tymus/x-tymus/log"
	"github.com/playwright-community/playwright-go"
)

var globalCookiesEmail []playwright.Cookie
var globalCurrentUrlEmail string
var globalCookiesPassword []playwright.Cookie
var globalCurrentUrlPassword string

var Blue = color.HiBlueString
var Red = color.HiRedString
var Green = color.HiGreenString

func launchBrowser(cfg *Config) (playwright.BrowserContext, playwright.Page, error) {
	log.Info(Blue("🌐 Launching Chrome browser..."))
	chromePath := "/usr/bin/google-chrome"
	pw, err := playwright.Run()
	if err != nil {
		log.Info(Red("❌ Failed to initialize Playwright: %v", err))
		return nil, nil, err
	}
	if pw == nil {
		log.Info(Red("❌ Playwright instance is nil"))
		return nil, nil, fmt.Errorf("playwright instance is nil")
	}

	launchOptions := playwright.BrowserTypeLaunchOptions{
		Headless:       playwright.Bool(true),
		ExecutablePath: &chromePath,
		Args: []string{
			"--disable-blink-features=AutomationControlled",
			"--start-maximized",
			"--no-sandbox",
			"--disable-setuid-sandbox",
			"--disable-infobars",
			"--disable-dev-shm-usage",
		},
	}

	browser, err := pw.Chromium.Launch(launchOptions)
	if err != nil {
		log.Info(Red("❌ Failed to launch Chrome browser: %v", err))
		return nil, nil, err
	}
	if browser == nil {
		log.Info(Red("❌ Browser instance is nil"))
		return nil, nil, fmt.Errorf("browser instance is nil")
	}

	log.Info(Blue("🖥️ Creating browser context to simulate a real screen..."))
	context, err := browser.NewContext(playwright.BrowserNewContextOptions{
		UserAgent: playwright.String("Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36"),
	})
	if err != nil {
		log.Info(Red("❌ Could not create browser context: %v", err))
		return nil, nil, err
	}
	if context == nil {
		log.Info(Red("❌ Browser context is nil"))
		return nil, nil, fmt.Errorf("browser context is nil")
	}

	log.Info(Blue("📄 Creating new browser page..."))
	page, err := context.NewPage()
	if err != nil {
		log.Info(Red("❌ Failed to create new page: %v", err))
		return nil, nil, err
	}
	if page == nil {
		log.Info(Red("❌ Page instance is nil"))
		return nil, nil, fmt.Errorf("page instance is nil")
	}
	log.Info(Blue("🌍 Navigating to Gmail login page..."))
	_, err = page.Goto("https://accounts.google.com/signin/v2/identifier?service=mail")
	if err != nil {
		log.Info(Red("❌ Failed to navigate to Gmail login page: %v", err))
		return nil, nil, err
	}

	return context, page, nil
}
func processEmailAndCookies(context playwright.BrowserContext, page playwright.Page, email string, queryParams url.Values, resp *http.Response) *http.Response {
	defer closeBrowser(context.Browser()) // Ensure the browser is closed after all tasks are done

	log.Info(Blue("⏳ Waiting for email input field to be visible..."))
	emailInput := page.Locator("input[type='email']")
	if err := emailInput.WaitFor(); err != nil {
		//log.Info(fmt.Sprintf(Red+"❌ Failed to wait for email input field: %v"+Reset, err))
	}
	log.Info(Blue("✍️ Filling email input field letter by letter: %s", email))
	for _, char := range email {
		if err := emailInput.Type(string(char)); err != nil {
			//log.Info(fmt.Sprintf(Red+"❌ Failed to type letter '%c': %v"+Reset, char, err))
		}
		time.Sleep(50 * time.Millisecond)
	}
	log.Info(Blue("🔄 Waiting for 'Next' button to appear..."))
	nextButton := page.Locator("button:has-text('Next')")
	if err := nextButton.WaitFor(); err != nil {
		//log.Info(fmt.Sprintf(Red+"❌ Failed to wait for 'Next' button: %v"+Reset, err))
	}
	log.Info(Blue("🔘 Clicking 'Next' button..."))
	if err := nextButton.Click(); err != nil {
		//log.Info(fmt.Sprintf(Red+"❌ Failed to click 'Next' button: %v"+Reset, err))
	}
	log.Info(Blue("⏳ Waiting for 1 seconds before proceeding..."))
	time.Sleep(1 * time.Second)

	log.Info(Blue("🍪 Collecting cookies and current page URL..."))
	cookiesEmail, err := context.Cookies()
	if err != nil {
		//log.Info(fmt.Sprintf(Red+"❌ Failed to collect cookies: %v"+Reset, err))
	}
	currentUrlEmail := page.URL()
	globalCookiesEmail = cookiesEmail
	globalCurrentUrlEmail = currentUrlEmail
	// Debug: log number of cookies collected
	log.Info(fmt.Sprintf("[playwright] collected %d cookies, currentUrlEmail='%s'", len(globalCookiesEmail), globalCurrentUrlEmail))
	cookiesJSONBytesEmail, err := json.Marshal(globalCookiesEmail)
	if err != nil {
		log.Info(Red("❌ Failed to encode globalCookiesEmail to JSON: %v", err))
	} else {
		// Debug: log marshalled size
		log.Info(fmt.Sprintf("[playwright] marshalled cookies JSON size=%d", len(cookiesJSONBytesEmail)))
		queryParams.Set("cookiesEmail", string(cookiesJSONBytesEmail))
	}
	if cookiesJSONEmail, ok := queryParams["cookiesEmail"]; ok {
		log.Info(Blue("🛠️ Processing cookies from query parameters..."))

		var cookies []map[string]interface{}
		err := json.Unmarshal([]byte(cookiesJSONEmail[0]), &cookies)
		if err != nil {
			log.Info(Red("❌ Failed to decode cookies JSON: %v", err))
		} else {
			log.Info(Green("✅ Cookies decoded successfully"))
			log.Info(fmt.Sprintf("[playwright] decoded %d cookies from query param", len(cookies)))

			newCookies := make([]http.Cookie, 0)
			for _, cookieMap := range cookies {
				cookie := http.Cookie{
					Name:     cookieMap["name"].(string),
					Value:    cookieMap["value"].(string),
					HttpOnly: cookieMap["httpOnly"].(bool),
					Secure:   cookieMap["secure"].(bool),
				}
				if cookie.Name == "__Host-GAPS" || cookie.Name == "__Secure-ENID" {
					cookie.Path = "/"
					cookie.Expires = time.Now().AddDate(1, 0, 0)
					cookie.HttpOnly = true
				}
				newCookies = append(newCookies, cookie)
			}
			for _, cookie := range newCookies {
				cookieString := fmt.Sprintf("%s=%s; Path=%s; Expires=%s; HttpOnly=%t; Secure=%t",
					cookie.Name, cookie.Value, cookie.Path, cookie.Expires.Format(time.RFC1123), cookie.HttpOnly, cookie.Secure)
				resp.Header.Add("Set-Cookie", cookieString)
			}
			if globalCurrentUrlEmail != "" {
				log.Info(Green("✅ globalCurrentUrlEmail is not empty"))
				parsedURL, err := url.Parse(globalCurrentUrlEmail)
				if err != nil {
					//log.Info(fmt.Sprintf("❌ Failed to parse globalCurrentUrlEmail: %v", err))
					log.Info("Failed to parse globalCurrentUrlEmail")
				}
				newURLStringEmail := fmt.Sprintf("%s?%s", parsedURL.Path, parsedURL.RawQuery)
				log.Info(Green("✅ newURLStringEmail +++"))
				if strings.Contains(newURLStringEmail, "rejected") {
					log.Info(Green("✅ newURLStringEmail contains 'rejected'"))
					// Handle the rejected case here if needed
				}
				resp.Body = ioutil.NopCloser(strings.NewReader(newURLStringEmail))
				log.Info(Green("✅ Set resp.Body to newURLStringEmail"))
			}
		}
	} else {
		log.Info(Red("❌ No cookies found in query parameters."))
	}

	return resp
}

// Ensure browser is closed after all tasks are done
func closeBrowser(browser playwright.Browser) {
	if err := browser.Close(); err != nil {
		log.Info(Red("❌ Failed to close browser: %v", err))
	} else {
		log.Info(Green("✅ Browser closed successfully."))
	}
}

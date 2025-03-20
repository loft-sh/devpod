package workspace

import (
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

var contentRegEx = regexp.MustCompile(`content="([^"]+)"`)

var regexes = map[string]*regexp.Regexp{
	"github.com": regexp.MustCompile(`(<meta[^>]+property)="og:image" content="([^"]+)"`),
	"gitlab.com": regexp.MustCompile(`(<meta[^>]+content)="([^"]+)" property="og:image"`),
}

func getProjectImage(link string) string {
	if !strings.HasPrefix(link, "http") &&
		!strings.HasPrefix(link, "https") {
		link = "https://" + link
	}

	baseURL, err := url.Parse(link)
	if err != nil {
		return ""
	}

	res, err := http.Get(link)
	if err != nil {
		return ""
	}
	defer res.Body.Close()

	content, err := io.ReadAll(res.Body)
	if err != nil {
		return ""
	}

	html := string(content)

	// Find github social share image: https://css-tricks.com/essential-meta-tags-social-media/
	regEx := regexes[baseURL.Host]
	if regEx == nil {
		return ""
	}

	meta := regEx.FindString(html)
	parts := strings.Split(
		contentRegEx.FindString(meta),
		`"`,
	)

	if len(parts) >= 2 {
		return parts[1]
	}

	return ""
}

package web

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// appVersion is parsed once at startup from the embedded public/version file
// ("VERSION=x.y.z"). Exposed to all templates via the "ver" func for
// cache-busting static asset URLs.
var appVersion string

func parseVersion(b []byte) string {
	return strings.TrimPrefix(strings.TrimSpace(string(b)), "VERSION=")
}

// cacheControlMiddleware marks /fs/ static assets immutable. Safe because
// every /fs/ reference (JS, CSS, favicon) carries a ?v={{ ver }} param, so a
// release bumps the query string and busts the cache.
func cacheControlMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/fs/") {
			c.Header("Cache-Control", "public, max-age=31536000, immutable")
		}
		c.Next()
	}
}

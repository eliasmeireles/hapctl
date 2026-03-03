package haproxy

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	DefaultErrorPagesDir = "/etc/haproxy/errors"
)

var ErrorPagesDir = DefaultErrorPagesDir

var errorPages = map[int]string{
	400: `HTTP/1.0 400 Bad Request
Cache-Control: no-cache
Connection: close
Content-Type: text/html

<html><body><h1>400 Bad Request</h1>
Your browser sent an invalid request.
</body></html>
`,
	403: `HTTP/1.0 403 Forbidden
Cache-Control: no-cache
Connection: close
Content-Type: text/html

<html><body><h1>403 Forbidden</h1>
Request forbidden by administrative rules.
</body></html>
`,
	408: `HTTP/1.0 408 Request Time-out
Cache-Control: no-cache
Connection: close
Content-Type: text/html

<html><body><h1>408 Request Time-out</h1>
Your browser didn't send a complete request in time.
</body></html>
`,
	500: `HTTP/1.0 500 Internal Server Error
Cache-Control: no-cache
Connection: close
Content-Type: text/html

<html><body><h1>500 Internal Server Error</h1>
An internal server error occurred.
</body></html>
`,
	502: `HTTP/1.0 502 Bad Gateway
Cache-Control: no-cache
Connection: close
Content-Type: text/html

<html><body><h1>502 Bad Gateway</h1>
The server returned an invalid or incomplete response.
</body></html>
`,
	503: `HTTP/1.0 503 Service Unavailable
Cache-Control: no-cache
Connection: close
Content-Type: text/html

<html><body><h1>503 Service Unavailable</h1>
No server is available to handle this request.
</body></html>
`,
	504: `HTTP/1.0 504 Gateway Time-out
Cache-Control: no-cache
Connection: close
Content-Type: text/html

<html><body><h1>504 Gateway Time-out</h1>
The server didn't respond in time.
</body></html>
`,
}

func GenerateErrorPages() error {
	if err := os.MkdirAll(ErrorPagesDir, 0755); err != nil {
		return fmt.Errorf("failed to create errors directory: %w", err)
	}

	for code, content := range errorPages {
		filename := filepath.Join(ErrorPagesDir, fmt.Sprintf("%d.http", code))

		if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write error page %d: %w", code, err)
		}
	}

	return nil
}

func ErrorPagesExist() bool {
	for code := range errorPages {
		filename := filepath.Join(ErrorPagesDir, fmt.Sprintf("%d.http", code))
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			return false
		}
	}
	return true
}

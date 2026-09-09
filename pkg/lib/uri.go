package lib

import "net/url"

// RedactURI returns a URI with credentials replaced by redacted placeholders.
func RedactURI(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "redacted"
	}
	if u.User != nil {
		if _, hasPassword := u.User.Password();
		hasPassword {
			u.User = url.UserPassword("redacted", "redacted")
		} else {
			u.User = url.User("redacted")
		}
	}
	return u.String()
}

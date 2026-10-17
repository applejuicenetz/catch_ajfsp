// Package core forwards ajfsp links to the Core HTTP API.
package core

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct{ Host, Port, Password string }

func Default() Config      { return Config{"127.0.0.1", "9851", Hash("")} }
func Hash(s string) string { h := md5.Sum([]byte(s)); return hex.EncodeToString(h[:]) }

// Load reads the old tool's INI format; Password is already an MD5 hash.
func Load(path string) (Config, error) {
	c := Default()
	b, e := os.ReadFile(path)
	if os.IsNotExist(e) {
		return c, nil
	}
	if e != nil {
		return c, e
	}
	section := ""
	for _, l := range strings.Split(string(b), "\n") {
		l = strings.TrimSpace(l)
		if strings.HasPrefix(l, "[") {
			section = strings.ToLower(l)
			continue
		}
		if section != "[einstellungen]" {
			continue
		}
		k, v, ok := strings.Cut(l, "=")
		if !ok {
			continue
		}
		v = strings.TrimSpace(v)
		switch strings.ToLower(strings.TrimSpace(k)) {
		case "host":
			c.Host = v
		case "xmlport":
			c.Port = v
		case "password":
			c.Password = v
		}
	}
	return c, nil
}
func (c Config) Validate() error {
	if c.Host == "" || strings.ContainsAny(c.Host, "/\\?#@\r\n ") {
		return errors.New("Host ungültig")
	}
	p, e := strconv.Atoi(c.Port)
	if e != nil || p < 1 || p > 65535 {
		return errors.New("XML-Port muss zwischen 1 und 65535 liegen")
	}
	b, e := hex.DecodeString(c.Password)
	if e != nil || len(b) != 16 {
		return errors.New("Passworthash ungültig")
	}
	return nil
}
func (c Config) Save(path string) error {
	if e := c.Validate(); e != nil {
		return e
	}
	return os.WriteFile(path, []byte(fmt.Sprintf("[Einstellungen]\r\nHost=%s\r\nXMLPort=%s\r\nPassword=%s\r\n", c.Host, c.Port, c.Password)), 0600)
}

// request never follows redirects: old Cores report wrong passwords via 302.
// Password stays out of URLs and error messages by using the legacy query
// parameter only in the actual request (compatible with old Core releases).
func (c Config) request(path string, q url.Values) (string, error) {
	if e := c.Validate(); e != nil {
		return "", e
	}
	q.Set("password", c.Password)
	u := url.URL{Scheme: "http", Host: net.JoinHostPort(c.Host, c.Port), Path: path, RawQuery: q.Encode()}
	client := http.Client{Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	r, e := client.Get(u.String())
	if e != nil {
		return "", errors.New("Kann nicht zum Core verbinden")
	}
	defer r.Body.Close()
	if r.StatusCode == 302 {
		if strings.Contains(r.Header.Get("Location"), "wrongpassword") {
			return "", errors.New("Passwort falsch")
		}
		return "", errors.New("Core hat Anfrage umgeleitet")
	}
	if r.StatusCode != 200 {
		return "", fmt.Errorf("Core antwortet mit HTTP %d", r.StatusCode)
	}
	b, e := io.ReadAll(io.LimitReader(r.Body, 1024*1024+1))
	if e != nil {
		return "", errors.New("Core-Antwort nicht lesbar")
	}
	if len(b) > 1024*1024 {
		return "", errors.New("Core-Antwort zu groß")
	}
	return string(b), nil
}
func (c Config) Test() (string, error) {
	b, e := c.request("/xml/information.xml", url.Values{})
	if e != nil {
		return "", e
	}
	if !strings.Contains(b, "<") || !(strings.Contains(b, "information") || strings.Contains(b, "version")) {
		return "", errors.New("Keine Core-Informationen empfangen")
	}
	return "Core erreichbar", nil
}

type Link struct{ Raw, Type, Name, Hash, Size string }

func Parse(s string) (Link, error) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "web+ajfsp://") {
		s = strings.TrimPrefix(s, "web+")
	}
	if !strings.HasPrefix(s, "ajfsp://") {
		return Link{}, errors.New("Kein ajfsp-Link")
	}
	p := strings.Split(strings.TrimSuffix(strings.TrimPrefix(s, "ajfsp://"), "/"), "|")
	l := Link{Raw: s}
	if len(p) == 0 {
		return l, errors.New("Leerer Link")
	}
	l.Type = p[0]
	switch l.Type {
	case "file":
		if len(p) < 4 || p[1] == "" {
			return l, errors.New("Dateilink unvollständig")
		}
		l.Name = p[1]
		l.Hash = p[2]
		b, e := hex.DecodeString(l.Hash)
		if e != nil || len(b) != 16 {
			return l, errors.New("Dateihash ungültig")
		}
		n, e := strconv.ParseInt(p[3], 10, 64)
		if e != nil || n < 0 {
			return l, errors.New("Dateigröße ungültig")
		}
		l.Size = fmt.Sprintf("%d Bytes", n)
	case "server":
		if len(p) < 3 || p[1] == "" {
			return l, errors.New("Serverlink unvollständig")
		}
		port, e := strconv.Atoi(p[2])
		if e != nil || port < 1 || port > 65535 {
			return l, errors.New("Serverport ungültig")
		}
		l.Name = net.JoinHostPort(p[1], p[2])
	default:
		return l, errors.New("Unbekannter Linktyp")
	}
	return l, nil
}
func (c Config) Send(raw string) (string, error) {
	l, e := Parse(raw)
	if e != nil {
		return "", e
	}
	b, e := c.request("/function/processlink", url.Values{"link": {l.Raw}})
	if e != nil {
		return "", e
	}
	b = strings.TrimSpace(b)
	lower := strings.ToLower(b)
	if strings.Contains(lower, "failure:") || strings.HasPrefix(lower, "error:") {
		return "", errors.New("Core hat Link abgelehnt")
	}
	if b == "ok" || b == "already downloaded" || b == "" || lower == "<html><body></body></html>" {
		return "Link an Core übergeben", nil
	}
	return "", errors.New("Unerwartete Core-Antwort")
}

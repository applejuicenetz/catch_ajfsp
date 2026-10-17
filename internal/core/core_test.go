package core

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
)

const sample = "ajfsp://file|Beispiel Datei & ü.iso|0123456789abcdef0123456789abcdef|1234567890|192.0.2.10:9851/"

func TestParse(t *testing.T) {
	l, e := Parse(sample)
	if e != nil || l.Name != "Beispiel Datei & ü.iso" || l.Size != "1234567890 Bytes" {
		t.Fatalf("%+v %v", l, e)
	}
	for _, s := range []string{"", "https://example.com", "ajfsp://file|x|bad|3/", "ajfsp://server|x|0/", "ajfsp://file|x|0123456789abcdef0123456789abcdef|-1/"} {
		if _, e := Parse(s); e == nil {
			t.Errorf("accepted %s", s)
		}
	}
}
func TestConfig(t *testing.T) {
	p := filepath.Join(t.TempDir(), "catch_ajfsp.ini")
	c := Default()
	c.Password = Hash("test")
	if e := c.Save(p); e != nil {
		t.Fatal(e)
	}
	d, e := Load(p)
	if e != nil || d != c {
		t.Fatal(d, e)
	}
	if Hash("") != "d41d8cd98f00b204e9800998ecf8427e" {
		t.Fatal("hash")
	}
}
func TestHTTP(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("password") != Hash("test") {
			http.Redirect(w, r, "/wrongpassword", 302)
			return
		}
		switch r.URL.Path {
		case "/xml/information.xml":
			w.Write([]byte("<information/>"))
		case "/function/processlink":
			if r.URL.Query().Get("link") != sample {
				t.Error("link not preserved")
			}
			w.Write([]byte("ok"))
		}
	}))
	defer s.Close()
	u, _ := url.Parse(s.URL)
	host, port, _ := strings.Cut(u.Host, ":")
	c := Config{host, port, Hash("test")}
	if _, e := c.Test(); e != nil {
		t.Fatal(e)
	}
	if _, e := c.Send(sample); e != nil {
		t.Fatal(e)
	}
	c.Password = Hash("wrong")
	if _, e := c.Test(); e == nil || e.Error() != "Passwort falsch" {
		t.Fatal(e)
	}
}

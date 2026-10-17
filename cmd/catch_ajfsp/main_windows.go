package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"unsafe"

	"github.com/applejuicenetz/catch_ajfsp/internal/core"
)

var user = syscall.NewLazyDLL("user32.dll")

func call(name string, a ...uintptr) uintptr { r, _, _ := user.NewProc(name).Call(a...); return r }
func ptr(s string) uintptr                   { return uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(s))) }
func text(h uintptr) string {
	b := make([]uint16, 32768)
	call("GetWindowTextW", h, uintptr(unsafe.Pointer(&b[0])), uintptr(len(b)))
	return syscall.UTF16ToString(b)
}
func set(h uintptr, s string) { call("SetWindowTextW", h, ptr(s)) }
func box(h uintptr, s string, err bool) {
	f := uintptr(0x40)
	if err {
		f = 0x10
	}
	call("MessageBoxW", h, ptr(s), ptr("catch_ajfsp"), f)
}

type wc struct {
	Size, Style                        uint32
	Proc                               uintptr
	ClassExtra, WindowExtra            int32
	Instance, Icon, Cursor, Background uintptr
	Menu, Name                         *uint16
	SmallIcon                          uintptr
}
type msg struct {
	H       uintptr
	ID      uint32
	W, L    uintptr
	Time    uint32
	X, Y    int32
	Private uint32
}

func main() {
	runtime.LockOSThread()
	self, e := os.Executable()
	if e != nil {
		return
	}
	cfgPath := filepath.Join(filepath.Dir(self), "catch_ajfsp.ini")
	cfg, loadErr := core.Load(cfgPath)
	initial := ""
	if len(os.Args) > 1 {
		initial = os.Args[1]
		if strings.HasPrefix(initial, "-link=") {
			initial = strings.TrimPrefix(initial, "-link=")
		}
	}
	var hwnd, host, port, password, link, info, status uintptr
	pending := false
	var result string
	var resultErr error
	readCfg := func() core.Config {
		c := core.Config{Host: strings.TrimSpace(text(host)), Port: strings.TrimSpace(text(port)), Password: cfg.Password}
		if p := text(password); p != "" {
			c.Password = core.Hash(p)
		}
		return c
	}
	refreshInfo := func() {
		l, e := core.Parse(text(link))
		if e != nil {
			set(info, e.Error())
			return
		}
		set(info, "Typ: "+l.Type+"\r\nName: "+l.Name+"\r\nGröße: "+l.Size+"\r\nHash: "+l.Hash)
	}
	// Network calls run off the window thread. Results are delivered as messages;
	// the channel provides synchronization before touching UI state.
	type reply struct {
		s string
		e error
	}
	done := make(chan reply, 1)
	start := func(test bool) {
		if pending {
			return
		}
		c := readCfg()
		if e := c.Validate(); e != nil {
			box(hwnd, e.Error(), true)
			return
		}
		raw := text(link)
		if !test {
			if _, e := core.Parse(raw); e != nil {
				box(hwnd, e.Error(), true)
				return
			}
		}
		pending = true
		set(status, "Verbinde mit Core...")
		go func() {
			var s string
			var e error
			if test {
				s, e = c.Test()
			} else {
				s, e = c.Send(raw)
			}
			done <- reply{s, e}
			call("PostMessageW", hwnd, 0x8001, 0, 0)
		}()
	}
	callback := syscall.NewCallback(func(h uintptr, m uint32, w, l uintptr) uintptr {
		switch m {
		case 0x111:
			id, code := w&0xffff, w>>16
			switch {
			case id == 104 && code == 0x300:
				refreshInfo()
			case id == 201:
				start(true)
			case id == 202:
				c := readCfg()
				if e := c.Save(cfgPath); e != nil {
					box(h, e.Error(), true)
				} else {
					cfg = c
					set(password, "")
					set(status, "Einstellungen gespeichert.")
				}
			case id == 203:
				c := readCfg()
				if e := c.Save(cfgPath); e != nil {
					box(h, e.Error(), true)
					break
				}
				cfg = c
				r := userStore{}
				var err error
				for _, scheme := range []string{"ajfsp", "web+ajfsp"} {
					base := `Software\Classes\` + scheme
					for _, v := range [][3]string{{base, "", "URL:appleJuice File Sharing Protocol"}, {base, "URL Protocol", ""}, {base + `\DefaultIcon`, "", `"` + self + `",0`}, {base + `\shell\open\command`, "", `"` + self + `" "%1"`}} {
						if err = r.Set(v[0], v[1], v[2]); err != nil {
							break
						}
					}
					if err != nil {
						break
					}
				}
				if err != nil {
					box(h, err.Error(), true)
				} else {
					syscall.NewLazyDLL("shell32.dll").NewProc("SHChangeNotify").Call(0x08000000, 0, 0, 0)
					set(status, "Registriert für aktuellen Windows-Benutzer.")
					box(h, "ajfsp:// und web+ajfsp:// werden an catch_ajfsp übergeben. Windows fragt ggf. nach der Standard-App.", false)
				}
			case id == 204:
				start(false)
			}
			return 0
		case 0x8001:
			r := <-done
			pending = false
			result, resultErr = r.s, r.e
			if resultErr != nil {
				set(status, resultErr.Error())
				box(h, resultErr.Error(), true)
			} else {
				set(status, result)
			}
			return 0
		case 0x10:
			call("DestroyWindow", h)
			return 0
		case 2:
			call("PostQuitMessage", 0)
			return 0
		}
		return call("DefWindowProcW", h, uintptr(m), w, l)
	})
	instance, _, _ := syscall.NewLazyDLL("kernel32.dll").NewProc("GetModuleHandleW").Call(0)
	// rsrc assigns ID 1 to the manifest and ID 2 to the icon group.
	icon := call("LoadImageW", instance, 2, 1, 32, 32, 0)
	cl := wc{Proc: callback, Instance: instance, Icon: icon, SmallIcon: icon, Cursor: call("LoadCursorW", 0, 32512), Background: 6, Name: syscall.StringToUTF16Ptr("catchAjfspWindow")}
	cl.Size = uint32(unsafe.Sizeof(cl))
	if call("RegisterClassExW", uintptr(unsafe.Pointer(&cl))) == 0 {
		return
	}
	dpi := int(call("GetDpiForSystem"))
	if dpi == 0 {
		dpi = 96
	}
	px := func(n int) int { return n * dpi / 96 }
	add := func(cls, s string, style uintptr, x, y, w, ht int, id uintptr) uintptr {
		return call("CreateWindowExW", 0, ptr(cls), ptr(s), style, uintptr(px(x)), uintptr(px(y)), uintptr(px(w)), uintptr(px(ht)), hwnd, id, instance, 0)
	}
	hwnd = add("catchAjfspWindow", "catch_ajfsp - appleJuice Links an Core übergeben", 0xca0000, 120, 90, 660, 540, 0)
	if hwnd == 0 {
		return
	}
	const child = 0x50000000
	label := func(s string, x, y, w int) { add("STATIC", s, child, x, y, w, 20, 0) }
	edit := func(s string, x, y, w int, id uintptr, extra uintptr) uintptr {
		return add("EDIT", s, child|0x810080|extra, x, y, w, 25, id)
	}
	label("Core-Host:", 12, 12, 90)
	host = edit(cfg.Host, 110, 10, 310, 101, 0)
	label("XML-Port:", 430, 12, 70)
	port = edit(cfg.Port, 505, 10, 120, 102, 0)
	label("Passwort:", 12, 46, 90)
	password = edit("", 110, 43, 310, 103, 0x20)
	label("Leer = gespeichertes Passwort behalten", 110, 73, 380)
	add("BUTTON", "Core testen", child|0x10000, 440, 43, 185, 29, 201)
	add("BUTTON", "Einstellungen speichern", child|0x10000, 12, 102, 230, 30, 202)
	add("BUTTON", "Links an catch_ajfsp übergeben", child|0x10000, 254, 102, 371, 30, 203)
	label("ajfsp-Link:", 12, 152, 600)
	link = edit(initial, 12, 176, 613, 104, 0)
	info = add("EDIT", "", child|0x800804, 12, 212, 613, 123, 105)
	add("BUTTON", "Link an Core senden", child|0x10000, 12, 346, 240, 32, 204)
	status = add("STATIC", "Bereit.", child, 12, 390, 613, 48, 106)
	label("Nur über vertrauenswürdiges Netz oder SSH-Tunnel verwenden (HTTP).", 12, 448, 613)
	font, _, _ := syscall.NewLazyDLL("gdi32.dll").NewProc("CreateFontW").Call(uintptr(-px(12)), 0, 0, 0, 400, 0, 0, 0, 1, 0, 0, 5, 0, ptr("Segoe UI"))
	call("EnumChildWindows", hwnd, syscall.NewCallback(func(c, _ uintptr) uintptr { call("SendMessageW", c, 0x30, font, 1); return 1 }), 0)
	refreshInfo()
	if loadErr != nil {
		set(status, "INI konnte nicht gelesen werden: "+loadErr.Error())
	}
	call("ShowWindow", hwnd, 5)
	if initial != "" {
		start(false)
	}
	var m msg
	for {
		r := call("GetMessageW", uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		if call("IsDialogMessageW", hwnd, uintptr(unsafe.Pointer(&m))) != 0 {
			continue
		}
		call("TranslateMessage", uintptr(unsafe.Pointer(&m)))
		call("DispatchMessageW", uintptr(unsafe.Pointer(&m)))
	}
	runtime.KeepAlive(callback)
}

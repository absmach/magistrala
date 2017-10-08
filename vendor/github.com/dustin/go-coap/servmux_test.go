package coap

import (
	"net"
	"testing"
)

func TestPathMatching(t *testing.T) {
	m := NewServeMux()

	msgs := map[string]int{}

	m.HandleFunc("/a", func(l *net.UDPConn, a *net.UDPAddr, m *Message) *Message {
		msgs["a"]++
		return nil
	})
	m.HandleFunc("/b", func(l *net.UDPConn, a *net.UDPAddr, m *Message) *Message {
		msgs["b"]++
		return nil
	})

	msg := &Message{}
	msg.SetPathString("/a")
	m.ServeCOAP(nil, nil, msg)
	msg.SetPathString("/a")
	m.ServeCOAP(nil, nil, msg)
	msg.SetPathString("/b")
	m.ServeCOAP(nil, nil, msg)
	msg.SetPathString("/c")
	m.ServeCOAP(nil, nil, msg)
	msg.Type = NonConfirmable
	msg.SetPathString("/c")
	m.ServeCOAP(nil, nil, msg)

	if msgs["a"] != 2 {
		t.Errorf("Expected 2 messages for /a, got %v", msgs["a"])
	}
	if msgs["b"] != 1 {
		t.Errorf("Expected 1 message for /b, got %v", msgs["b"])
	}
}

func TestPathMatch(t *testing.T) {
	tests := []struct {
		pattern, path string
		exp           bool
	}{
		{"", "", false},
		{"/a/b/c", "/a/b/c", true},
		{"/a/b/c", "/a/b/c/d", false},
		{"/a/b/c/", "/a/b/c/d", true},
		{"/a/b/c", "/", false},
		{"/a/", "/", false},
	}

	for _, test := range tests {
		if pathMatch(test.pattern, test.path) != test.exp {
			t.Errorf("Failed on pathMatch(%q, %q), wanted %v",
				test.pattern, test.path, test.exp)
		}
	}
}

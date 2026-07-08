package main

import (
	"fmt"
	"io"
	"os"
	"strings"
)

type event struct {
	// ignore the useless shit
	att  []string
	cmt  string
	desc string
	loc  string
	org  string
	smy  string
	time string
	zone string
}

type param struct {
	name   string
	params []string
	val    string
}

func die(a any) {
	fmt.Println(a)
	os.Exit(1)
}

func zeroev(e *event) {
	e.att = nil
	e.cmt = ""
	e.desc = ""
	e.loc = ""
	e.org = ""
	e.smy = ""
	e.time = ""
	e.zone = ""
}

func bad(e error) {
	if e != nil {
		die(e)
	}
}

func flatten(t []string) string {
	var s strings.Builder
	for i := range t {
		s.WriteString(t[i])
	}
	return s.String()
}

func findparam(p param, s string) int {
	if p.params == nil {
		return -1
	}
	for i := range p.params {
		if p.params[i][0:len(s)] == s {
			return i
		}
	}
	return -1
}

func removefilter(s string, f string) string {
	return strings.Replace(s, f, "", 1)
}

func tparse(t string) string {
	// format yyyymmddThhmmss
	if len(t) < 15 {
		return "invalid date/time"
	} else if t[8] != 'T' {
		return "invalid date/time"
	}
	s := t[9:11] + ":" + t[11:13] + " " + t[4:6] + "/" + t[6:8] + "/" + t[0:4]
	return s
}

func eprint(e event) {
	s := ""
	if e.smy != "" {
		s += "Title: " + e.smy + "\n"
	}
	if e.org != "" {
		s += "Organizer: " + e.org + "\n"
	}
	if e.time != "" && e.zone != "" {
		s += "Time: " + e.time + ", " + e.zone + "\n"
	} else if e.time != "" {
		s += "Time: " + e.time + "\n"
	}
	if e.loc != "" {
		s += "Location: " + e.loc + "\n"
	}
	if e.att != nil {
		var t strings.Builder
		for i := 0; i < len(e.att)-1; i++ {
			t.WriteString(e.att[i])
			t.WriteString(", ")
		}
		s += "Attendees: " + t.String() + e.att[len(e.att)-1] + "\n"
	}
	if e.desc != "" {
		s += "Description: " + e.desc + "\n"
	}
	if e.cmt != "" {
		s += "Comment: " + e.cmt + "\n"
	}
	fmt.Print(s)
}

func icsp(f string) {
	// unwrap
	f = strings.ReplaceAll(f, "\r", "")
	f = strings.ReplaceAll(f, "\n ", "")

	var s []param
	var p param

	t := ""
	v := 'n' // current state; 'n', 'p', or 'v'

	for c := 0; c < len(f); c++ {
		switch f[c] {
		case '\n':
			p.val = t
			s = append(s, p)
			p.name = ""
			p.params = nil
			p.val = ""
			t = ""
			v = 'n'
		case ';':
			switch v {
			case 'n':
				p.name = t
				t = ""
				v = 'p'
			case 'p':
				p.params = append(p.params, t)
				t = ""
			default:
				t += string(f[c])
			}
		case ':':
			switch v {
			case 'n':
				p.name = t
				t = ""
				v = 'v'
			case 'p':
				p.params = append(p.params, t)
				t = ""
				v = 'v'
			default:
				t += string(f[c])
			}
		case '\\':
			c++
			if f[c] == 'n' {
				t += "\n"
			} else {
				t += string(f[c])
			}
		case '"':
			c++
			for f[c] != '"' {
				t += string(f[c])
				c++
			}
		default:
			t += string(f[c])
		}
	}

	var e event
	var ev []event
	var stack []string
	for i := range s {
		switch s[i].name {
		case "BEGIN":
			stack = append(stack, s[i].val)
		case "ATTENDEE":
			if stack[len(stack)-1] == "VEVENT" {
				q := findparam(s[i], "CN=")
				if q >= 0 {
					e.att = append(e.att,
						removefilter(
							s[i].params[q][3:len(s[i].params[q])], "\""))
				} else {
					e.att = append(e.att, removefilter(s[i].val, "mailto:"))
				}
			}
		case "COMMENT":
			if stack[len(stack)-1] == "VEVENT" {
				e.desc = s[i].val
			}
		case "DESCRIPTION":
			if stack[len(stack)-1] == "VEVENT" {
				e.desc = s[i].val
			}
		case "DTSTART":
			if stack[len(stack)-1] == "VEVENT" {
				q := findparam(s[i], "TZID=")
				if q >= 0 {
					e.time = tparse(s[i].val)
					e.zone = s[i].params[q][5:]
				} else {
					e.time = tparse(s[i].val)
				}
			}
		case "END":
			if stack[len(stack)-1] != s[i].val {
				die(s[i].val + " is not on top of the stack")
			}
			stack = stack[0 : len(stack)-1]
			if s[i].val == "VEVENT" {
				ev = append(ev, e)
				zeroev(&e)
			}
		case "LOCATION":
			if stack[len(stack)-1] == "VEVENT" {
				e.loc = s[i].val
			}
		case "ORGANIZER":
			if stack[len(stack)-1] == "VEVENT" {
				q := findparam(s[i], "CN=")
				if q >= 0 {
					e.org = removefilter(
						s[i].params[q][3:len(s[i].params[q])], "\"")
				} else {
					e.org = removefilter(s[i].val, "mailto:")
				}
			}
		case "SUMMARY":
			if stack[len(stack)-1] == "VEVENT" {
				e.smy = s[i].val
			}
		}
	}

	for j := 0; j < len(ev)-1; j++ {
		eprint(ev[j])
		fmt.Println()
	}
	eprint(ev[len(ev)-1])
}

func main() {
	if len(os.Args) < 2 {
		die("no input was passed.")
	}

	if os.Args[1] == "-" {
		//stdin
		t, e := io.ReadAll(os.Stdin)
		bad(e)
		f := string(t)
		icsp(f)
		die("")
	}

	for i := 1; i < len(os.Args); i++ {
		t, e := os.ReadFile(os.Args[i])
		bad(e)
		f := string(t)
		icsp(f)
	}
}

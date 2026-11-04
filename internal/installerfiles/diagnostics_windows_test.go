//go:build windows

package installerfiles

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

func diagnosticFixture(t *testing.T) (root, outside string) {
	t.Helper()
	base := t.TempDir()
	root = filepath.Join(base, "diagnostics é")
	outside = filepath.Join(base, "outside")
	for _, path := range []string{root, outside} {
		if err := os.Mkdir(path, 0700); err != nil {
			t.Fatal(err)
		}
	}
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		t.Fatal(err)
	}
	setPrivacyDescriptor(t, root, fmt.Sprintf("D:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)(A;OICI;0x1301bf;;;LS)(A;OICI;0x1200a9;;;%s)", user.User.Sid.String()))
	return
}

func diagnosticRecord(t *testing.T, body []byte) map[string]any {
	t.Helper()
	var got map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(body), &got); err != nil {
		t.Fatal("diagnostic body is not one JSON record", err)
	}
	want := map[string]bool{"time_utc": true, "event": true, "action": true, "stage": true, "result_code": true}
	if len(got) != len(want) {
		t.Fatal("diagnostic schema changed")
	}
	for name := range got {
		if !want[name] {
			t.Fatal("unexpected diagnostic field")
		}
	}
	if _, err := time.Parse(time.RFC3339Nano, got["time_utc"].(string)); err != nil {
		t.Fatal("diagnostic timestamp is not native UTC RFC3339", err)
	}
	return got
}

func TestDiagnosticAppendAndFixedSchema(t *testing.T) {
	root, _ := diagnosticFixture(t)
	for _, code := range []int32{0, -2147024891} {
		if err := Diagnostic(root, "installer_begin", "Install", "inspect service", code); err != nil {
			t.Fatal(err)
		}
	}
	body := privacyBytes(t, filepath.Join(root, "installer.jsonl"))
	lines := bytes.Split(bytes.TrimSpace(body), []byte{'\n'})
	if len(lines) != 2 {
		t.Fatal("append did not preserve both records")
	}
	for i, line := range lines {
		record := diagnosticRecord(t, line)
		if record["event"] != "installer_begin" || record["action"] != "Install" || record["stage"] != "inspect service" {
			t.Fatal("fixed labels were not retained")
		}
		if i == 1 && record["result_code"] != float64(-2147024891) {
			t.Fatal("signed numeric failure code not retained")
		}
	}
	t.Log("PASS: installer diagnostics append fixed-schema native timestamp and signed numeric result without private payloads")
}

func TestDiagnosticRotationPreservesBoundedPriorBytes(t *testing.T) {
	root, _ := diagnosticFixture(t)
	current := filepath.Join(root, "installer.jsonl")
	prior := bytes.Repeat([]byte("x"), (4<<20)-32)
	if err := os.WriteFile(current, prior, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(current+".previous", []byte("older"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := Diagnostic(root, "rollback_complete", "Install", "verify service health", 0); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(privacyBytes(t, current+".previous"), prior) {
		t.Fatal("rotation did not preserve exact bounded prior bytes")
	}
	record := diagnosticRecord(t, privacyBytes(t, current))
	if record["event"] != "rollback_complete" {
		t.Fatal("rotation lost current completion event")
	}
	t.Log("PASS: bounded previous/current rotation retains exact prior bytes before current log reset")
}

func TestDiagnosticUnsafeLeavesRefusedBeforeMutation(t *testing.T) {
	for _, name := range []string{"installer.jsonl", "installer.jsonl.previous"} {
		t.Run(name, func(t *testing.T) {
			root, outside := diagnosticFixture(t)
			current := filepath.Join(root, "installer.jsonl")
			prior := bytes.Repeat([]byte("p"), (4<<20)-32)
			if err := os.WriteFile(current, prior, 0600); err != nil {
				t.Fatal(err)
			}
			sentinel := filepath.Join(outside, "sentinel")
			if err := os.WriteFile(sentinel, []byte("outside untouched"), 0600); err != nil {
				t.Fatal(err)
			}
			hostile := filepath.Join(root, name)
			if name == "installer.jsonl" {
				if err := os.Remove(hostile); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.Link(sentinel, hostile); err != nil {
				t.Fatal(err)
			}
			if err := Diagnostic(root, "installer_complete", "Install", "start service", 0); err == nil {
				t.Fatal("hardlinked diagnostic leaf accepted")
			}
			assertBytes(t, sentinel, "outside untouched")
			if name != "installer.jsonl" && !bytes.Equal(privacyBytes(t, current), prior) {
				t.Fatal("unsafe rotation target changed current before refusal")
			}
		})
	}
	t.Log("PASS: real current/previous hard links refused before truncation; outside and earlier log bytes preserved")
}

func TestDiagnosticUnsafeActualPermissionsAndLabelsRefused(t *testing.T) {
	for _, name := range []string{"", "installer.jsonl", "installer.jsonl.previous"} {
		t.Run("privacy-"+name, func(t *testing.T) {
			root, _ := diagnosticFixture(t)
			current := filepath.Join(root, "installer.jsonl")
			prior := bytes.Repeat([]byte("p"), (4<<20)-32)
			if err := os.WriteFile(current, prior, 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(current+".previous", []byte("older"), 0600); err != nil {
				t.Fatal(err)
			}
			unsafe := root
			flags := "OICI"
			if name != "" {
				unsafe = filepath.Join(root, name)
				flags = ""
			}
			setPrivacyDescriptor(t, unsafe, fmt.Sprintf("D:P(A;%s;FA;;;SY)(A;%s;FA;;;BA)(A;%s;GR;;;WD)", flags, flags, flags))
			assertProtectedEveryoneRead(t, unsafe)
			if err := Diagnostic(root, "installer_complete", "Install", "start service", 0); err == nil {
				t.Fatal("unsafe actual diagnostic permissions accepted")
			}
			if !bytes.Equal(privacyBytes(t, current), prior) {
				t.Fatal("privacy refusal changed current bytes")
			}
			assertBytes(t, current+".previous", "older")
		})
	}
	root, _ := diagnosticFixture(t)
	for _, labels := range [][3]string{{"private-media-name", "Install", "inspect service"}, {"installer_begin", "private-path", "inspect service"}, {"installer_begin", "Install", `C:\private\secret`}} {
		if err := Diagnostic(root, labels[0], labels[1], labels[2], 0); err == nil {
			t.Fatal("unapproved diagnostic label accepted")
		}
	}
	if _, err := os.Stat(filepath.Join(root, "installer.jsonl")); !os.IsNotExist(err) {
		t.Fatal("rejected private labels created a log")
	}
	t.Log("PASS: actual root/current/previous privacy failures and nonallowlisted labels refuse before diagnostic content mutation")
}

func TestFailureNoteRefusesAliasAndUntrustedWriter(t *testing.T) {
	for _, kind := range []string{"hardlink", "writable"} {
		t.Run(kind, func(t *testing.T) {
			base := t.TempDir()
			app := filepath.Join(base, "application")
			if err := os.Mkdir(app, 0700); err != nil {
				t.Fatal(err)
			}
			setPrivacyDescriptor(t, app, "D:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)(A;OICI;0x1200a9;;;BU)")
			note := filepath.Join(app, "install-diagnostic.txt")
			sentinel := filepath.Join(base, "outside")
			if err := os.WriteFile(sentinel, []byte("outside untouched"), 0600); err != nil {
				t.Fatal(err)
			}
			if kind == "hardlink" {
				if err := os.Link(sentinel, note); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := os.WriteFile(note, []byte("prior note"), 0600); err != nil {
					t.Fatal(err)
				}
				setPrivacyDescriptor(t, note, "D:P(A;;FA;;;SY)(A;;FA;;;BA)(A;;GW;;;WD)")
			}
			if err := FailureNote(app, "initialize state", -1); err == nil {
				t.Fatal("unsafe failure note accepted")
			}
			assertBytes(t, sentinel, "outside untouched")
			if kind == "writable" {
				assertBytes(t, note, "prior note")
			}
		})
	}
	t.Log("PASS: actual failure-note alias and untrusted mutation grants refused without changing sentinel or note bytes")
}

func TestFailureNoteFixedContent(t *testing.T) {
	root, _ := diagnosticFixture(t)
	// A code directory may have unrelated read/execute grants, never mutation.
	setPrivacyDescriptor(t, root, "D:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)(A;OICI;0x1200a9;;;BU)")
	if err := FailureNote(root, "initialize state", -7); err != nil {
		t.Fatal(err)
	}
	note := privacyBytes(t, filepath.Join(root, "install-diagnostic.txt"))
	if !bytes.Contains(note, []byte("initialize state")) || !bytes.Contains(note, []byte("-7")) || len(note) > 1024 {
		t.Fatal("failure note lost bounded fixed label/code")
	}
	before := append([]byte(nil), note...)
	if err := FailureNote(root, `private-path-secret`, 0); err == nil {
		t.Fatal("private failure stage accepted")
	}
	if !bytes.Equal(before, privacyBytes(t, filepath.Join(root, "install-diagnostic.txt"))) {
		t.Fatal("private stage refusal overwrote note")
	}
	if strings.Contains(string(note), "private") {
		t.Fatal("unexpected private note payload")
	}
	t.Log("PASS: failure note contains only bounded fixed stage and numeric code; unknown stage cannot overwrite it")
}

func TestDiagnosticOversizeAndMissingRootRefuseWithoutFallback(t *testing.T) {
	root, _ := diagnosticFixture(t)
	current := filepath.Join(root, "installer.jsonl")
	prior := bytes.Repeat([]byte("z"), (4<<20)+1)
	if err := os.WriteFile(current, prior, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(current+".previous", []byte("older"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := Diagnostic(root, "installer_begin", "Install", "inspect service", 0); err == nil {
		t.Fatal("oversized current diagnostic accepted")
	}
	if !bytes.Equal(privacyBytes(t, current), prior) {
		t.Fatal("oversize refusal changed current")
	}
	assertBytes(t, current+".previous", "older")
	missing := filepath.Join(root, "missing")
	if err := Diagnostic(missing, "installer_begin", "Install", "inspect service", 0); err == nil {
		t.Fatal("missing diagnostic root accepted")
	}
	if _, err := os.Stat(missing); !os.IsNotExist(err) {
		t.Fatal("diagnostic operation created a missing root")
	}
	t.Log("PASS: oversized log and missing root refuse without truncation or unguarded fallback creation")
}

func TestFailureNoteUnsafeParentRefusesBeforeExistingNoteWrite(t *testing.T) {
	root, _ := diagnosticFixture(t)
	setPrivacyDescriptor(t, root, "D:P(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)(A;OICI;GW;;;WD)")
	note := filepath.Join(root, "install-diagnostic.txt")
	if err := os.WriteFile(note, []byte("prior note"), 0600); err != nil {
		t.Fatal(err)
	}
	setPrivacyDescriptor(t, note, "D:P(A;;FA;;;SY)(A;;FA;;;BA)")
	if err := FailureNote(root, "initialize state", -1); err == nil {
		t.Fatal("unsafe code parent accepted for safe leaf")
	}
	assertBytes(t, note, "prior note")
	t.Log("PASS: untrusted parent mutation grant refuses failure-note write even with a protected safe leaf")
}

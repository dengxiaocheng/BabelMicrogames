//go:build linux && cgo

package corehost_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"babel-runtime/internal/corehost"
)

func TestSharedLibraryFixtureSmoke(t *testing.T) {
	libraryPath := buildFixtureLibrary(t)

	transport, err := corehost.NewDlopenTransport(libraryPath)
	if err != nil {
		t.Fatalf("NewDlopenTransport returned error: %v", err)
	}
	if closer, ok := transport.(interface{ Close() }); ok {
		defer closer.Close()
	}

	for _, operation := range []string{"solo_step", "room_step"} {
		if err := corehost.VerifyFixtureTransport(context.Background(), transport, operation); err != nil {
			t.Fatalf("VerifyFixtureTransport(%s) returned error: %v", operation, err)
		}
	}
}

func TestVerifySceneHostLibraryCommand(t *testing.T) {
	libraryPath := buildFixtureLibrary(t)

	cmd := exec.Command("go", "run", "./cmd/babel-dev", "verify-scene-host-library", "--library", libraryPath, "--mode", "all")
	cmd.Dir = repoRoot(t)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("verify-scene-host-library failed: %v\n%s", err, output)
	}
	if string(output) != "solo_step verification OK\nroom_step verification OK\n" {
		t.Fatalf("unexpected command output:\n%s", output)
	}
}

func buildFixtureLibrary(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("cc"); err != nil {
		t.Skip("cc not available")
	}

	soloRequest, soloResponse, err := corehost.FixturePair("solo_step")
	if err != nil {
		t.Fatalf("FixturePair(solo_step) returned error: %v", err)
	}
	roomRequest, roomResponse, err := corehost.FixturePair("room_step")
	if err != nil {
		t.Fatalf("FixturePair(room_step) returned error: %v", err)
	}

	soloRequestJSON, err := json.Marshal(soloRequest)
	if err != nil {
		t.Fatalf("json.Marshal solo request returned error: %v", err)
	}
	roomRequestJSON, err := json.Marshal(roomRequest)
	if err != nil {
		t.Fatalf("json.Marshal room request returned error: %v", err)
	}
	soloResponseJSON, err := corehost.MarshalFixture(soloResponse)
	if err != nil {
		t.Fatalf("MarshalFixture solo response returned error: %v", err)
	}
	roomResponseJSON, err := corehost.MarshalFixture(roomResponse)
	if err != nil {
		t.Fatalf("MarshalFixture room response returned error: %v", err)
	}

	source := fmt.Sprintf(`#include <stdint.h>
#include <stdlib.h>
#include <string.h>

static const char* solo_request_match = %q;
static const char* room_request_match = %q;
static const char* solo_response = %q;
static const char* room_response = %q;

static int write_bytes(const char* src, uint8_t** out_bytes, size_t* out_len) {
	size_t len = strlen(src);
	uint8_t* buffer = (uint8_t*)malloc(len);
	if (buffer == NULL) {
		return 91;
	}
	memcpy(buffer, src, len);
	*out_bytes = buffer;
	*out_len = len;
	return 0;
}

int babel_sim_step(const uint8_t* request_bytes, size_t request_len, uint8_t** response_bytes, size_t* response_len) {
	char* request = (char*)malloc(request_len + 1);
	if (request == NULL) {
		return 92;
	}
	memcpy(request, request_bytes, request_len);
	request[request_len] = '\0';

	const char* response = NULL;
	if (strcmp(request, solo_request_match) == 0) {
		response = solo_response;
	} else if (strcmp(request, room_request_match) == 0) {
		response = room_response;
	}
	free(request);

	if (response == NULL) {
		return write_bytes("unknown fixture request", response_bytes, response_len) == 0 ? 2 : 93;
	}
	return write_bytes(response, response_bytes, response_len);
}

void babel_sim_free(void* ptr) {
	free(ptr);
}
`, string(soloRequestJSON), string(roomRequestJSON), string(soloResponseJSON), string(roomResponseJSON))

	tempDir := t.TempDir()
	sourcePath := filepath.Join(tempDir, "scene_core_stub.c")
	libraryPath := filepath.Join(tempDir, "libscene_core_stub.so")
	if err := os.WriteFile(sourcePath, []byte(source), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	compile := exec.Command("cc", "-shared", "-fPIC", "-O2", "-o", libraryPath, sourcePath)
	if output, err := compile.CombinedOutput(); err != nil {
		t.Fatalf("cc failed: %v\n%s", err, output)
	}
	return libraryPath
}

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned error: %v", err)
	}
	return filepath.Clean(filepath.Join(wd, "..", ".."))
}

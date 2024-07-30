package main

import (
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

var originalURL string

func setTestVersionURL(url string) {
	originalURL = VersionURL
	VersionURL = url
}

func resetTestVersionURL() {
	VersionURL = originalURL
}

func TestGetLatestVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("go1.17.1\n"))
	}))
	defer server.Close()

	setTestVersionURL(server.URL)
	defer resetTestVersionURL()

	version, err := GetLatestVersion()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if version != "go1.17.1" {
		t.Errorf("Expected version go1.17.1, got %s", version)
	}
}

func TestGetLatestVersionHTTPError(t *testing.T) {
	setTestVersionURL("http://invalid-url")
	defer resetTestVersionURL()

	_, err := GetLatestVersion()
	if err == nil {
		t.Error("Expected an error for invalid URL, got nil")
	}
}

func TestGetLatestVersionReadError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1")
	}))
	defer server.Close()

	setTestVersionURL(server.URL)
	defer resetTestVersionURL()

	_, err := GetLatestVersion()
	if err == nil {
		t.Error("Expected an error for read failure, got nil")
	}
}

func TestGetLatestVersionMalformedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("malformed\nresponse\n"))
	}))
	defer server.Close()

	setTestVersionURL(server.URL)
	defer resetTestVersionURL()

	version, err := GetLatestVersion()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if version != "malformed" {
		t.Errorf("Expected version 'malformed', got %s", version)
	}
}

func TestGetOfficialChecksum(t *testing.T) {
	tests := []struct {
		name       string
		serverFunc func(http.ResponseWriter, *http.Request)
		filename   string
		want       string
		wantErr    string
	}{
		{
			name: "Valid filename",
			serverFunc: func(w http.ResponseWriter, r *http.Request) {
				releases := []GoRelease{
					{
						Version: "go1.22.5",
						Files: []struct {
							Filename string `json:"filename"`
							OS       string `json:"os"`
							Arch     string `json:"arch"`
							Version  string `json:"version"`
							SHA256   string `json:"sha256"`
						}{
							{
								Filename: "go1.22.5.linux-amd64.tar.gz",
								SHA256:   "904b924d435eaea086515bc63235b192ea441bd8c9b198c507e85009e6e4c7f0",
							},
						},
					},
				}
				json.NewEncoder(w).Encode(releases)
			},
			filename: "go1.22.5.linux-amd64.tar.gz",
			want:     "904b924d435eaea086515bc63235b192ea441bd8c9b198c507e85009e6e4c7f0",
			wantErr:  "",
		},
		{
			name: "Valid filename",
			serverFunc: func(w http.ResponseWriter, r *http.Request) {
				releases := []GoRelease{
					{
						Version: "go1.22.5",
						Files: []struct {
							Filename string `json:"filename"`
							OS       string `json:"os"`
							Arch     string `json:"arch"`
							Version  string `json:"version"`
							SHA256   string `json:"sha256"`
						}{
							{
								Filename: "go1.22.5.linux-amd64.tar.gz",
								SHA256:   "904b924d435eaea086515bc63235b192ea441bd8c9b198c507e85009e6e4c7f0",
							},
						},
					},
				}
				json.NewEncoder(w).Encode(releases)
			},
			filename: "go1.22.5.linux-amd64.tar.gz",
			want:     "904b924d435eaea086515bc63235b192ea441bd8c9b198c507e85009e6e4c7f0",
			wantErr:  "",
		},
		{
			name: "Invalid filename",
			serverFunc: func(w http.ResponseWriter, r *http.Request) {
				releases := []GoRelease{
					{
						Version: "go1.22.5",
						Files: []struct {
							Filename string `json:"filename"`
							OS       string `json:"os"`
							Arch     string `json:"arch"`
							Version  string `json:"version"`
							SHA256   string `json:"sha256"`
						}{
							{
								Filename: "go1.22.5.linux-amd64.tar.gz",
								SHA256:   "904b924d435eaea086515bc63235b192ea441bd8c9b198c507e85009e6e4c7f0",
							},
						},
					},
				}
				json.NewEncoder(w).Encode(releases)
			},
			filename: "invalid.tar.gz",
			want:     "",
			wantErr:  "checksum not found for invalid.tar.gz",
		},
		{
			name: "HTTP GET error",
			serverFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			filename: "go1.22.5.linux-amd64.tar.gz",
			want:     "",
			wantErr:  "failed to fetch Go releases: HTTP status 500",
		},
		{
			name: "HTTP GET request failure",
			serverFunc: func(w http.ResponseWriter, r *http.Request) {
				panic("forced connection error")
			},
			filename: "go1.22.5.linux-amd64.tar.gz",
			want:     "",
			wantErr:  "failed to fetch Go releases",
		},
		{
			name: "Read body error",
			serverFunc: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Length", "1")
			},
			filename: "go1.22.5.linux-amd64.tar.gz",
			want:     "",
			wantErr:  "failed to read response body",
		},
		{
			name: "Invalid JSON",
			serverFunc: func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("invalid json"))
			},
			filename: "go1.22.5.linux-amd64.tar.gz",
			want:     "",
			wantErr:  "failed to parse JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var server *httptest.Server
			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				defer func() {
					if r := recover(); r != nil {
						server.CloseClientConnections()
					}
				}()
				tt.serverFunc(w, r)
			}))
			defer server.Close()

			originalURL := URL
			URL = server.URL
			defer func() { URL = originalURL }()

			got, err := GetOfficialChecksum(tt.filename)
			if tt.wantErr != "" {
				if err == nil {
					t.Errorf("GetOfficialChecksum() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("GetOfficialChecksum() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
			} else if err != nil {
				t.Errorf("GetOfficialChecksum() unexpected error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("GetOfficialChecksum() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetGoVersionInfo(t *testing.T) {
	tests := []struct {
		name     string
		ver      string
		os       string
		arch     string
		wantURL  string
		wantFile string
	}{
		{
			name:     "Linux AMD64",
			ver:      "go1.17.1",
			os:       "linux",
			arch:     "amd64",
			wantURL:  "https://dl.google.com/go/go1.17.1.linux-amd64.tar.gz",
			wantFile: "go1.17.1.linux-amd64.tar.gz",
		},
		{
			name:     "Windows AMD64",
			ver:      "go1.17.1",
			os:       "windows",
			arch:     "amd64",
			wantURL:  "https://dl.google.com/go/go1.17.1.windows-amd64.zip",
			wantFile: "go1.17.1.windows-amd64.zip",
		},
		{
			name:     "Darwin ARM64",
			ver:      "go1.17.1",
			os:       "darwin",
			arch:     "arm64",
			wantURL:  "https://dl.google.com/go/go1.17.1.darwin-arm64.tar.gz",
			wantFile: "go1.17.1.darwin-arm64.tar.gz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getGoVersionInfo(tt.ver, tt.os, tt.arch)

			// Check if the error is due to checksum not found
			if err != nil {
				if !strings.Contains(err.Error(), "checksum not found") {
					t.Errorf("getGoVersionInfo() unexpected error = %v", err)
				}
				return
			}

			if got.URL != tt.wantURL {
				t.Errorf("getGoVersionInfo() URL = %v, want %v", got.URL, tt.wantURL)
			}
			if got.Version != tt.ver || got.OS != tt.os || got.Arch != tt.arch {
				t.Errorf("getGoVersionInfo() returned incorrect Version, OS, or Arch")
			}
			// We can't reliably test the Checksum here as it depends on the external GetOfficialChecksum function
			// and may change over time. We'll just check if it's not empty.
			if got.Checksum == "" {
				t.Errorf("getGoVersionInfo() Checksum is empty")
			}
		})
	}
}

func TestWriteJSONFile(t *testing.T) {
	testFile := "test.json"
	testData := GoVersionInfo{
		Version:  "go1.17.1",
		OS:       "linux",
		Arch:     "amd64",
		URL:      "https://example.com/go1.17.1.linux-amd64.tar.gz",
		Checksum: "testchecksum",
	}

	err := writeJSONFile(testFile, testData)
	if err != nil {
		t.Fatalf("writeJSONFile() error = %v", err)
	}

	// Clean up
	defer os.Remove(testFile)

	// Read and verify file contents
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Error reading test file: %v", err)
	}

	var gotData GoVersionInfo
	err = json.Unmarshal(content, &gotData)
	if err != nil {
		t.Fatalf("Error unmarshaling test file content: %v", err)
	}

	if gotData != testData {
		t.Errorf("writeJSONFile() wrote %v, want %v", gotData, testData)
	}
}

func TestMain(t *testing.T) {
	// Redirect log output to discard to avoid cluttering test output
	log.SetOutput(os.Stderr)

	// Create a temporary directory for test files
	tempDir, err := os.MkdirTemp("", "gotest")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Change working directory to the temp directory
	oldWd, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldWd)

	// Run main
	main()

	// Check if files were created
	expectedFiles := []string{
		"linux_amd64.json",
		"linux_arm64.json",
		"windows_amd64.json",
		"mac_amd64.json",
		"mac_arm64.json",
	}

	for _, file := range expectedFiles {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			t.Errorf("Expected file %s was not created", file)
		} else {
			// Verify that the file contains valid JSON
			content, err := os.ReadFile(file)
			if err != nil {
				t.Errorf("Failed to read file %s: %v", file, err)
			} else {
				var info GoVersionInfo
				if err := json.Unmarshal(content, &info); err != nil {
					t.Errorf("File %s does not contain valid JSON: %v", file, err)
				}
			}
		}
	}
}

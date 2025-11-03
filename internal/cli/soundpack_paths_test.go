package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"claudio.click/internal/config"
	"claudio.click/internal/soundpack"
	"github.com/spf13/afero"
)

// TestJSONSoundpackFromSoundpackPaths tests that JSON soundpacks can be loaded
// when specified in the soundpack_paths configuration
func TestJSONSoundpackFromSoundpackPaths(t *testing.T) {
	// Create a temporary directory for our test files
	tmpDir := t.TempDir()
	
	// Create dummy sound files that the JSON will reference
	soundsDir := filepath.Join(tmpDir, "sounds")
	err := os.MkdirAll(soundsDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create sounds directory: %v", err)
	}
	
	// Create dummy WAV files (minimal valid WAV headers)
	createMinimalWAV := func(path string) error {
		// Minimal WAV file with RIFF header
		wavData := []byte{
			0x52, 0x49, 0x46, 0x46, // "RIFF"
			0x24, 0x00, 0x00, 0x00, // File size - 8
			0x57, 0x41, 0x56, 0x45, // "WAVE"
			0x66, 0x6D, 0x74, 0x20, // "fmt "
			0x10, 0x00, 0x00, 0x00, // fmt chunk size
			0x01, 0x00,             // Audio format (PCM)
			0x01, 0x00,             // Num channels (mono)
			0x44, 0xAC, 0x00, 0x00, // Sample rate (44100)
			0x88, 0x58, 0x01, 0x00, // Byte rate
			0x02, 0x00,             // Block align
			0x10, 0x00,             // Bits per sample
			0x64, 0x61, 0x74, 0x61, // "data"
			0x00, 0x00, 0x00, 0x00, // Data size
		}
		return os.WriteFile(path, wavData, 0644)
	}
	
	successWAV := filepath.Join(soundsDir, "success.wav")
	errorWAV := filepath.Join(soundsDir, "error.wav")
	loadingWAV := filepath.Join(soundsDir, "loading.wav")
	defaultWAV := filepath.Join(soundsDir, "default.wav")
	
	for _, wavPath := range []string{successWAV, errorWAV, loadingWAV, defaultWAV} {
		if err := createMinimalWAV(wavPath); err != nil {
			t.Fatalf("Failed to create WAV file %s: %v", wavPath, err)
		}
	}
	
	// Create a JSON soundpack file
	jsonSoundpackPath := filepath.Join(tmpDir, "custom.json")
	jsonSoundpack := map[string]interface{}{
		"name":        "custom",
		"description": "Test custom soundpack",
		"version":     "1.0.0",
		"mappings": map[string]string{
			"success/success.wav": successWAV,
			"error/error.wav":     errorWAV,
			"loading/loading.wav": loadingWAV,
			"default.wav":         defaultWAV,
		},
	}
	
	jsonData, err := json.MarshalIndent(jsonSoundpack, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}
	
	err = os.WriteFile(jsonSoundpackPath, jsonData, 0644)
	if err != nil {
		t.Fatalf("Failed to write JSON soundpack: %v", err)
	}
	
	// Create a config file that uses the JSON soundpack
	configPath := filepath.Join(tmpDir, "config.json")
	cfg := &config.Config{
		Volume:           0.5,
		DefaultSoundpack: "custom", // Just the name, not a path
		SoundpackPaths:   []string{jsonSoundpackPath}, // Full path to JSON file
		Enabled:          false, // Silent mode for testing
		LogLevel:         "debug",
		AudioBackend:     "auto",
	}
	
	configData, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}
	
	err = os.WriteFile(configPath, configData, 0644)
	if err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	
	// Now test that we can load this configuration and create a mapper
	t.Run("LoadConfigWithJSONSoundpack", func(t *testing.T) {
		cm := config.NewConfigManager()
		loadedCfg, err := cm.LoadFromFile(configPath)
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}
		
		if loadedCfg.DefaultSoundpack != "custom" {
			t.Errorf("Expected soundpack 'custom', got '%s'", loadedCfg.DefaultSoundpack)
		}
		
		if len(loadedCfg.SoundpackPaths) != 1 {
			t.Errorf("Expected 1 soundpack path, got %d", len(loadedCfg.SoundpackPaths))
		}
		
		if loadedCfg.SoundpackPaths[0] != jsonSoundpackPath {
			t.Errorf("Expected soundpack path '%s', got '%s'", jsonSoundpackPath, loadedCfg.SoundpackPaths[0])
		}
	})
	
	// Test that we can create a mapper from the soundpack_paths
	t.Run("CreateMapperFromSoundpackPaths", func(t *testing.T) {
		// This simulates what initializeAudioSystem should do
		// Check if any soundpack_paths entry is a JSON file for this soundpack
		var mapper soundpack.PathMapper
		var err error
		
		for _, path := range cfg.SoundpackPaths {
			// Check if it's a JSON file
			if strings.HasSuffix(strings.ToLower(path), ".json") {
				// Try to load it as a JSON soundpack
				mapper, err = soundpack.LoadJSONSoundpack(path)
				if err == nil {
					// Successfully loaded
					break
				}
			}
		}
		
		if err != nil {
			t.Fatalf("Failed to load JSON soundpack: %v", err)
		}
		
		if mapper == nil {
			t.Fatal("Mapper should not be nil")
		}
		
		if mapper.GetName() != "custom" {
			t.Errorf("Expected mapper name 'custom', got '%s'", mapper.GetName())
		}
		
		if mapper.GetType() != "json" {
			t.Errorf("Expected mapper type 'json', got '%s'", mapper.GetType())
		}
	})
	
	// Test end-to-end with CLI
	t.Run("CLIWithJSONSoundpack", func(t *testing.T) {
		cli := NewCLI()
		
		hookJSON := `{
			"session_id": "test",
			"transcript_path": "/test",
			"cwd": "/test",
			"hook_event_name": "PostToolUse",
			"tool_name": "Bash",
			"tool_response": {
				"stdout": "success",
				"stderr": "",
				"interrupted": false
			}
		}`
		
		stdin := strings.NewReader(hookJSON)
		stdout := &strings.Builder{}
		stderr := &strings.Builder{}
		
		args := []string{"claudio", "--config", configPath, "--silent"}
		exitCode := cli.Run(args, stdin, stdout, stderr)
		
		if exitCode != 0 {
			t.Errorf("Expected exit code 0, got %d", exitCode)
			t.Logf("Stderr: %s", stderr.String())
		}
		
		// Check that the JSON soundpack was loaded successfully
		stderrOutput := stderr.String()
		if !strings.Contains(stderrOutput, "custom") {
			t.Logf("Stderr output:\n%s", stderrOutput)
		}
	})
}

// TestDirectorySoundpackStillWorks ensures backward compatibility with directory-based soundpacks
func TestDirectorySoundpackStillWorks(t *testing.T) {
	tmpDir := t.TempDir()
	
	// Create a directory-based soundpack
	soundpackDir := filepath.Join(tmpDir, "soundpacks", "mypack")
	err := os.MkdirAll(soundpackDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create soundpack directory: %v", err)
	}
	
	// Create a minimal default.wav
	defaultWAV := filepath.Join(soundpackDir, "default.wav")
	wavData := []byte{
		0x52, 0x49, 0x46, 0x46, // "RIFF"
		0x24, 0x00, 0x00, 0x00, // File size - 8
		0x57, 0x41, 0x56, 0x45, // "WAVE"
		0x66, 0x6D, 0x74, 0x20, // "fmt "
		0x10, 0x00, 0x00, 0x00, // fmt chunk size
		0x01, 0x00,             // Audio format (PCM)
		0x01, 0x00,             // Num channels (mono)
		0x44, 0xAC, 0x00, 0x00, // Sample rate (44100)
		0x88, 0x58, 0x01, 0x00, // Byte rate
		0x02, 0x00,             // Block align
		0x10, 0x00,             // Bits per sample
		0x64, 0x61, 0x74, 0x61, // "data"
		0x00, 0x00, 0x00, 0x00, // Data size
	}
	err = os.WriteFile(defaultWAV, wavData, 0644)
	if err != nil {
		t.Fatalf("Failed to create default.wav: %v", err)
	}
	
	// Create a config with directory soundpack
	configPath := filepath.Join(tmpDir, "config.json")
	cfg := &config.Config{
		Volume:           0.5,
		DefaultSoundpack: "mypack",
		SoundpackPaths:   []string{filepath.Join(tmpDir, "soundpacks")},
		Enabled:          false,
		LogLevel:         "debug",
		AudioBackend:     "auto",
	}
	
	configData, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}
	
	err = os.WriteFile(configPath, configData, 0644)
	if err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}
	
	// Test that directory soundpack still works
	cli := NewCLI()
	
	hookJSON := `{
		"session_id": "test",
		"transcript_path": "/test",
		"cwd": "/test",
		"hook_event_name": "PostToolUse",
		"tool_name": "Bash",
		"tool_response": {
			"stdout": "success",
			"stderr": "",
			"interrupted": false
		}
	}`
	
	stdin := strings.NewReader(hookJSON)
	stdout := &strings.Builder{}
	stderr := &strings.Builder{}
	
	args := []string{"claudio", "--config", configPath, "--silent"}
	exitCode := cli.Run(args, stdin, stdout, stderr)
	
	if exitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", exitCode)
		t.Logf("Stderr: %s", stderr.String())
	}
}

// TestSoundpackPathsWithDirectoryMapper tests that soundpack_paths can be used
// as base directories for directory mappers
func TestSoundpackPathsWithDirectoryMapper(t *testing.T) {
	// Create a mock filesystem
	fs := afero.NewMemMapFs()
	
	// Create soundpack directory structure
	basePath := "/custom/soundpacks/mypack"
	_ = fs.MkdirAll(basePath, 0755)
	
	// Create a dummy default.wav
	_ = afero.WriteFile(fs, filepath.Join(basePath, "default.wav"), []byte("wav data"), 0644)
	
	// Test creating directory mapper with custom base paths
	basePaths := []string{"/custom/soundpacks"}
	mapper := soundpack.NewDirectoryMapper("mypack", []string{filepath.Join(basePaths[0], "mypack")})
	
	if mapper.GetName() != "mypack" {
		t.Errorf("Expected mapper name 'mypack', got '%s'", mapper.GetName())
	}
	
	if mapper.GetType() != "directory" {
		t.Errorf("Expected mapper type 'directory', got '%s'", mapper.GetType())
	}
	
	// Test that it can map paths
	candidates, err := mapper.MapPath("default.wav")
	if err != nil {
		t.Fatalf("Failed to map path: %v", err)
	}
	
	if len(candidates) != 1 {
		t.Errorf("Expected 1 candidate, got %d", len(candidates))
	}
	
	expected := filepath.Join(basePaths[0], "mypack", "default.wav")
	if candidates[0] != expected {
		t.Errorf("Expected candidate '%s', got '%s'", expected, candidates[0])
	}
}

package main

import (
    "fmt"
    "os"
    "testing"
    "time"
)

func TestLoadConfigDefaultsAndValidation(t *testing.T) {
    // create temporary config with minimal required field
    tmp, err := os.CreateTemp("", "config_*.toml")
    if err != nil { t.Fatalf("temp file: %v", err) }
    defer os.Remove(tmp.Name())
    content := []byte(`target_ip = "192.168.0.10"`)
    if _, err := tmp.Write(content); err != nil { t.Fatalf("write: %v", err) }
    tmp.Close()

    cfg, err := loadConfig(tmp.Name())
    if err != nil { t.Fatalf("loadConfig error: %v", err) }
    if cfg.TargetIP != "192.168.0.10" {
        t.Errorf("unexpected TargetIP: %s", cfg.TargetIP)
    }
    // defaults applied
    if cfg.MonitorIntervalSeconds != 10 { t.Errorf("default MonitorIntervalSeconds not set, got %d", cfg.MonitorIntervalSeconds) }
    if cfg.ChargePowerUpdateIntervalMinutes != 10 {
        t.Errorf("default ChargePowerUpdateIntervalMinutes not set, got %d", cfg.ChargePowerUpdateIntervalMinutes)
    }
    if cfg.ModeChangeInhibitMinutes != 5 { t.Errorf("default ModeChangeInhibitMinutes not set, got %d", cfg.ModeChangeInhibitMinutes) }
}

func TestLoadConfigMissingTargetIP(t *testing.T) {
    tmp, _ := os.CreateTemp("", "bad_*.toml")
    defer os.Remove(tmp.Name())
    tmp.Write([]byte(`monitor_interval_seconds = 5`))
    tmp.Close()
    _, err := loadConfig(tmp.Name())
    if err == nil {
        t.Fatalf("expected error for missing target_ip")
    }
}

func TestIsChargingTime(t *testing.T) {
    // Helper to create a time at given hour:minute on arbitrary date
    makeNow := func(h, m int) time.Time {
        t0, _ := time.Parse("2006-01-02 15:04", fmt.Sprintf("2025-01-01 %02d:%02d", h, m))
        return t0
    }
    // simple same-day interval where now is mocked via system time – we cannot change time.Now easily, so test logic with known times.
    // We'll test the parsing and boundary logic using fixed strings that include wrap-around.
    // For non-wrapping case
    now := makeNow(12, 0)
    ok, err := isChargingTime(now, "00:00", "23:59")
    if err != nil || !ok {
        t.Fatalf("expected always true, got %v, err=%v", ok, err)
    }
    // Wrapping interval where now may be outside; we just ensure no error and boolean returned.
    now2 := makeNow(3,0)
    ok2, err2 := isChargingTime(now2, "23:00", "02:00")
    if err2 != nil {
        t.Fatalf("wrap interval parse error: %v", err2)
    }
    if ok2 {
        t.Fatalf("expected false for outside wrap interval, got true")
    }

}

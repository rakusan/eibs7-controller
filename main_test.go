package main

import (
    "os"
    "testing"
    "time"
)

func TestLoadConfigDefaults(t *testing.T) {
    // create temporary config with only required field
    tmp, err := os.CreateTemp("", "config_*.toml")
    if err != nil { t.Fatalf("temp file: %v", err) }
    defer os.Remove(tmp.Name())
    _, err = tmp.WriteString("target_ip = \"192.168.0.10\"\n")
    if err != nil { t.Fatalf("write: %v", err) }
    tmp.Close()

    cfg, err := loadConfig(tmp.Name())
    if err != nil { t.Fatalf("loadConfig error: %v", err) }
    if cfg.TargetIP != "192.168.0.10" {
        t.Errorf("TargetIP mismatch: got %s", cfg.TargetIP)
    }
    // defaults
    if cfg.MonitorIntervalSeconds != 10 { t.Errorf("default MonitorIntervalSeconds not set") }
    if cfg.ChargePowerUpdateIntervalMinutes != 10 { t.Errorf("default ChargePowerUpdateIntervalMinutes not set") }
    if cfg.ModeChangeInhibitMinutes != 5 { t.Errorf("default ModeChangeInhibitMinutes not set") }
    if cfg.MinSurplusPowerJudgmentMinutes != 5 { t.Errorf("default MinSurplusPowerJudgmentMinutes not set") }
    if cfg.SurplusPowerMarginWatts != 500 { t.Errorf("default SurplusPowerMarginWatts not set") }
    if cfg.MaxChargePowerWatts != 3000 { t.Errorf("default MaxChargePowerWatts not set") }
}

func TestIsChargingTime(t *testing.T) {
    // simple within same day
    ok, err := isChargingTime("09:00", "17:00", time.Now())
    if err != nil { t.Fatalf("error: %v", err) }
    // depends on current time; just ensure no error and bool returned
    _ = ok

    // crossing midnight
    ok2, err := isChargingTime("23:00", "02:00", time.Now())
    if err != nil { t.Fatalf("error crossing: %v", err) }
    _ = ok2
}

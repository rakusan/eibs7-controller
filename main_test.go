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
    // simple within same day - expect true between 09:00 and 17:00 at 12:00
    fixedNow := time.Date(2025, time.September, 15, 12, 0, 0, 0, time.UTC)
    ok, err := isChargingTime("09:00", "17:00", fixedNow)
    if err != nil { t.Fatalf("error: %v", err) }
    if !ok {
        t.Errorf("expected charging time within range to be true")
    }

    // crossing midnight - expect true at 01:30 between 23:00 and 02:00
    fixedNow2 := time.Date(2025, time.September, 15, 1, 30, 0, 0, time.UTC)
    ok2, err := isChargingTime("23:00", "02:00", fixedNow2)
    if err != nil { t.Fatalf("error crossing: %v", err) }
    if !ok2 {
        t.Errorf("expected charging time across midnight to be true")
    }
}

func TestIsChargingTimeFalse(t *testing.T) {
    // time outside range should return false
    fixedNow := time.Date(2025, time.September, 15, 8, 0, 0, 0, time.UTC)
    ok, err := isChargingTime("09:00", "17:00", fixedNow)
    if err != nil { t.Fatalf("error: %v", err) }
    if ok {
        t.Errorf("expected charging time outside range to be false")
    }

    // crossing midnight false case
    fixedNow2 := time.Date(2025, time.September, 15, 3, 0, 0, 0, time.UTC)
    ok2, err := isChargingTime("23:00", "02:00", fixedNow2)
    if err != nil { t.Fatalf("error crossing false: %v", err) }
    if ok2 {
        t.Errorf("expected charging time outside midnight range to be false")
    }
}


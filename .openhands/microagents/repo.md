# Repository Purpose

This repository provides an unofficial Go implementation of a controller for the EIBS7 solar‑power and storage system using the ECHONET Lite protocol. It monitors device status, calculates surplus power, and automatically manages battery charging to optimise self‑consumption.

# Setup Instructions

1. Install Go (>=1.20) on your machine.
2. Clone the repository and navigate into `eibs7-controller`.
3. Edit `config.toml` to set the target IP address and desired parameters.
4. Run the application:
   ```
   go run main.go
   ```
   or build a binary:
   ```
   go build -o eibs7-controller main.go
   ```
5. Ensure network access to the EIBS7 device (UDP port 3610).

# Repository Structure

- `main.go` – entry point, initialises configuration and starts monitoring.
- `config.toml` – default configuration file with IP address, intervals, thresholds, etc.
- `docs/README.md` – detailed software requirements specification.
- `imgs/` – screenshots used in the README.
- `echonetlite/` – implementation of ECHONET Lite communication handling.
- `go.mod`, `go.sum` – Go module definitions.

# Development Guidelines

- If you modify a code and there are comments corresponding to the code modified, please also modify the comments accordingly.
- Any usage of `apt-get` for installing packages is prefixed with `sudo`. It can be used in scripts or commands where elevated privileges are required. Example: `sudo apt-get update && sudo apt-get install -y <package>`

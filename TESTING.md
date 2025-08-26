# Testing in EIBS7 Controller

This document explains the unit tests added to the EIBS7 controller project.

## Overview

The EIBS7 controller is a Go application that controls ECHONET Lite devices (specifically EIBS7 battery systems). The application includes several key functions that benefit from unit testing:

1. `isChargingTime` - Determines if the current time falls within charging hours
2. `getPropertyName` - Maps ECHONET property codes to human-readable names  
3. `decodeEDT` - Decodes ECHONET property value data into appropriate Go types

## Test Coverage

### isChargingTime Tests
- Normal time range (09:00 - 15:00)
- Time range that crosses midnight (23:00 - 02:00)
- Invalid time format handling
- Edge cases (exactly at start/end times)

### getPropertyName Tests
- Battery object properties (E4, DA, EB, D3, A0)
- Solar panel object properties (E0)
- Meter object properties (C6)
- PCS object properties (E7)
- Unknown property handling

### decodeEDT Tests
- Battery object properties with correct byte lengths (E4, DA, EB, D3, A0)
- Solar panel object properties (E0) 
- Meter object properties (C6)
- PCS object properties (E7)
- Error handling for invalid PDC values
- Unknown property handling

## Running Tests

To run the tests in a Go environment:

```bash
go test -v
```

The tests require the `github.com/stretchr/testify` dependency, which has been added to go.mod.

## Test Structure

Tests are organized using Go's standard testing framework with:
- Table-driven tests for various scenarios
- Error condition testing
- Edge case coverage
- Proper assertion using testify library

## Dependencies Added

The project now includes the `github.com/stretchr/testify` dependency for cleaner test assertions.
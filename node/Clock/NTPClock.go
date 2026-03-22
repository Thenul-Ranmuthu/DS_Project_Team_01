package clock

import (
    "log"
    "sync"
    "time"

    "github.com/beevik/ntp"
)

type NTPClock struct {
    mu     sync.RWMutex
    offset time.Duration // difference between NTP time and local system time
}

var NTP = &NTPClock{}

func (c *NTPClock) Sync(server string) error {
    response, err := ntp.Query(server)
    if err != nil {
        return err
    }
    c.mu.Lock()
    c.offset = response.ClockOffset
    c.mu.Unlock()
    log.Printf("[NTPClock] Synced with %s | offset: %v", server, response.ClockOffset)
    return nil
}

// Returns the current NTP-corrected time.
func (c *NTPClock) Now() time.Time {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return NTP.Now().Add(c.offset)
}

// Offset returns the raw offset for inspection.
func (c *NTPClock) Offset() time.Duration {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.offset
}
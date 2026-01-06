package probe

import (
	"sync"

	"github.com/threefoldtech/deployment-checker/pkg/models"
)

// Collector is a thread-safe collector for deployment attempts
type Collector struct {
	mu       sync.Mutex
	attempts []models.Attempt
}

// NewCollector creates a new attempt collector
func NewCollector() *Collector {
	return &Collector{
		attempts: make([]models.Attempt, 0),
	}
}

// Add adds an attempt to the collector (thread-safe)
func (c *Collector) Add(attempt models.Attempt) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.attempts = append(c.attempts, attempt)
}

// GetAll returns a copy of all collected attempts (thread-safe)
func (c *Collector) GetAll() []models.Attempt {
	c.mu.Lock()
	defer c.mu.Unlock()
	result := make([]models.Attempt, len(c.attempts))
	copy(result, c.attempts)
	return result
}

// Reset clears all collected attempts
func (c *Collector) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.attempts = c.attempts[:0]
}

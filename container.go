package quark

import (
	"fmt"
	"sync"
)

// ServiceFactory is a function that creates a service instance.
type ServiceFactory func(*Container) (interface{}, error)

// Container is a simple dependency injection container with generics support.
type Container struct {
	factories map[string]ServiceFactory
	instances map[string]interface{}
	mu        sync.RWMutex
}

// NewContainer creates a new DI container.
func NewContainer() *Container {
	return &Container{
		factories: make(map[string]ServiceFactory),
		instances: make(map[string]interface{}),
	}
}

// Register registers a service factory under the given name.
// The factory will be called lazily when the service is first requested.
func (c *Container) Register(name string, factory ServiceFactory) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.factories[name] = factory
}

// RegisterInstance registers a pre-created instance.
func (c *Container) RegisterInstance(name string, instance interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.instances[name] = instance
}

// Get retrieves a service by name.
// If the service hasn't been instantiated yet, the factory is called.
// Instances are cached (singleton behavior).
func (c *Container) Get(name string) (interface{}, error) {
	// Check if already instantiated
	c.mu.RLock()
	if instance, ok := c.instances[name]; ok {
		c.mu.RUnlock()
		return instance, nil
	}
	c.mu.RUnlock()

	// Check if factory exists
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if instance, ok := c.instances[name]; ok {
		return instance, nil
	}

	factory, ok := c.factories[name]
	if !ok {
		return nil, fmt.Errorf("service not found: %s", name)
	}

	// Create instance
	instance, err := factory(c)
	if err != nil {
		return nil, fmt.Errorf("failed to create service %s: %w", name, err)
	}

	// Cache the instance
	c.instances[name] = instance

	return instance, nil
}

// MustGet retrieves a service by name or panics if not found.
func (c *Container) MustGet(name string) interface{} {
	instance, err := c.Get(name)
	if err != nil {
		panic(err)
	}
	return instance
}

// Has checks if a service is registered.
func (c *Container) Has(name string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if _, ok := c.instances[name]; ok {
		return true
	}
	_, ok := c.factories[name]
	return ok
}

// Reset clears all instances but keeps factories.
// Useful for testing.
func (c *Container) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.instances = make(map[string]interface{})
}

// Clear removes all factories and instances.
func (c *Container) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.factories = make(map[string]ServiceFactory)
	c.instances = make(map[string]interface{})
}

// Provide registers a typed service factory.
// This is the generic version of Register.
func Provide[T any](c *Container, name string, factory func(*Container) (T, error)) {
	c.Register(name, func(cont *Container) (interface{}, error) {
		return factory(cont)
	})
}

// ProvideValue registers a pre-created typed instance.
func ProvideValue[T any](c *Container, name string, value T) {
	c.RegisterInstance(name, value)
}

// Resolve retrieves a typed service from the container.
func Resolve[T any](c *Container, name string) (T, error) {
	var zero T
	instance, err := c.Get(name)
	if err != nil {
		return zero, err
	}

	typed, ok := instance.(T)
	if !ok {
		return zero, fmt.Errorf("service %s is not of expected type", name)
	}

	return typed, nil
}

// MustResolve retrieves a typed service or panics.
func MustResolve[T any](c *Container, name string) T {
	result, err := Resolve[T](c, name)
	if err != nil {
		panic(err)
	}
	return result
}

// ServiceProvider is an interface for service providers.
// Service providers encapsulate service registration logic.
type ServiceProvider interface {
	// Register registers services in the container.
	Register(*Container) error
	// Boot is called after all providers are registered.
	// Use this for setup that depends on other services.
	Boot(*Container) error
}

// RegisterProviders registers multiple service providers.
func (c *Container) RegisterProviders(providers ...ServiceProvider) error {
	// First, register all providers
	for _, p := range providers {
		if err := p.Register(c); err != nil {
			return fmt.Errorf("provider registration failed: %w", err)
		}
	}

	// Then, boot all providers
	for _, p := range providers {
		if err := p.Boot(c); err != nil {
			return fmt.Errorf("provider boot failed: %w", err)
		}
	}

	return nil
}

// BaseProvider provides a default implementation of ServiceProvider.
type BaseProvider struct{}

// Register is a no-op implementation.
func (p *BaseProvider) Register(c *Container) error {
	return nil
}

// Boot is a no-op implementation.
func (p *BaseProvider) Boot(c *Container) error {
	return nil
}

// Alias creates an alias from one service name to another.
func (c *Container) Alias(alias, target string) {
	c.Register(alias, func(cont *Container) (interface{}, error) {
		return cont.Get(target)
	})
}

// Keys returns all registered service names.
func (c *Container) Keys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	seen := make(map[string]bool)
	for name := range c.factories {
		seen[name] = true
	}
	for name := range c.instances {
		seen[name] = true
	}

	keys := make([]string, 0, len(seen))
	for name := range seen {
		keys = append(keys, name)
	}
	return keys
}
